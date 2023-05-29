// Copyright 2018 Telefónica
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

type instInfoResponse struct {
	Result bool `json:"result"`
	Code   int  `json:"code"`
	Data   struct {
		Count int `json:"count"`
		Info  []struct {
			BkInstId   int    `json:"bk_inst_id"`
			BkInstName string `json:"bk_inst_name"`
			BkObjId    string `json:"bk_obj_id"`
		} `json:"info"`
	} `json:"data"`
	Message    string      `json:"message"`
	Permission interface{} `json:"permission"`
	RequestId  string      `json:"request_id"`
}

type bkObjSetResponse struct {
	Result bool `json:"result"`
	Code   int  `json:"code"`
	Data   []struct {
		Id                int    `json:"id"`
		BkInstId          int    `json:"bk_inst_id"`
		BkObjId           string `json:"bk_obj_id"`
		BkAsstInstId      int    `json:"bk_asst_inst_id"`
		BkAsstObjId       string `json:"bk_asst_obj_id"`
		BkSupplierAccount string `json:"bk_supplier_account"`
		BkObjAsstId       string `json:"bk_obj_asst_id"`
		BkAsstId          string `json:"bk_asst_id"`
	} `json:"data"`
	Message    string      `json:"message"`
	Permission interface{} `json:"permission"`
	RequestId  string      `json:"request_id"`
}

type bizResponse struct {
	Result bool `json:"result"`
	Code   int  `json:"code"`
	Data   struct {
		Count int `json:"count"`
		Info  []struct {
			BkBizId   int    `json:"bk_biz_id"`
			BkSetName string `json:"bk_set_name"`
			Default   int    `json:"default"`
		} `json:"info"`
	} `json:"data"`
	Message    string      `json:"message"`
	Permission interface{} `json:"permission"`
	RequestId  string      `json:"request_id"`
}

type relationGroupResponse struct {
	Result bool `json:"result"`
	Code   int  `json:"code"`
	Data   []struct {
		Id                int    `json:"id"`
		BkInstId          int    `json:"bk_inst_id"`
		BkObjId           string `json:"bk_obj_id"`
		BkAsstInstId      int    `json:"bk_asst_inst_id"`
		BkAsstObjId       string `json:"bk_asst_obj_id"`
		BkSupplierAccount string `json:"bk_supplier_account"`
		BkObjAsstId       string `json:"bk_obj_asst_id"`
		BkAsstId          string `json:"bk_asst_id"`
	} `json:"data"`
	Message    string      `json:"message"`
	Permission interface{} `json:"permission"`
	RequestId  string      `json:"request_id"`
}
