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
	Protocol   = "protocol"
	Kubernetes = "kubernetes"
	SNMP       = "snmp"
	Ipmi       = "ipmi"

	K8sPodObjectId  = "k8s_pod"
	K8sNodeObjectId = "bk_node"
)

var BkObjDataIdMap = make(map[string]int)

type CommonMetrics struct {
	Name     string `json:"name"`
	Protocol string `json:"protocol"`
	Instance string `json:"Instance"`
}

// K8sPodMetrics k8s容器指标
var K8sPodMetrics = map[string]string{
	"pod_cpu_utilization":                      "pod_cpu_utilization",
	"container_cpu_utilization":                "container_cpu_utilization",
	"pod_memory_usage":                         "pod_memory_usage",
	"container_memory_usage_bytes":             "container_memory_usage",
	"pod_memory_utilization":                   "pod_memory_utilization",
	"container_memory_utilization":             "container_memory_utilization",
	"pod_network_receive":                      "pod_network_receive",
	"pod_network_transmit":                     "pod_network_transmit",
	"pod_cpu_load":                             "pod_cpu_load",
	"container_cpu_load_average_10s":           "container_cpu_load",
	"kube_pod_start_time":                      "pod_start_time_seconds",
	"kube_pod_status_phase":                    "kube_pod_status_phase",
	"kube_pod_container_status_restarts_total": "kube_pod_container_status_restarts_total",
}

// K8sNodeMetrics k8s节点指标
var K8sNodeMetrics = map[string]string{
	"node_cpu_utilization":             "node_app_memory_usage",
	"node_app_memory_usage_bytes":      "node_app_memory_usage",
	"node_app_memory_utilization":      "node_app_memory_utilization",
	"node_physical_memory_usage_bytes": "node_physical_memory_usage",
	"node_physical_memory_utilization": "node_physical_memory_utilization",
	"node_disk_io_now":                 "node_io_current",
	"node_network_receive":             "node_network_receive",
	"node_network_transmit":            "node_network_transmit",
	"node_load1":                       "node_cpu_load1",
	"node_load5":                       "node_cpu_load5",
	"node_load15":                      "node_cpu_load15",
	"node_filesystem_usage_bytes":      "node_filesystem_usage",
	"node_filesystem_avail_bytes":      "node_filesystem_free",
	"node_filesystem_utilization":      "node_filesystem_utilization",
	"kube_node_status_condition":       "kube_node_status_condition",
}

type MetricsData struct {
	Data []struct {
		Dimension map[string]interface{} `json:"dimension"`
		Metrics   map[string]float64     `json:"metrics"`
		Timestamp int64                  `json:"timestamp"`
	} `json:"data"`
}
