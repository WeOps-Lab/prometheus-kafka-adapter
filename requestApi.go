package main

import (
	"encoding/json"
	"fmt"
	"github.com/sirupsen/logrus"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

var weopsOpenApiUrl = fmt.Sprintf("%s/o/%s/open_api", bkAppPaasHost, bkAppWeopsId)
var bkApi = fmt.Sprintf("%s/api/c/compapi/v2/cc", bkAppPaasHost)

const (
	searchInst  = "search_inst"
	findInstAss = "find_instance_association"
)

// getBizId 获取CMDB业务ID
func getBkBizId(bkObjId string, bkInstId int) (bkBizId int) {
	for _, data := range getObjSetInfo(bkObjId).Data {
		if data.BkInstId == bkInstId {
			setId := data.BkAsstInstId
			return getBizFromSet(setId).Data.Info[0].BkBizId
		}
	}
	return 0
}

// 获取实例id
func getBkInstId(bkObjId, bkInstName string) int {
	if result, found := bkInstCache.Get(fmt.Sprintf("obj_inst_%v", bkObjId)); found {
		if instId, ok := result.(map[string]int)[bkInstName]; ok {
			return instId
		}
	}
	return 0
}

// requestDataId 获取监控对象data id
func requestDataId() map[string]string {
	httpClient := createHTTPClient()
	body, err := sendHTTPRequest(fmt.Sprintf("%s/get_all_data_id", weopsOpenApiUrl), httpClient)
	if err != nil {
		logrus.WithError(err).Errorf("response for get_all_data_id error")
	}

	var result AllObjDataIdResponse
	bkObjDataId := make(map[string]string, 0)
	err = json.Unmarshal(body, &result)
	if err == nil && result.Result {
		for _, info := range result.Data {
			bkObjDataId[info.BkObjId] = strconv.Itoa(info.BkDataId)
		}
	}

	return bkObjDataId
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

// 获取CMDB中所有的实例信息
func getObjInstResp(start int, bkObjId string) (bkInstInfo instInfoResponse) {
	payload := strings.NewReader(fmt.Sprintf(`{
			"bk_app_code": "%v",
			"bk_app_secret": "%v",
			"bk_username": "admin",
			"bk_obj_id": "%v",
			"fields": {
				"bk_switch": [
					"bk_inst_id",
					"bk_inst_name",
					"bk_obj_id"
				]
			},
			"page": {
				"start": %v,
				"limit": 200,
				"sort": "bk_inst_id"
			}
		}`, bkAppWeopsId, bkAppSecret, bkObjId, start))

	instResponse, err := cmdbPostApi(bkObjId, searchInst, payload)
	if err != nil {
		logrus.WithError(err).Errorf("get inst info postHttpRequest error for object: %v", bkObjId)
		return instInfoResponse{}
	}

	err = json.Unmarshal(instResponse, &bkInstInfo)
	if err != nil {
		logrus.WithError(err).Errorf("get inst info json Unmarshal error for object: %v", bkObjId)
		return
	}
	return bkInstInfo
}

func getObjInstInfo(bkObjId string) {
	bkObjInst := fmt.Sprintf("obj_inst_%s", bkObjId)
	response := getObjInstResp(0, bkObjId)
	instMap := make(map[string]int)

	for _, inst := range response.Data.Info {
		instMap[inst.BkInstName] = inst.BkInstId
	}

	allInstCount := response.Data.Count
	start := len(response.Data.Info)
	for start < allInstCount {
		response = getObjInstResp(start, bkObjId)
		for _, inst := range response.Data.Info {
			instMap[inst.BkInstName] = inst.BkInstId
		}
		start += len(response.Data.Info)
	}

	bkInstCache.Set(bkObjInst, instMap, time.Duration(cacheExpiration)*time.Second)
}

// 获取对象的全部set_id
func getObjSetInfo(bkObjId string) (bkSetInfo bkObjSetResponse) {
	bkObjSet := fmt.Sprintf("obj_set_%s", bkObjId)
	if result, found := bkObjSetCache.Get(bkObjSet); found {
		return result.(bkObjSetResponse)
	} else {
		payload := strings.NewReader(fmt.Sprintf(`
		{
			"bk_app_code": "%v",
			"bk_app_secret": "%v",
			"bk_username": "admin",
			"bk_obj_id": "%v",
			"condition": {
				"bk_obj_asst_id": "%v_group_set"
			}
		}`, bkAppWeopsId, bkAppSecret, bkObjId, bkObjId))

		instResponse, err := cmdbPostApi(bkObjId, findInstAss, payload)
		if err != nil {
			logrus.WithError(err).Errorf("get object set info for object: %v", bkObjId)
			return
		}

		err = json.Unmarshal(instResponse, &bkSetInfo)
		if err != nil {
			logrus.WithError(err).Errorf("getInstInfo json Unmarshal error for object: %v", bkObjId)
			return
		}
		bkObjSetCache.Set(bkObjSet, bkSetInfo, time.Duration(cacheExpiration)*time.Second)
	}
	return bkSetInfo
}

// 获取关联对象id
func getRelationId(bkObjId, bkObjAssId string) (bkRelInstInfo relationGroupResponse) {
	bkRelInst := fmt.Sprintf("obj_rela_%v_group_%v", bkObjId, bkObjAssId)
	if result, found := bkObjRelaCache.Get(bkRelInst); found {
		return result.(relationGroupResponse)
	} else {
		payload := strings.NewReader(fmt.Sprintf(`
	{
		"bk_app_code": "%v",
		"bk_app_secret": "%v",
		"bk_username": "admin",
		"bk_obj_id": "%v",
		"condition": {
			"bk_obj_asst_id": "%v_group_%v",
			"bk_asst_id": "group",
			"bk_asst_obj_id": "%v"
		}
	}`, bkAppWeopsId, bkAppSecret, bkObjId, bkObjId, bkObjAssId, bkObjAssId))

		instResponse, err := cmdbPostApi(bkObjId, findInstAss, payload)
		if err != nil {
			logrus.WithError(err).Errorf("get object set info for object: %v", bkObjId)
			return
		}

		err = json.Unmarshal(instResponse, &bkRelInstInfo)
		if err != nil {
			logrus.WithError(err).Errorf("getInstInfo json Unmarshal error for object: %v", bkObjId)
			return
		}
		bkObjRelaCache.Set(bkRelInst, bkRelInstInfo, time.Duration(cacheExpiration)*time.Second)
	}
	return bkRelInstInfo
}

func getBizFromSet(setId int) (bizInfo bizResponse) {
	bkObjSet := fmt.Sprintf("set_id_biz_id@@%v", setId)
	if result, found := bkSetBizCache.Get(bkObjSet); found {
		return result.(bizResponse)
	} else {
		payload := strings.NewReader(fmt.Sprintf(`
			{
			  "bk_app_code": "%v",
			  "bk_app_secret": "%v",
			  "bk_username": "admin",
			  "bk_obj_id": "set",
			  "fields": {
				"set": [
				  "bk_biz_id"
				]
			  },
			  "condition": {
				"set": [
				  {
					"field": "bk_set_id",
					"operator": "$eq",
					"value": %v
				  }
				]
			  }
			}`, bkAppWeopsId, bkAppSecret, setId))

		bizBodyResponse, err := cmdbPostApi("", searchInst, payload)
		if err != nil {
			logrus.WithError(err).Errorf("get biz id postHttpRequest error for set id: %v", setId)
			return
		}

		err = json.Unmarshal(bizBodyResponse, &bizInfo)
		if err != nil {
			logrus.WithError(err).Errorf("get biz info json Unmarshal error for set id: %v", setId)
			return
		}
		bkSetBizCache.Set(bkObjSet, bizInfo, time.Duration(cacheExpiration)*time.Second)
	}
	return bizInfo
}

func cmdbPostApi(bkObjId, apiName string, payload *strings.Reader) ([]byte, error) {
	instAssResponse, err := postHttpRequest(fmt.Sprintf("%v/%v", bkApi, apiName), payload)
	if err != nil {
		logrus.WithError(err).Errorf("find instance association error for object: %v", bkObjId)
		return nil, err
	}

	return instAssResponse, nil
}

// post请求
func postHttpRequest(url string, payload *strings.Reader) (body []byte, err error) {
	httpClient := createHTTPClient()
	response, err := httpClient.Post(url, "application/json", payload)
	if err != nil {
		logrus.WithError(err).Errorf("post http request error %v", url)
		return nil, err
	}
	defer response.Body.Close()

	if response.StatusCode == http.StatusOK {
		body, err = io.ReadAll(response.Body)
		if err != nil {
			logrus.WithError(err).Errorf("post httpRequest response error")
			return nil, err
		}
	}
	return body, nil
}
