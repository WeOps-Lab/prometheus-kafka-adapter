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
	switch {
	case math.IsInf(value, -1), math.IsNaN(value):
		return 0
	case math.IsInf(value, 1):
		return -1
	default:
		return value
	}
}

// formatMetricsData 标准化输出数据
func formatMetricsData(metricName string, dimensions map[string]interface{}, sample prompb.Sample, bkSource bool) (data []byte, err error) {
	var handleData interface{}

	if bkSource {
		strVal := fmt.Sprintf("%.2f", handleSpecialValue(sample.Value))
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
						metricName: handleSpecialValue(sample.Value),
					},
					Timestamp: timestamp,
				},
			},
		}
	}

	return json.Marshal(handleData)
}

// k8sMetricsPreHandler 判断k8s指标，并补充k8s类的bk_obj_id
func k8sMetricsPreHandler(labels map[string]string) bool {
	metricName := labels["__name__"]

	if clusterMetricName, clusterMetricsExist := K8sClusterMetrics[metricName]; clusterMetricsExist {
		labels["__name__"] = clusterMetricName
		labels["bk_obj_id"] = K8sClusterObjectId
		return true
	}

	if nodeMetricName, nodeMetricsExist := K8sNodeMetrics[metricName]; nodeMetricsExist {
		if _, ok := labels["node"]; !ok {
			return false
		}
		labels["__name__"] = nodeMetricName
		labels["bk_obj_id"] = K8sNodeObjectId
		labels["instance_name"] = labels["node"]
		labels["cluster_name"] = labels["cluster"]
		return true
	}

	if podMetricName, podMetricsExist := K8sPodMetrics[metricName]; podMetricsExist {
		if _, ok := labels["pod"]; !ok {
			return false
		}
		labels["__name__"] = podMetricName
		labels["bk_obj_id"] = K8sPodObjectId
		labels["instance_name"] = labels["uid"]
		labels["cluster_name"] = labels["cluster"]
		return true
	}

	return false
}

// fillUpBkInfo 补充蓝鲸指标信息
func fillUpBkInfo(labels map[string]string) map[string]interface{} {
	// 初始化维度信息
	dimensions := make(map[string]interface{})
	for key, value := range labels {
		dimensions[key] = value
	}

	// 直接返回 Vector 协议的维度信息
	if labels[Protocol] == Vector {
		return dimensions
	}

	// 验证必需字段
	bkObjectId, ok := dimensions["bk_obj_id"].(string)
	if !ok || bkObjectId == "" {
		logrus.Debugf("bk_obj_id is null: %v", labels)
		return nil
	}

	protocol, ok := dimensions[Protocol].(string)
	if !ok || protocol == "" {
		logrus.Debugf("protocol is null: %v", labels)
		return nil
	}

	// 更新 objList
	if !objList[bkObjectId] {
		mutex.Lock()
		objList[bkObjectId] = true
		mutex.Unlock()
	}

	// 处理 bk_data_id
	if val, ok := dimensions["bk_data_id"]; !ok || val == nil {
		if bkDataId := getDataId(bkObjectId); bkDataId == "" {
			logrus.Debugf("bk_data_id is null: %v", labels)
			return nil
		} else {
			dimensions["bk_data_id"] = bkDataId
		}
	}

	// 处理K8s对象
	switch bkObjectId {
	// cluster类全放通
	case K8sClusterObjectId:
		return dimensions
	case K8sPodObjectId:
		if !handleK8sPodObjectId(dimensions, labels) {
			return nil
		}
	case K8sNodeObjectId:
		if !handleK8sNodeObjectId(dimensions, labels) {
			return nil
		}
	}

	// 处理 SNMP 和 IPMI 协议
	if protocol == SNMP || protocol == IPMI {
		dimensions["instance_name"] = dimensions["bk_inst_name"]
	}

	// 处理 bk_inst_id
	bkInstId := handleBkInstId(dimensions, bkObjectId)
	if bkInstId == 0 {
		logrus.Debugf("bk_inst_id is null: %v", labels)
		return nil
	}
	dimensions["bk_inst_id"] = bkInstId

	// 处理 bk_biz_id
	if bkObjectId != K8sNodeObjectId && bkObjectId != K8sPodObjectId {
		handleBkBizId(dimensions, bkObjectId, bkInstId)
	}

	return dimensions
}

