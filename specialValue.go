package main

import (
	"fmt"
	"github.com/prometheus/prometheus/prompb"
	"github.com/sirupsen/logrus"
	"math"
)

// handleSpecialValue 检查是否存在特殊值(+Inf、-Inf、NaN)
func hasSpecialValue(metricName string, sample prompb.Sample, dimensions map[string]interface{}) bool {
	if math.IsInf(sample.Value, 1) {
		logSpecialValue(metricName, "+Inf", sample, dimensions)
		return true
	} else if math.IsInf(sample.Value, -1) {
		logSpecialValue(metricName, "-Inf", sample, dimensions)
		return true
	} else if math.IsNaN(sample.Value) {
		logSpecialValue(metricName, "NaN", sample, dimensions)
		return true
	}
	return false
}

func logSpecialValue(metricName, valueType string, sample prompb.Sample, dimensions map[string]interface{}) {
	dimensionsStr := fmt.Sprintf("%v", dimensions)
	instanceName := "unknown"
	if name, ok := dimensions["instance_name"]; ok {
		instanceName = fmt.Sprint(name)
	}

	logrus.Debugf("Dropping metric due to special value: metric=%s, value_type=%s, value=%v, timestamp=%v, instance=%s, dimensions=%s",
		metricName, valueType, sample.Value, sample.Timestamp, instanceName, dimensionsStr)
	weopsSpecialValueDropped.WithLabelValues(metricName, valueType, instanceName).Add(float64(1))
}
