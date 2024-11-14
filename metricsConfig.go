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

import "time"

const (
	Protocol             = "protocol"
	Source               = "source"
	Automate             = "automate"
	Kubernetes           = "kubernetes"
	SNMP                 = "snmp"
	IPMI                 = "ipmi"
	Vector               = "vector"
	CLOUD                = "cloud"
	K8sPodObjectId       = "k8s_pod"
	K8sNodeObjectId      = "bk_node"
	K8sClusterObjectId   = "k8s_cluster"
	K8sNameSpaceObjectId = "k8s_namespace"
	K8sWorkloadObjectId  = "k8s_workload"
)

var objList = map[string]bool{
	K8sPodObjectId:       true,
	K8sClusterObjectId:   true,
	K8sNodeObjectId:      true,
	K8sWorkloadObjectId:  true,
	K8sNameSpaceObjectId: true,
	"bk_switch":          true,
	"bk_router":          true,
	"bk_firewall":        true,
	"bk_loadbalance":     true,
	"hard_server":        true,
}

// K8sPodMetrics k8s容器指标
var K8sPodMetrics = make(map[string]string)
var k8sPodDimension = map[string]bool{
	Protocol:        true,
	"bk_data_id":    true,
	"bk_inst_id":    true,
	"bk_obj_id":     true,
	"cluster":       true,
	"instance_name": true,
	"namespace_id":  true,
	"node_id":       true,
	"pod_id":        true,
	"pod":           true,
	"workload":      true,
}

var K8sPodStatusPhaseMap = map[string]float64{
	"Failed":    float64(0),
	"Running":   float64(1),
	"Pending":   float64(2),
	"Succeeded": float64(3),
	"Unknown":   float64(4),
}

var commonDimensionFilter = map[string]bool{
	// influx保留字段
	"name":        true,
	"user":        true,
	"users":       true,
	"queries":     true,
	"database":    true,
	"databases":   true,
	"field":       true,
	"group":       true,
	"groups":      true,
	"info":        true,
	"offset":      true,
	"replication": true,
	"values":      true,
	"shard":       true,
	"tag":         true,
	"__name__":    true,
}

// K8sNodeMetrics k8s节点指标
var K8sNodeMetrics = make(map[string]string)
var k8sNodeDimension = map[string]bool{
	Protocol:        true,
	"bk_data_id":    true,
	"bk_inst_id":    true,
	"bk_obj_id":     true,
	"cluster":       true,
	"instance_name": true,
	"node_id":       true,
}

// K8sClusterMetrics cluster自定义指标
var K8sClusterMetrics = make(map[string]string)

// TelegrafIpmiMetrics telegraf采集的ipmi指标
var TelegrafIpmiMetrics = make(map[string]string)

var K8sNodeStatusConditionMap = map[string]float64{
	"false":   float64(0),
	"true":    float64(1),
	"unknown": float64(2),
}

type MetricsData struct {
	Data []struct {
		Dimension map[string]interface{} `json:"dimension"`
		Metrics   map[string]float64     `json:"metrics"`
		Timestamp int64                  `json:"timestamp"`
	} `json:"data"`
}

// BKMetricsData 蓝鲸自有链路指标格式
type BKMetricsData struct {
	Timestamp  time.Time  `json:"@timestamp"`
	BkBizId    int        `json:"bk_biz_id"`
	BkCloudId  int        `json:"bk_cloud_id"`
	DataId     int        `json:"dataid"`
	GroupInfo  GroupInfo  `json:"group_info"`
	Prometheus Prometheus `json:"prometheus"`
	Service    string     `json:"service"`
	Type       string     `json:"type"`
}

// Prometheus represents the Prometheus data structure
type Prometheus struct {
	Collector Collector `json:"collector"`
}

// Collector represents the Collector data structure
type Collector struct {
	Metrics Metrics `json:"metrics"`
}

type GroupInfo []struct {
	BkCollectConfigId string `json:"bk_collect_config_id"`
}

// Metrics represents the Metrics data structure
type Metrics []struct {
	Key       string                 `json:"key"`
	Labels    map[string]interface{} `json:"labels"`
	Timestamp int64                  `json:"timestamp"`
	Value     float64                `json:"value"`
}

type MetricsFileData struct {
	NodeMetrics         map[string]string `yaml:"K8sNodeMetrics"`
	PodMetrics          map[string]string `yaml:"K8sPodMetrics"`
	CLusterMetrics      map[string]string `yaml:"K8sClusterMetrics"`
	TelegrafIpmiMetrics map[string]string `yaml:"TelegrafIpmiMetrics"`
}