func handleK8sPodObjectId(dimensions map[string]interface{}, labels map[string]string) bool {
	bkInstId := getBkInstId(K8sPodObjectId, dimensions["instance_name"].(string))
	if bkInstId == 0 {
		return false
	}
	dimensions["pod_id"] = bkInstId
	dimensions["bk_inst_id"] = bkInstId

	clusterId := getBkInstId(K8sClusterObjectId, dimensions["cluster_name"].(string))
	if clusterId == 0 {
		logrus.Debugf("cannot find k8s pod clusterId: %v", labels)
		return false
	}
	dimensions["cluster"] = clusterId

	if podWorkloadInfo, found := bkObjRelaCache.Get(fmt.Sprintf("pod_workload_rel_map@@%v", bkInstId)); found && podWorkloadInfo.(int) != 0 {
		dimensions["workload"] = podWorkloadInfo.(int)
	} else {
		logrus.Debugf("cannot find k8s pod workload: %v", labels)
		return false
	}

	if node, ok := dimensions["node"].(string); ok {
		if nodeId := getBkInstId(K8sNodeObjectId, node); nodeId != 0 {
			dimensions["node_id"] = nodeId
		} else {
			logrus.Debugf("cannot find k8s pod node_id: %v", labels)
			return false
		}
	}

	namespaceId := getBkInstId(K8sNameSpaceObjectId, fmt.Sprintf("%v (%v)", dimensions["namespace"].(string), dimensions["cluster_name"].(string)))
	if namespaceId == 0 {
		logrus.Debugf("cannot find k8s pod namespace_id: %v", labels)
		return false
	}
	dimensions["namespace_id"] = namespaceId

	if bizInfo, bizFound := bkSetBizCache.Get(fmt.Sprintf("%v_set_id_biz_id", K8sNameSpaceObjectId)); bizFound {
		if bizId, bizIdFound := bizInfo.(map[int]int)[namespaceId]; bizIdFound {
			dimensions["bk_biz_id"] = bizId
		}
	}

	k8sDimisionHandler(&dimensions, k8sPodDimension)
	deleteUselessDimension(&dimensions, k8sPodDimension, true)
	return true
}

func handleK8sNodeObjectId(dimensions map[string]interface{}, labels map[string]string) bool {
	clusterId := getBkInstId(K8sClusterObjectId, dimensions["cluster"].(string))
	if clusterId == 0 {
		logrus.Debugf("cannot find k8s node clusterId: %v", labels)
		return false
	}
	dimensions["cluster"] = clusterId

	if node, ok := dimensions["node"].(string); ok {
		dimensions["node_id"] = getBkInstId(K8sNodeObjectId, node)
	}

	if bizInfo, bizFound := bkSetBizCache.Get(fmt.Sprintf("%v_set_id_biz_id", K8sClusterObjectId)); bizFound {
		if bizId, bizIdFound := bizInfo.(map[int]int)[clusterId]; bizIdFound {
			dimensions["bk_biz_id"] = bizId
		}
	}

	k8sDimisionHandler(&dimensions, k8sNodeDimension)
	deleteUselessDimension(&dimensions, k8sNodeDimension, true)
	return true
}

func handleBkInstId(dimensions map[string]interface{}, bkObjectId string) int {
	if val, ok := dimensions["bk_inst_id"]; ok {
		return labelsIdValHandler(val)
	}
	return getBkInstId(bkObjectId, dimensions["instance_name"].(string))
}

func handleBkBizId(dimensions map[string]interface{}, bkObjectId string, bkInstId int) {
	if val, ok := dimensions["bk_biz_id"]; ok {
		dimensions["bk_biz_id"] = labelsIdValHandler(val)
	} else {
		dimensions["bk_biz_id"] = getBkBizId(bkObjectId, bkInstId)
	}
}

func getDataId(bkObjectId string) string {
	key := fmt.Sprintf("bk_data_id@%s", bkObjectId)
	result, found := bkCache.Get(key)
	if found {
		return result.(string)
	}

	weopsObjGetDataIdFailTotal.WithLabelValues(bkObjectId).Add(float64(1))
	logrus.Debugf("not found data id cache for object: %s", bkObjectId)
	return ""
}

func deleteUselessDimension(dimensions *map[string]interface{}, objDimensions map[string]bool, keep bool) {
	mutex.Lock()
	defer mutex.Unlock()

	for key := range *dimensions {
		lowerKey := strings.ToLower(key)
		_, exists := objDimensions[lowerKey]

		if keep && !exists {
			delete(*dimensions, key)
		} else if !keep && exists {
			if key != "__name__" {
				(*dimensions)[fmt.Sprintf("__%v__", key)] = (*dimensions)[key]
			}
			delete(*dimensions, key)
		}
	}
}

// k8s指标中dimision需要保留的维度信息
func k8sDimisionHandler(dimensions *map[string]interface{}, k8sDimensionKeep map[string]bool) {
	metricDimension, exists := (*dimensions)["dimision"]
	if !exists || metricDimension == nil {
		return
	}

	dimensionList := strings.Split(metricDimension.(string), ",")
	mutex.Lock()
	defer mutex.Unlock()

	for _, s := range dimensionList {
		if _, ok := (*dimensions)[s]; ok {
			k8sDimensionKeep[s] = true
		}
	}
}

func labelsIdValHandler(val interface{}) int {
	switch v := val.(type) {
	case int:
		return v
	case string:
		if intValue, err := strconv.Atoi(v); err == nil {
			return intValue
		}
	}
	return 0
}
