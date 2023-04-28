package main

import (
	"encoding/json"
	"github.com/prometheus/prometheus/prompb"
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
	timestamp := time.Unix(sample.Timestamp/1000, 0).UTC().UnixNano() / int64(time.Millisecond)

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

// k8sMetricsExist 判断k8s指标，并补充k8s类的bk_object_id
func k8sMetricsHandle(labels map[string]string, metricName string) (exist bool) {
	if _, nodeMetricsExist := K8sNodeMetrics[metricName]; nodeMetricsExist {
		labels["bk_object_id"] = K8sNodeObjectId
		labels["instance_name"] = labels["node"]
		return true
	} else if _, podMetricsExist := K8sPodMetrics[metricName]; podMetricsExist {
		labels["bk_object_id"] = K8sPodObjectId
		return true
	} else {
		return false
	}
}

// TODO: 调用接口获取到以下字段信息
func fillUpBkInfo(labels map[string]string) (dimensions map[string]interface{}) {
	dimensions = make(map[string]interface{})
	for key, value := range labels {
		dimensions[key] = value
	}

	return dimensions
}
