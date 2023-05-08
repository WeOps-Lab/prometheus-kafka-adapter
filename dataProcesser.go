package main

import (
	"encoding/json"
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

// k8sMetricsPreHandler 判断k8s指标，并补充k8s类的bk_object_id
func k8sMetricsPreHandler(labels map[string]string) (exist bool) {
	metricName := labels["__name__"]
	if _, nodeMetricsExist := K8sNodeMetrics[metricName]; nodeMetricsExist {
		labels["bk_object_id"] = K8sNodeObjectId
		labels["instance_name"] = labels["node"]
		return true
	} else if _, podMetricsExist := K8sPodMetrics[metricName]; podMetricsExist {
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
		dimensions["workload"] = getWorkloadID(dimensions["bk_inst_id"].(int))
	}

	return dimensions
}

func getDataId(bkObjectId string) (bkDataId int) {
	if result, found := bkCache.Get(bkObjectId); found {
		// Result found in cache, use it
		logrus.Debugf("using data id cache for object: %v", bkObjectId)
		return result.(int)
	} else {
		query := "SELECT bk_data_id FROM home_application_customtstable WHERE id IN (SELECT JSON_EXTRACT(JSON_ARRAYAGG(bk_ts_table_ids), '$[0][0]') FROM home_application_monitorcentercustomts WHERE monitor_obj_id = (SELECT id FROM home_application_monitorobject WHERE bk_obj_id = ?))"
		err := db.QueryRow(query, bkObjectId).Scan(&bkDataId)
		if err != nil {
			logrus.WithError(err).Errorf("find bk_obj_id [%s] data id error", bkObjectId)
		}
		// Setting cache for data id
		bkCache.Set(bkObjectId, bkDataId, time.Duration(cacheExpiration)*time.Second)
	}

	return bkDataId
}
