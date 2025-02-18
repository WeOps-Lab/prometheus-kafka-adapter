package main

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/sirupsen/logrus"
)

// fillUpBkInfo 补充蓝鲸指标信息
func fillUpBkInfo(labels map[string]string) map[string]interface{} {
	// 转换labels为dimensions
	dimensions := make(map[string]interface{})
	for k, v := range labels {
		dimensions[k] = v
	}

	// Vector协议特殊处理
	if labels[Protocol] == Vector {
		return dimensions
	}

	// 基础字段验证
	if err := validateBasicFields(dimensions); err != nil {
		logrus.Debugf("Basic field validation failed: %v, labels: %v", err, labels)
		return nil
	}

	// 处理bk_data_id
	if err := processBkDataId(dimensions); err != nil {
		logrus.Debugf("Process bk_data_id failed: %v, labels: %v", err, labels)
		return nil
	}

	// 处理维度信息
	if err := processDimensions(dimensions); err != nil {
		logrus.Debugf("Process dimensions failed: %v, labels: %v", err, labels)
		return nil
	}

	return dimensions
}

// validateBasicFields 验证基础字段
func validateBasicFields(dimensions map[string]interface{}) error {
	bkObjectId, ok := dimensions["bk_obj_id"].(string)
	if !ok || bkObjectId == "" {
		return errors.New("bk_obj_id is null")
	}

	protocol, ok := dimensions[Protocol].(string)
	if !ok || protocol == "" {
		return errors.New("protocol is null")
	}

	if !objList[bkObjectId] {
		mutex.Lock()
		objList[bkObjectId] = true
		mutex.Unlock()
	}

	return nil
}

// processBkDataId 处理数据ID
func processBkDataId(dimensions map[string]interface{}) error {
	bkObjectId := dimensions["bk_obj_id"].(string)

	if _, ok := dimensions["bk_data_id"]; !ok {
		dataId := getDataId(bkObjectId)
		if dataId == "" {
			return errors.New("failed to get bk_data_id")
		}
		dimensions["bk_data_id"] = dataId
	}
	return nil
}

// processDimensions 处理维度信息
func processDimensions(dimensions map[string]interface{}) error {
	bkObjectId := dimensions["bk_obj_id"].(string)
	protocol := dimensions[Protocol].(string)

	// 选择并执行处理器
	var handler ProtocolHandler
	if h, exists := protocolHandlers[bkObjectId]; exists {
		handler = h
	} else if h, exists := protocolHandlers[protocol]; exists {
		handler = h
	} else {
		return errors.New("no handler found")
	}

	// 验证和处理维度
	if !handler.ValidateDimensions(dimensions) || !handler.ProcessDimensions(dimensions) {
		return errors.New("dimensions validation or processing failed")
	}

	// 处理bk_inst_id
	bkInstId := handleBkInstId(dimensions, bkObjectId)
	if bkInstId == 0 {
		return errors.New("invalid bk_inst_id")
	}
	dimensions["bk_inst_id"] = bkInstId

	// 处理bk_biz_id
	if !isK8sObject(bkObjectId) {
		handleBkBizId(dimensions, bkObjectId, bkInstId)
	}

	return nil
}

// isK8sObject 判断是否为k8s对象
func isK8sObject(objectId string) bool {
	return objectId == K8sNodeObjectId ||
		objectId == K8sPodObjectId ||
		objectId == K8sClusterObjectId
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
