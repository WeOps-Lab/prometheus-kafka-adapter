package main

import "fmt"

type K8sPodHandler struct{}

func (h *K8sPodHandler) Handle(labels map[string]string) bool {
	metricName := labels["__name__"]
	if podMetricName, exists := K8sPodMetrics[metricName]; exists {
		if _, ok := labels["pod"]; !ok {
			return false
		}
		labels["__name__"] = podMetricName
		labels["bk_obj_id"] = K8sPodObjectId
		labels["instance_name"] = labels["uid"]
		return true
	}
	return false
}

func (h *K8sPodHandler) GetObjectId() string {
	return K8sPodObjectId
}

func (h *K8sPodHandler) ValidateDimensions(dimensions map[string]interface{}) bool {
	return dimensions["pod"] != nil
}

func (h *K8sPodHandler) ProcessDimensions(dimensions map[string]interface{}) bool {
	instanceName, ok := dimensions["instance_name"].(string)
	if !ok || instanceName == "" {
		return false
	}
	bkInstId := getBkInstId(K8sPodObjectId, instanceName)
	if bkInstId == 0 {
		return false
	}

	dimensions["pod_id"] = bkInstId
	dimensions["bk_inst_id"] = bkInstId

	// 处理集群ID
	clusterId := getBkInstId(K8sClusterObjectId, dimensions["cluster_name"].(string))
	if clusterId == 0 {
		return false
	}
	dimensions["cluster"] = clusterId

	// 处理工作负载
	if podWorkloadInfo, found := bkObjRelaCache.Get(fmt.Sprintf("pod_workload_rel_map@@%v", bkInstId)); found && podWorkloadInfo.(int) != 0 {
		dimensions["workload"] = podWorkloadInfo.(int)
	} else {
		return false
	}

	// 处理节点ID
	if node, ok := dimensions["node"].(string); ok {
		if nodeId := getBkInstId(K8sNodeObjectId, node); nodeId != 0 {
			dimensions["node_id"] = nodeId
		} else {
			return false
		}
	}

	// 处理命名空间
	namespaceId := getBkInstId(K8sNameSpaceObjectId, fmt.Sprintf("%v (%v)", dimensions["namespace"].(string), dimensions["cluster_name"].(string)))
	if namespaceId == 0 {
		return false
	}
	dimensions["namespace_id"] = namespaceId

	// 处理业务ID
	if bizInfo, bizFound := bkSetBizCache.Get(fmt.Sprintf("%v_set_id_biz_id", K8sNameSpaceObjectId)); bizFound {
		if bizId, bizIdFound := bizInfo.(map[int]int)[namespaceId]; bizIdFound {
			dimensions["bk_biz_id"] = bizId
		}
	}

	k8sDimisionHandler(&dimensions, k8sPodDimension)
	deleteUselessDimension(&dimensions, k8sPodDimension, true)
	return true
}
