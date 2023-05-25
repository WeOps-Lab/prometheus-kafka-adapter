package main

import (
	"encoding/json"
	"fmt"
	"github.com/prometheus/prometheus/prompb"
	"github.com/sirupsen/logrus"
	"math"
	"strconv"
	"time"
)

// handleSpecialValue 处理+Inf、-Inf、NaN特殊值
func handleSpecialValue(value float64) float64 {
	if math.IsNaN(value) || math.IsInf(value, 0) {
		return -1
	}
	return value
}

// processData 标准化输出数据
func processData(metricName string, dimensions map[string]interface{}, sample prompb.Sample) (data []byte, err error) {
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
	if _, nodeMetricsExist := K8sNodeMetrics[metricName]; nodeMetricsExist {
		labels["bk_obj_id"] = K8sNodeObjectId
		labels["instance_name"] = labels["node"]
		labels["cluster_name"] = labels["cluster"]
		return true
	} else if _, podMetricsExist := K8sPodMetrics[metricName]; podMetricsExist {
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
		bkObjectId = dimensions["bk_obj_id"].(string)
	)

	// 第一层过滤无业务、实例的指标
	if val, ok := dimensions["bk_inst_id"]; !ok {
		bkInstId = getK8sBkInstId(bkObjectId, dimensions["instance_name"].(string))
	} else {
		switch val.(type) {
		case int:
			bkInstId = val.(int)
		case string:
			bkInstId, _ = strconv.Atoi(val.(string))
		default:
			bkInstId = 0
		}
	}

	if bkInstId != 0 {
		dimensions["bk_inst_id"] = bkInstId
	} else {
		return nil
	}

	if val, ok := dimensions["bk_biz_id"]; !ok {
		bkBizId = getBkBizId(bkObjectId, bkInstId)
	} else {
		switch val.(type) {
		case int:
			bkBizId = val.(int)
		case string:
			bkBizId, _ = strconv.Atoi(val.(string))
		default:
			bkBizId = 0
		}
	}

	if bkBizId != 0 {
		dimensions["bk_biz_id"] = bkBizId
	} else {
		if dimensions[Protocol] != CLOUD {
			return nil
		}
	}

	if val, ok := dimensions["bk_data_id"]; !ok || val == nil {
		if bkDataId = getDataId(bkObjectId); bkDataId == "" {
			return nil
		}
		dimensions["bk_data_id"] = bkDataId
	}

	// 第二层对node、pod分别处理
	if bkObjectId == K8sPodObjectId {
		dimensions["pod_id"] = bkInstId
		dimensions["cluster"] = getK8sBkInstId(K8sClusterObjectId, dimensions["cluster_name"].(string))
		if dimensions["cluster"].(int) == 0 {
			return nil
		}
		dimensions["workload"] = getWorkloadID(dimensions["instance_name"].(string), bkInstId)
		dimensions["node_id"] = getK8sBkInstId(K8sNodeObjectId, dimensions["node"].(string))
		dimensions["namespace_id"] = getK8sBkInstId(K8sNameSpaceObjectId, fmt.Sprintf("%v (%v)", dimensions["namespace"].(string), dimensions["cluster_name"].(string)))
		deleteUselessDimension(&dimensions, K8sPodDimension, true)
	} else if bkObjectId == K8sNodeObjectId {
		dimensions["cluster"] = getK8sBkInstId(K8sClusterObjectId, dimensions["cluster"].(string))
		dimensions["node_id"] = getK8sBkInstId(K8sNodeObjectId, dimensions["node"].(string))
		deleteUselessDimension(&dimensions, K8sNodeDimension, true)
	}

	if dimensions[Protocol] == SNMP {
		dimensions["instance_name"] = dimensions["bk_inst_name"]
		delete(dimensions, "bk_inst_name")
	} else if dimensions[Protocol] == IPMI {
		dimensions["instance_name"] = dimensions["bk_inst_name"]
	}

	deleteUselessDimension(&dimensions, CommonDimensionFilter, false)
	return
}

func getDataId(bkObjectId string) (bkDataId string) {
	bkObjIdDataId := fmt.Sprintf("bkDataID@@%s", bkObjectId)
	if result, found := bkCache.Get(bkObjIdDataId); found {
		// Result found in cache, use it
		logrus.Debugf("using data id cache for object: %v", bkObjectId)
		return result.(string)
	} else {
		bkDataId = requestDataId(bkObjectId)
		// Setting cache for data id
		bkCache.Set(bkObjIdDataId, bkDataId, time.Duration(cacheExpiration)*time.Second)
	}

	return bkDataId
}

func deleteUselessDimension(dimensions *map[string]interface{}, objDimensions map[string]bool, keep bool) {
	for key := range *dimensions {
		if (!objDimensions[key] && keep) || (objDimensions[key] && !keep) {
			delete(*dimensions, key)
		}
	}
}

// dropMetrics 补充信息后，过滤出可用指标
func dropMetrics(dimensions map[string]interface{}) bool {
	if dimensions == nil {
		return true
	}

	// 丢弃业务id和实例id为0的指标
	if dimensions[Protocol] != CLOUD && (dimensions["bk_inst_id"] == 0 || dimensions["bk_biz_id"] == 0) {
		return true
	}

	if val, ok := dimensions["bk_data_id"]; !ok || val == nil {
		return true
	}

	// 过滤缺少重要信息的指标
	switch dimensions["bk_obj_id"].(string) {
	case K8sPodObjectId:
		if dimensions["cluster"].(int) == 0 || dimensions["namespace_id"].(int) == 0 {
			return true
		}
	case K8sNodeObjectId:
		if dimensions["cluster"].(int) == 0 {
			return true
		}
	}

	return false
}
