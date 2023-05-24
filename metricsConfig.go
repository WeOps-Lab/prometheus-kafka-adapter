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

const (
	Protocol             = "protocol"
	Kubernetes           = "kubernetes"
	SNMP                 = "snmp"
	IPMI                 = "ipmi"
	K8sPodObjectId       = "k8s_pod"
	K8sNodeObjectId      = "bk_node"
	K8sClusterObjectId   = "k8s_cluster"
	K8sNameSpaceObjectId = "k8s_namespace"
)

// K8sPodMetrics k8s容器指标
var K8sPodMetrics = make(map[string]string)
var K8sPodDimension = map[string]bool{
	"bk_data_id":    true,
	"bk_biz_id":     true,
	"bk_inst_id":    true,
	"bk_obj_id":     true,
	"cluster":       true,
	"instance_name": true,
	"namespace_id":  true,
	"node_id":       true,
	"pod_id":        true,
	"workload":      true,
}
var CommonDimensionFilter = map[string]bool{
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
}

// K8sNodeMetrics k8s节点指标
var K8sNodeMetrics = make(map[string]string)
var K8sNodeDimension = map[string]bool{
	"bk_data_id":    true,
	"bk_biz_id":     true,
	"bk_inst_id":    true,
	"bk_obj_id":     true,
	"cluster":       true,
	"instance_name": true,
	"node_id":       true,
}

type MetricsData struct {
	Data []struct {
		Dimension map[string]interface{} `json:"dimension"`
		Metrics   map[string]float64     `json:"metrics"`
		Timestamp int64                  `json:"timestamp"`
	} `json:"data"`
}

type MetricsFileData struct {
	NodeMetrics []string `yaml:"K8sNodeMetrics"`
	PodMetrics  []string `yaml:"K8sPodMetrics"`
}
