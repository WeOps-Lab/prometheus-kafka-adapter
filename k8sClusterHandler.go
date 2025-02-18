package main

type K8sClusterHandler struct{}

func (h *K8sClusterHandler) Handle(labels map[string]string) bool {
	metricName := labels["__name__"]
	if clusterMetricName, exists := K8sClusterMetrics[metricName]; exists {
		labels["__name__"] = clusterMetricName
		labels["bk_obj_id"] = K8sClusterObjectId
		return true
	}
	return false
}

func (h *K8sClusterHandler) GetObjectId() string {
	return K8sClusterObjectId
}

// ValidateDimensions 确认是否有集群名称
func (h *K8sClusterHandler) ValidateDimensions(dimensions map[string]interface{}) bool {
	if clusterInstName, ok := dimensions["cluster"]; ok {
		dimensions["instance_name"] = clusterInstName
		return true
	}
	return false
}

func (h *K8sClusterHandler) ProcessDimensions(dimensions map[string]interface{}) bool {
	return true // 集群对象全部放通
}
