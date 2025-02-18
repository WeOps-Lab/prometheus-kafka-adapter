package main

import (
	"encoding/json"
	"fmt"
	"github.com/sirupsen/logrus"
	"strconv"
	"time"

	"github.com/prometheus/prometheus/prompb"
)

// formatMetricsData 标准化输出数据
func formatMetricsData(metricName string, dimensions map[string]interface{}, sample prompb.Sample, bkSource bool) ([]byte, error) {
	// 检查特殊值
	if hasSpecialValue(metricName, sample, dimensions) {
		return nil, nil
	}

	if bkSource {
		return formatBKMetrics(metricName, dimensions, sample)
	}
	return formatStandardMetrics(metricName, dimensions, sample)
}

// formatBKMetrics 处理蓝鲸源数据格式
func formatBKMetrics(metricName string, dimensions map[string]interface{}, sample prompb.Sample) ([]byte, error) {
	// 获取并验证必要字段
	bkBizId, err := getIntField(dimensions, "bk_biz_id")
	if err != nil {
		logrus.Debugf("bk_biz_id is not a string or is missing")
		return nil, nil
	}

	bkCloudId, err := getIntField(dimensions, "bk_cloud_id")
	if err != nil {
		logrus.Debugf("bk_cloud_id is not a string or is missing")
		return nil, nil
	}

	configId, ok := dimensions["bk_collect_config_id"].(string)
	if !ok {
		logrus.Debugf("bk_collect_config_id is not a string or is missing")
		return nil, nil
	}

	// 格式化数值
	value, _ := strconv.ParseFloat(fmt.Sprintf("%.2f", sample.Value), 64)
	timestamp := time.Unix(sample.Timestamp/1000, 0).UTC()

	data := BKMetricsData{
		Timestamp: time.Unix(sample.Timestamp/1000, (sample.Timestamp%1000)*int64(time.Millisecond)),
		BkBizId:   bkBizId,
		BkCloudId: bkCloudId,
		GroupInfo: GroupInfo{{BkCollectConfigId: configId}},
		Prometheus: Prometheus{
			Collector: Collector{
				Metrics: Metrics{{
					Key:       metricName,
					Labels:    dimensions,
					Timestamp: timestamp.UnixNano() / int64(time.Second),
					Value:     value,
				}},
			},
		},
		Service: "prometheus",
		Type:    "metricbeat",
	}

	return json.Marshal(data)
}

// formatStandardMetrics 处理标准数据格式
func formatStandardMetrics(metricName string, dimensions map[string]interface{}, sample prompb.Sample) ([]byte, error) {
	timestamp := time.Unix(sample.Timestamp/1000, 0).UTC().UnixNano() / int64(time.Millisecond)

	data := MetricsData{
		Data: []struct {
			Dimension map[string]interface{} `json:"dimension"`
			Metrics   map[string]float64     `json:"metrics"`
			Timestamp int64                  `json:"timestamp"`
		}{{
			Dimension: dimensions,
			Metrics:   map[string]float64{metricName: sample.Value},
			Timestamp: timestamp,
		}},
	}

	return json.Marshal(data)
}

// getIntField 从dimensions中获取并转换整数字段
func getIntField(dimensions map[string]interface{}, key string) (int, error) {
	val, ok := dimensions[key].(string)
	if !ok {
		return 0, fmt.Errorf("field %s not found or not a string", key)
	}

	intVal, err := strconv.Atoi(val)
	if err != nil {
		return 0, fmt.Errorf("failed to convert %s to int: %v", key, err)
	}

	return intVal, nil
}
