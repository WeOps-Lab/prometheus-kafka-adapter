package main

import (
	"encoding/json"
	"fmt"
	"math"
	"strconv"
	"time"

	"github.com/prometheus/prometheus/prompb"
)

// handleSpecialValue 处理+Inf、-Inf、NaN特殊值
func handleSpecialValue(sample prompb.Sample) (float64, bool) {
	switch {
	case math.IsInf(sample.Value, -1), math.IsNaN(sample.Value):
		return 0, true
	case math.IsInf(sample.Value, 1):
		return -1, true
	default:
		return sample.Value, false
	}
}

// formatMetricsData 标准化输出数据
func formatMetricsData(metricName string, dimensions map[string]interface{}, sample prompb.Sample, bkSource bool) (data []byte, err error) {
	var handleData interface{}

	// prometheus中的特殊值处理 +-Inf、Nan
	value, specialValue := handleSpecialValue(sample)
	if specialValue {
		logrus.Debugf("Handle special value (+-Inf or NaN), metric name: %v, dimensions: %v, sample info: %v", metricName, dimensions, sample)
	}

	if bkSource {
		strVal := fmt.Sprintf("%.2f", value)
		metricsValue, _ := strconv.ParseFloat(strVal, 64)

		// 检查并断言 dimensions["bk_biz_id"]
		bkBizIdStr, ok := dimensions["bk_biz_id"].(string)
		if !ok {
			return nil, fmt.Errorf("bk_biz_id is not a string or is missing")
		}
		bkBizId, err := strconv.Atoi(bkBizIdStr)
		if err != nil {
			return nil, fmt.Errorf("failed to convert bk_biz_id to int: %v", err)
		}

		// 检查并断言 dimensions["bk_cloud_id"]
		bkCloudIdStr, ok := dimensions["bk_cloud_id"].(string)
		if !ok {
			return nil, fmt.Errorf("bk_cloud_id is not a string or is missing")
		}
		bkCloudId, err := strconv.Atoi(bkCloudIdStr)
		if err != nil {
			return nil, fmt.Errorf("failed to convert bk_cloud_id to int: %v", err)
		}

		handleData = BKMetricsData{
			Timestamp: time.Unix(sample.Timestamp/1000, (sample.Timestamp%1000)*int64(time.Millisecond)),
			BkBizId:   bkBizId,
			BkCloudId: bkCloudId,
			GroupInfo: GroupInfo{
				{
					BkCollectConfigId: dimensions["bk_collect_config_id"].(string),
				},
			},
			Prometheus: Prometheus{
				Collector: Collector{
					Metrics: Metrics{
						{
							Key:       metricName,
							Labels:    dimensions,
							Timestamp: time.Unix(sample.Timestamp/1000, 0).UTC().UnixNano() / int64(time.Second),
							Value:     metricsValue,
						},
					},
				},
			},
			Service: "prometheus",
			Type:    "metricbeat",
		}
	} else {
		var timestamp int64
		timestamp = time.Unix(sample.Timestamp/1000, 0).UTC().UnixNano() / int64(time.Millisecond)
		handleData = MetricsData{
			Data: []struct {
				Dimension map[string]interface{} `json:"dimension"`
				Metrics   map[string]float64     `json:"metrics"`
				Timestamp int64                  `json:"timestamp"`
			}{
				{
					Dimension: dimensions,
					Metrics: map[string]float64{
						metricName: value,
					},
					Timestamp: timestamp,
				},
			},
		}
	}

	return json.Marshal(handleData)
}
