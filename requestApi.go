package main

import (
	"encoding/json"
	"fmt"
	"github.com/sirupsen/logrus"
	"io"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"
)

var weopsOpenApiUrl = fmt.Sprintf("%s/o/%s/open_api", bkAppPaasHost, bkAppWeopsId)
var monitorCenterOpenApiUrl = fmt.Sprintf("%s/o/%s/open_api", bkAppPaasHost, bkAppMonitorCenterId)

// getBizId 获取CMDB业务ID
func getBkBizId(bkObjId string, bkInstId int) (bkBizId int) {
	bkObjIdBizId := fmt.Sprintf("%s@@%s", bkObjId, bkInstId)
	if result, found := bkCache.Get(bkObjIdBizId); found {
		logrus.Debugf("using bkObjIdBizId cache for object: %v, inst id: %v", bkObjId, bkInstId)
		return result.(int)
	} else {
		bkBizId = getId(fmt.Sprintf("%s/get_inst_biz_id/?bk_obj_id=%v&bk_inst_id=%v", weopsOpenApiUrl, bkObjId, bkInstId), bkObjId, bkInstId)
		bkCache.Set(bkObjIdBizId, bkBizId, time.Duration(cacheExpiration)*time.Second)
	}
	return
}

// getK8sBkInstId 获取CMDB中k8s实例ID
func getK8sBkInstId(bkObjId, bkInstName string) (bkInstId int) {
	bkObjIdInstName := fmt.Sprintf("%s@@%s", bkObjId, bkInstName)
	if result, found := bkCache.Get(bkObjIdInstName); found {
		logrus.Debugf("using bkObjIdInstName cache for object: %v, inst name: %v", bkObjId, bkInstName)
		return result.(int)
	} else {
		bkInstId = getId(fmt.Sprintf("%s/get_k8s_inst_id/?bk_obj_id=%v&bk_inst_name=%v", weopsOpenApiUrl, bkObjId, bkInstName), bkObjId, bkInstName)
		bkCache.Set(bkObjIdInstName, bkInstId, time.Duration(cacheExpiration)*time.Second)
	}
	return bkInstId
}

// getWorkloadID 获取workload ID
func getWorkloadID(instanceName string, podId int) (workloadId int) {
	bkPodIdWkId := fmt.Sprintf("%v@@%v", instanceName, podId)
	if result, found := bkCache.Get(bkPodIdWkId); found {
		logrus.Debugf("using bkPodIdWkId cache for instance: %v, pod: %v", instanceName, podId)
		return result.(int)
	} else {
		workloadId = getId(fmt.Sprintf("%s/get_k8s_workload_id/?pod_id=%v", weopsOpenApiUrl, podId), podId)
		bkCache.Set(bkPodIdWkId, workloadId, time.Duration(cacheExpiration)*time.Second)
	}
	return workloadId
}

// getId 获取不同的ID
func getId(url string, instanceIDs ...interface{}) (bkInstId int) {
	httpClient := createHTTPClient()
	body, err := sendHTTPRequest(url, httpClient, instanceIDs...)
	if err != nil {
		logrus.WithError(err).Errorf("response for instance error: %v", instanceIDs)
		return 0
	}

	stringBody := strings.TrimSpace(string(body))
	if stringBody == "" {
		logrus.Debugf("empty value: %v, %v", instanceIDs, url)
		return 0
	} else {
		bkInstId, err = strconv.Atoi(stringBody)
		if err != nil {
			logrus.WithError(err).Errorf("parse body to bkInstId error: %v", instanceIDs)
			return 0
		}

	}

	return bkInstId
}

// requestDataId 获取监控对象data id
func requestDataId(bkObjectId string) (bkDataId string) {
	url := fmt.Sprintf("%s/get_obj_table_id/?bk_obj_id=%v", monitorCenterOpenApiUrl, bkObjectId)
	httpClient := createHTTPClient()

	body, err := sendHTTPRequest(url, httpClient, bkObjectId)
	if err != nil {
		logrus.WithError(err).Errorf("response for data id error: %v", bkObjectId)
	}

	var result struct {
		Result bool   `json:"result"`
		Data   string `json:"data"`
	}

	err = json.Unmarshal(body, &result)
	if err == nil && result.Result {
		re := regexp.MustCompile(`\d+`)
		matches := re.FindAllString(result.Data, -1)
		if len(matches) > 0 {
			bkDataId = matches[len(matches)-1]
		}
	}

	return bkDataId
}

// createHTTPClient 创建一个不带代理的 HTTP Client
func createHTTPClient() *http.Client {
	return &http.Client{
		Transport: &http.Transport{
			Proxy: nil,
		},
	}
}

// sendHTTPRequest 发送 HTTP 请求并返回响应体内容
func sendHTTPRequest(url string, httpClient *http.Client, logParams ...interface{}) (body []byte, err error) {
	response, err := httpClient.Get(url)
	if err != nil {
		logrus.WithError(err).Errorf("sendHTTPRequest http get url error: %v", logParams...)
		return nil, err
	}
	defer response.Body.Close()

	if response.StatusCode == http.StatusOK {
		body, err = io.ReadAll(response.Body)
		if err != nil {
			logrus.WithError(err).Errorf("sendHTTPRequest response error: %v", logParams...)
			return nil, err
		}
	}
	return body, nil
}
