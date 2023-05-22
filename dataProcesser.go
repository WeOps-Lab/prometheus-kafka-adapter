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

	// 删除无用数据
	keysToDelete := []string{"__name__", "protocol", "job"}
	for _, key := range keysToDelete {
		delete(dimensions, key)
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

// k8sMetricsPreHandler 判断k8s指标，并补充k8s类的bk_object_id
func k8sMetricsPreHandler(labels map[string]string) (exist bool) {
	if _, nodeMetricsExist := K8sNodeMetrics[labels["__name__"]]; nodeMetricsExist {
		labels["bk_object_id"] = K8sNodeObjectId
		labels["instance_name"] = labels["node"]
		return true
	} else if _, podMetricsExist := K8sPodMetrics[labels["__name__"]]; podMetricsExist {
		labels["bk_object_id"] = K8sPodObjectId
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

	bkObjectId := dimensions["bk_object_id"].(string)
	instanceName := dimensions["instance_name"].(string)

	dimensions["bk_inst_id"] = getK8sBkInstId(bkObjectId, instanceName)
	dimensions["bk_biz_id"] = getBkBizId(bkObjectId, dimensions["bk_inst_id"].(int))
	dimensions["bk_data_id"] = getDataId(bkObjectId)

	if bkObjectId == K8sPodObjectId {
		dimensions["workload"] = getWorkloadID(instanceName, dimensions["bk_inst_id"].(int))
		dimensions["pod_id"] = dimensions["bk_inst_id"]
		dimensions["node_id"] = getK8sBkInstId(K8sNodeObjectId, dimensions["node"].(string))
		namespace, namespaceExist := dimensions["namespace"].(string)
		if namespaceExist {
			namespaceCluster := fmt.Sprintf("%v (%v)", namespace, dimensions["cluster"].(string))
			dimensions["namespace_id"] = getK8sBkInstId(K8sNameSpaceObjectId, namespaceCluster)
		} else {
			logrus.Debugf("k8s pod metrics without namespace label: %v", dimensions["__name__"])
			dimensions["namespace_id"] = 0
		}
	} else if bkObjectId == K8sNodeObjectId {
		dimensions["cluster_id"] = getK8sBkInstId(K8sClusterObjectId, dimensions["cluster"].(string))
		dimensions["node_id"] = getK8sBkInstId(K8sNodeObjectId, dimensions["node"].(string))
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

func nullIdHandler() {

}
