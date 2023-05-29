package main

import (
	"encoding/json"
	"fmt"
	"github.com/prometheus/prometheus/prompb"
	"github.com/sirupsen/logrus"
	"math"
	"strconv"
	"strings"
	"time"
)

// handleSpecialValue 处理+Inf、-Inf、NaN特殊值
func handleSpecialValue(value float64) float64 {
	if math.IsNaN(value) || math.IsInf(value, 0) {
		return -1
	}
	return value
}

// formatMetricsData 标准化输出数据
func formatMetricsData(metricName string, dimensions map[string]interface{}, sample prompb.Sample) (data []byte, err error) {
	var timestamp int64
	if dimensions[Protocol] != CLOUD {
		timestamp = time.Unix(sample.Timestamp/1000, 0).UTC().UnixNano() / int64(time.Millisecond)
	} else {
		timestamp, err = strconv.ParseInt(dimensions["metric_timestamp"].(string), 10, 64)
		if err != nil {
			logrus.WithError(err).Errorf("%v parse timestamp error", dimensions[Protocol])
		}
		delete(dimensions, "metric_timestamp")
	}

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
	metricName := labels["__name__"]
	if _, ok := labels["node"]; !ok {
		return false
	}
	if _, ok := labels["cluster"]; !ok {
		return false
	}
	if _, nodeMetricsExist := K8sNodeMetrics[metricName]; nodeMetricsExist {
		labels["bk_obj_id"] = K8sNodeObjectId
		labels["instance_name"] = labels["node"]
		labels["cluster_name"] = labels["cluster"]
		return true
	} else if _, podMetricsExist := K8sPodMetrics[metricName]; podMetricsExist {
		if _, ok := labels["pod"]; !ok {
			return false
		}
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
		bkBizId    int
		bkDataId   string
		protocol   = dimensions[Protocol].(string)
		bkObjectId = dimensions["bk_obj_id"].(string)
	)

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
		}

		dimensions["node_id"] = getBkInstId(K8sNodeObjectId, dimensions["node"].(string))
		dimensions["namespace_id"] = getBkInstId(K8sNameSpaceObjectId, fmt.Sprintf("%v (%v)", dimensions["namespace"].(string), dimensions["cluster_name"].(string)))
		namespaceId, ok := dimensions["namespace_id"].(int)
		if !ok || namespaceId == 0 {
			return nil
		}

		if bizInfo, bizFound := bkSetBizCache.Get(fmt.Sprintf("%v_set_id_biz_id", K8sNameSpaceObjectId)); bizFound {
			if bizId, bizIdFound := bizInfo.(map[int]int)[namespaceId]; bizIdFound && bizId != 0 {
				dimensions["bk_biz_id"] = bizId
			} else {
				return nil
			}
		}
		k8sDimisionHandler(&dimensions, k8sPodDimension)
		deleteUselessDimension(&dimensions, k8sPodDimension, true)
	} else if bkObjectId == K8sNodeObjectId {
		if clusterId := getBkInstId(K8sClusterObjectId, dimensions["cluster"].(string)); clusterId == 0 {
			return nil
		} else {
			dimensions["cluster"] = clusterId
		}

		dimensions["node_id"] = getBkInstId(K8sNodeObjectId, dimensions["node"].(string))
		if bizInfo, bizFound := bkSetBizCache.Get(fmt.Sprintf("%v_set_id_biz_id", K8sClusterObjectId)); bizFound {
			if bizId, bizIdFound := bizInfo.(map[int]int)[dimensions["cluster"].(int)]; bizIdFound && bizId != 0 {
				dimensions["bk_biz_id"] = bizId
			} else {
				return nil
			}
		}
		k8sDimisionHandler(&dimensions, k8sNodeDimension)
		deleteUselessDimension(&dimensions, k8sNodeDimension, true)
	}

	if protocol == SNMP {
		dimensions["instance_name"] = dimensions["bk_inst_name"]
		delete(dimensions, "bk_inst_name")
	} else if protocol == IPMI {
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

	// 业务判断
	if val, ok := dimensions["bk_biz_id"]; !ok {
		bkBizId = getBkBizId(bkObjectId, bkInstId)
	} else {
		bkBizId = labelsIdValHandler(val)
	}

	if bkBizId != 0 {
		dimensions["bk_biz_id"] = bkBizId
	} else {
		if protocol != CLOUD {
			return nil
		}
	}

	deleteUselessDimension(&dimensions, commonDimensionFilter, false)
	return
}

func getDataId(bkObjectId string) (bkDataId string) {
	bkObjIdDataId := fmt.Sprintf("bk_data_id_%s", bkObjectId)
	if result, found := bkCache.Get(bkObjIdDataId); found {
		return result.(string)
	} else {
		bkDataId = requestDataId(bkObjectId)
		bkCache.Set(bkObjIdDataId, bkDataId, time.Duration(cacheExpiration)*time.Second)
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
