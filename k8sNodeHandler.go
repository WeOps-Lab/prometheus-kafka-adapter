package main

import (
	"fmt"
	"github.com/sirupsen/logrus"
)

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
	clusterName := dimensions["cluster"].(string)
	clusterId := getBkInstId(K8sClusterObjectId, clusterName)
	if clusterId == 0 {
		logrus.Debugf("K8sNodeHandler ProcessDimensions clusterId is 0, cluster_name: %s", clusterName)
		return false
	}
	dimensions["cluster"] = clusterId

	// 处理节点ID
	if node, nodeExist := dimensions["node"].(string); nodeExist {
		if nodeId := getBkInstId(K8sNodeObjectId, fmt.Sprintf("%s(%s)", node, clusterName)); nodeId != 0 {
			dimensions["node_id"] = nodeId
		} else {
			logrus.Debugf("K8sNodeHandler ProcessDimensions can not find bk_node(bk_inst_id), node: %s, cluster_name: %s", node, clusterName)
			return false
		}
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
