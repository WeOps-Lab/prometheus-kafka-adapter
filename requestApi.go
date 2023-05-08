package main

import (
	"fmt"
	"github.com/sirupsen/logrus"
	"io"
	"net/http"
	"strconv"
	"strings"
)

var weopsOpenApiUrl = fmt.Sprintf("%s/o/%s/open_api", bkAppPaasHost, bkAppWeopsAppId)

// getBizId 获取CMDB业务ID
func getBkBizId(bkObjId string, bkInstId int) int {
	return getId(fmt.Sprintf("%s/get_inst_biz_id/?bk_obj_id=%v&bk_inst_id=%s", weopsOpenApiUrl, bkObjId, bkInstId), bkObjId, bkInstId)
}

// getBkInstId 获取CMDB实例ID
func getBkInstId(bkObjId, bkInstName string) int {
	return getId(fmt.Sprintf("%s/get_k8s_inst_id/?bk_obj_id=%v&bk_inst_name=%s", weopsOpenApiUrl, bkObjId, bkInstName), bkObjId, bkInstName)
}

// getWorkloadID 获取workload ID
func getWorkloadID(podId int, podName string) int {
	return getId(fmt.Sprintf("%s/get_k8s_workload_id/?pod_id=%d", weopsOpenApiUrl, podId), podId, podName)
}

// getId 获取不同的ID
func getId(url string, instanceIDs ...interface{}) int {
	// 创建一个不带代理的 HTTP Client
	httpClient := &http.Client{
		Transport: &http.Transport{
			Proxy: nil,
		},
	}
	// 发起 HTTP 请求
	response, err := httpClient.Get(url)
	if err != nil {
		logrus.WithError(err).Errorf("查询不到实例: %v", instanceIDs)
	}
	defer response.Body.Close()

	body, err := io.ReadAll(response.Body)
	if err != nil {
		logrus.WithError(err).Errorf("读取实例响应时出错: %v", instanceIDs)
	}

	bkInstId, _ := strconv.Atoi(strings.TrimSpace(string(body)))

	//if bkInstId == 0 {
	//logrus.WithError(err).Errorf("查询到实例ID为0: %v", instanceIDs)
	//}

	return bkInstId
}
