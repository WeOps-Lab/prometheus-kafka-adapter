package main

import (
	"encoding/json"
	"fmt"
	"github.com/prometheus/prometheus/prompb"
	"github.com/sirupsen/logrus"
	"math"
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
	if dimensions["protocol"] != AutoMate {
		timestamp = time.Unix(sample.Timestamp/1000, 0).UTC().UnixNano() / int64(time.Millisecond)
	} else {
		timestamp = dimensions["timestamp"].(int64)
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
		return true
	} else if _, podMetricsExist := K8sPodMetrics[metricName]; podMetricsExist {
		labels["bk_obj_id"] = K8sPodObjectId
		labels["instance_name"] = labels["uid"]
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

	bkObjectId := dimensions["bk_obj_id"].(string)
	instanceName := dimensions["instance_name"].(string)

	// 第一层过滤k8s中无业务、实例的指标
	dimensions["bk_inst_id"] = getK8sBkInstId(bkObjectId, instanceName)
	if dimensions["bk_inst_id"].(int) == 0 {
		return dimensions
	}

	dimensions["bk_biz_id"] = getBkBizId(bkObjectId, dimensions["bk_inst_id"].(int))
	if dimensions["bk_biz_id"].(int) == 0 {
		return dimensions
	}

	dimensions["bk_data_id"] = getDataId(bkObjectId)

	// 第二层对node、pod分别处理
	if bkObjectId == K8sPodObjectId {
		dimensions["pod_id"] = dimensions["bk_inst_id"]
		dimensions["cluster"] = getK8sBkInstId(K8sClusterObjectId, dimensions["cluster"].(string))
		dimensions["workload"] = getWorkloadID(instanceName, dimensions["bk_inst_id"].(int))
		dimensions["node_id"] = getK8sBkInstId(K8sNodeObjectId, dimensions["node"].(string))
		dimensions["namespace_id"] = getK8sBkInstId(K8sNameSpaceObjectId, fmt.Sprintf("%v (%v)", dimensions["namespace"].(string), dimensions["cluster"].(string)))
		deleteUselessDimension(&dimensions, K8sPodDimension)
	} else if bkObjectId == K8sNodeObjectId {
		dimensions["cluster"] = getK8sBkInstId(K8sClusterObjectId, dimensions["cluster"].(string))
		dimensions["node_id"] = getK8sBkInstId(K8sNodeObjectId, dimensions["node"].(string))
		deleteUselessDimension(&dimensions, K8sNodeDimension)
	}

	return dimensions
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

func deleteUselessDimension(dimensions *map[string]interface{}, objDimensions map[string]bool) {
	for key := range *dimensions {
		if !objDimensions[key] {
			delete(*dimensions, key)
		}
	}
}

// dropMetrics 补充信息后，过滤出可用指标
func dropMetrics(dimensions map[string]interface{}) bool {

	// 丢弃业务id和实例id为0的指标
	if dimensions["bk_inst_id"] == 0 || dimensions["bk_biz_id"] == 0 {
		return true
	}

	if dimensions["bk_data_id"] != "" {
		kafkaTopic = fmt.Sprintf("0bkmonitor_%v0", dimensions["bk_data_id"])
	} else {
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
