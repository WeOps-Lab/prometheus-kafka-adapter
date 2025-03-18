package main

import (
	"errors"
	"fmt"
	"strings"
)

// protocolProcessors 定义协议处理器映射
var protocolProcessors = map[string]func(labels map[string]string) bool{
	Kubernetes: func(labels map[string]string) bool {
		for _, handler := range []ProtocolHandler{
			protocolHandlers[K8sClusterObjectId],
			protocolHandlers[K8sNodeObjectId],
			protocolHandlers[K8sPodObjectId],
		} {
			if handler.Handle(labels) {
				return true
			}
		}
		return false
	},
	SNMP:     func(labels map[string]string) bool { return true },
	IPMI:     func(labels map[string]string) bool { return protocolHandlers[IPMI].Handle(labels) },
	Vector:   func(labels map[string]string) bool { return true },
	Automate: func(labels map[string]string) bool { return true },
	CLOUD:    func(labels map[string]string) bool { return true },
}

// shouldProcess 判断是否需要处理该指标
func shouldProcess(labels map[string]string) bool {
	if processor, exists := protocolProcessors[labels[Protocol]]; exists {
		return processor(labels)
	}
	return false
}

// getTopic 提取topic并删除无用的维度信息
func getTopic(dimensions map[string]interface{}) (string, error) {
	if dataID, ok := dimensions["bk_data_id"].(string); ok && dataID != "" {
		t := fmt.Sprintf("0bkmonitor_%v0", dataID)
		for _, key := range []string{"bk_data_id", "job"} {
			delete(dimensions, key)
		}
		return t, nil
	}
	return "", errors.New("dataID is empty or not found")
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
