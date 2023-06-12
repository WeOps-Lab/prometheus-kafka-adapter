package main

import (
	"encoding/json"
	"fmt"
	"github.com/prometheus/prometheus/prompb"
	"math"
	"strconv"
	"strings"
	"time"
)

// handleSpecialValue 处理+Inf、-Inf、NaN特殊值
func handleSpecialValue(value float64) float64 {
	if math.IsInf(value, -1) || math.IsNaN(value) {
		return float64(0)
	}
	if math.IsInf(value, 1) {
		return float64(-1)
	}
	return value
}

// formatMetricsData 标准化输出数据
func formatMetricsData(metricName string, dimensions map[string]interface{}, sample prompb.Sample) (data []byte, err error) {
	var timestamp int64
	timestamp = time.Unix(sample.Timestamp/1000, 0).UTC().UnixNano() / int64(time.Millisecond)
	handleData := MetricsData{
		Data: []struct {
			Dimension map[string]interface{} `json:"dimension"`
			Metrics   map[string]float64     `json:"metrics"`
			Timestamp int64                  `json:"timestamp"`
		}{
			{
				Dimension: dimensions,
				Metrics: map[string]float64{
					metricName: handleSpecialValue(sample.Value),
				},
				Timestamp: timestamp,
			},
		},
	}

	return json.Marshal(handleData)
}

// k8sMetricsPreHandler 判断k8s指标，并补充k8s类的bk_obj_id
func k8sMetricsPreHandler(labels map[string]string) (exist bool) {
	if _, ok := labels["node"]; !ok {
		return false
	}
	if _, ok := labels["cluster"]; !ok {
		return false
	}
	if nodeMetricName, nodeMetricsExist := K8sNodeMetrics[labels["__name__"]]; nodeMetricsExist {
		labels["__name__"] = nodeMetricName
		labels["bk_obj_id"] = K8sNodeObjectId
		labels["instance_name"] = labels["node"]
		labels["cluster_name"] = labels["cluster"]
		return true
	} else if podMetricName, podMetricsExist := K8sPodMetrics[labels["__name__"]]; podMetricsExist {
		if _, ok := labels["pod"]; !ok {
			return false
		}
		labels["__name__"] = podMetricName
		labels["bk_obj_id"] = K8sPodObjectId
		labels["instance_name"] = labels["uid"]
		labels["cluster_name"] = labels["cluster"]
		return true
	} else {
		return false
	}
}

