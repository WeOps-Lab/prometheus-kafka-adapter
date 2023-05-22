package main

import (
	"fmt"
	"github.com/sirupsen/logrus"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

var weopsOpenApiUrl = fmt.Sprintf("%s/o/%s/open_api", bkAppPaasHost, bkAppWeopsAppId)

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
		bkInstId = getId(fmt.Sprintf("%s/get_k8s_inst_id/?bk_obj_id=%v&bk_inst_name=%s", weopsOpenApiUrl, bkObjId, bkInstName), bkObjId, bkInstName)
		bkCache.Set(bkObjIdInstName, bkInstId, time.Duration(cacheExpiration)*time.Second)
	}
	return bkInstId
}

// getWorkloadID 获取workload ID
func getWorkloadID(podId int) (workloadId int) {
	bkPodIdWkId := fmt.Sprintf("workload@@%v", podId)
	if result, found := bkCache.Get(bkPodIdWkId); found {
		logrus.Debugf("using bkPodIdWkId cache for pod: %v", podId)
		return result.(int)
	} else {
		workloadId = getId(fmt.Sprintf("%s/get_k8s_workload_id/?pod_id=%v", weopsOpenApiUrl, podId), podId)
		bkCache.Set(bkPodIdWkId, workloadId, time.Duration(cacheExpiration)*time.Second)
	}
	return workloadId
}

// getId 获取不同的ID
func getId(url string, instanceIDs ...interface{}) int {
	// 创建一个不带代理的 HTTP Client
	httpClient := &http.Client{
		Transport: &http.Transport{
			Proxy: nil,
		},
	}

	response, err := httpClient.Get(url)
	if err != nil {
		logrus.WithError(err).Errorf("http get url error: %v", url)
	}
	defer response.Body.Close()

	body, err := io.ReadAll(response.Body)
	if err != nil {
		logrus.WithError(err).Errorf("response for instance error: %v", instanceIDs)
	}

	bkInstId, _ := strconv.Atoi(strings.TrimSpace(string(body)))
	return bkInstId
}
