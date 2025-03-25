package main

import "fmt"

type K8sNodeHandler struct{}

func (h *K8sNodeHandler) Handle(labels map[string]string) bool {
	metricName := labels["__name__"]
	if nodeMetricName, exists := K8sNodeMetrics[metricName]; exists {
		if _, ok := labels["node"]; !ok {
			return false
		}
		labels["__name__"] = nodeMetricName
		labels["bk_obj_id"] = K8sNodeObjectId
		labels["instance_name"] = fmt.Sprintf("%s(%s)", labels["node"], labels["cluster"])
		labels["cluster_name"] = labels["cluster"]
		return true
	}
	return false
}

func (h *K8sNodeHandler) GetObjectId() string {
	return K8sNodeObjectId
}

func (h *K8sNodeHandler) ValidateDimensions(dimensions map[string]interface{}) bool {
	return dimensions["node"] != nil
}

func (h *K8sNodeHandler) ProcessDimensions(dimensions map[string]interface{}) bool {
	clusterId := getBkInstId(K8sClusterObjectId, dimensions["cluster"].(string))
	if clusterId == 0 {
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