// fillUpBkInfo 补充蓝鲸指标信息
func fillUpBkInfo(labels map[string]string) (dimensions map[string]interface{}) {
	// 先填入所有维度信息
	dimensions = make(map[string]interface{})
	for key, value := range labels {
		dimensions[key] = value
	}

	var (
		bkInstId   int
		bkDataId   string
		protocol   = dimensions[Protocol].(string)
		bkObjectId = dimensions["bk_obj_id"].(string)
	)

	if !objList[bkObjectId] {
		objList[bkObjectId] = true
	}

	if val, ok := dimensions["bk_data_id"]; !ok || val == nil {
		if bkDataId = getDataId(bkObjectId); bkDataId == "" {
			return nil
		}
		dimensions["bk_data_id"] = bkDataId
	}

	// 第二层对node、pod分别处理
	if bkObjectId == K8sPodObjectId {
		if bkInstId = getBkInstId(K8sPodObjectId, dimensions["instance_name"].(string)); bkInstId == 0 {
			return nil
		}
		dimensions["pod_id"] = bkInstId
		dimensions["bk_inst_id"] = bkInstId

		if clusterId := getBkInstId(K8sClusterObjectId, dimensions["cluster_name"].(string)); clusterId == 0 {
			return nil
		} else {
			dimensions["cluster"] = clusterId
		}

		podWorkloadInfo, found := bkObjRelaCache.Get("pod_workload_rel_map")
		if found {
			if dimensions["workload"] = podWorkloadInfo.(map[int]int)[bkInstId]; dimensions["workload"].(int) == 0 {
				return nil
			}
		} else {
			return nil
		}

		dimensions["node_id"] = getBkInstId(K8sNodeObjectId, dimensions["node"].(string))
		dimensions["namespace_id"] = getBkInstId(K8sNameSpaceObjectId, fmt.Sprintf("%v (%v)", dimensions["namespace"].(string), dimensions["cluster_name"].(string)))
		namespaceId, ok := dimensions["namespace_id"].(int)
		if !ok || namespaceId == 0 {
			return nil
		}

		if bizInfo, bizFound := bkSetBizCache.Get(fmt.Sprintf("%v_set_id_biz_id", K8sNameSpaceObjectId)); bizFound {
			if bizId, bizIdFound := bizInfo.(map[int]int)[namespaceId]; bizIdFound {
				dimensions["bk_biz_id"] = bizId
			}
		}
		deleteUselessDimension(&dimensions, k8sPodDimension, true)
	} else if bkObjectId == K8sNodeObjectId {
		if clusterId := getBkInstId(K8sClusterObjectId, dimensions["cluster"].(string)); clusterId == 0 {
			return nil
		} else {
			dimensions["cluster"] = clusterId
		}

		dimensions["node_id"] = getBkInstId(K8sNodeObjectId, dimensions["node"].(string))
		if bizInfo, bizFound := bkSetBizCache.Get(fmt.Sprintf("%v_set_id_biz_id", K8sClusterObjectId)); bizFound {
			if bizId, bizIdFound := bizInfo.(map[int]int)[dimensions["cluster"].(int)]; bizIdFound {
				dimensions["bk_biz_id"] = bizId
			}
		}
		k8sDimisionHandler(&dimensions, k8sNodeDimension)
		deleteUselessDimension(&dimensions, k8sNodeDimension, true)
	}

	if protocol == SNMP || protocol == IPMI {
		dimensions["instance_name"] = dimensions["bk_inst_name"]
	}

	if val, ok := dimensions["bk_inst_id"]; !ok {
		bkInstId = getBkInstId(bkObjectId, dimensions["instance_name"].(string))
	} else {
		bkInstId = labelsIdValHandler(val)
	}

	if bkInstId != 0 {
		dimensions["bk_inst_id"] = bkInstId
	} else {
		return nil
	}

	if bkObjectId != K8sNodeObjectId && bkObjectId != K8sPodObjectId {
		// 业务判断
		if val, ok := dimensions["bk_biz_id"]; !ok {
			dimensions["bk_biz_id"] = getBkBizId(bkObjectId, bkInstId)
		} else {
			dimensions["bk_biz_id"] = labelsIdValHandler(val)
		}
	}

	deleteUselessDimension(&dimensions, commonDimensionFilter, false)
	return
}

func getDataId(bkObjectId string) (bkDataId string) {
	if result, found := bkCache.Get("bk_data_id"); found {
		if dataID, found := result.(map[string]string)[bkObjectId]; found {
			return dataID
		}
	} else {
		bkObjData := requestDataId()
		bkCache.Set("bk_data_id", bkObjData, time.Duration(cacheExpiration)*time.Second)
		if bkDataId, found := bkObjData[bkObjectId]; found {
			return bkDataId
		}
	}

	return bkDataId
}

func deleteUselessDimension(dimensions *map[string]interface{}, objDimensions map[string]bool, keep bool) {
	mutex.Lock()
	defer mutex.Unlock()
	for key := range *dimensions {
		if (!objDimensions[strings.ToLower(key)] && keep) || (objDimensions[strings.ToLower(key)] && !keep) {
			delete(*dimensions, key)
		}
	}
}

// k8s指标中dimision需要保留的维度信息
func k8sDimisionHandler(dimensions *map[string]interface{}, k8sDimensionKeep map[string]bool) {
	mutex.Lock()
	defer mutex.Unlock()
	metricDimension := (*dimensions)["dimision"]
	if metricDimension != nil {
		dimensionList := strings.Split(metricDimension.(string), ",")
		for _, s := range dimensionList {
			_, ok := (*dimensions)[s]
			if ok {
				k8sDimensionKeep[s] = true
			}
		}
	}
}

func labelsIdValHandler(val interface{}) (Id int) {
	switch v := val.(type) {
	case int:
		return val.(int)
	case string:
		if intValue, err := strconv.Atoi(v); err == nil {
			return intValue
		}
	default:
		return Id
	}
	return Id
}
