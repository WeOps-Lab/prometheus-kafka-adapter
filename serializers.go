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

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/linkedin/goavro"
	"github.com/prometheus/prometheus/prompb"
	"github.com/sirupsen/logrus"
	"io/ioutil"
)

// Serializer represents an abstract metrics serializer
type Serializer interface {
	Marshal(metric map[string]interface{}) ([]byte, error)
}

// Serialize generates the JSON representation for a given Prometheus metric.
func Serialize(s Serializer, req *prompb.WriteRequest) (map[string][][]byte, error) {
	promBatches.Add(float64(1))
	result := make(map[string][][]byte)

	for _, ts := range req.Timeseries {
		labels := make(map[string]string, len(ts.Labels))

		for _, l := range ts.Labels {
			labels[l.Name] = l.Value
		}

		// 必须带有protocol字段才会当做处理指标
		if _, protocolExist := labels[Protocol]; !protocolExist {
			continue
		}

		metricName := labels["__name__"]

		// 提取维度信息
		dimensions := make(map[string]interface{})

		// 过滤指标
		if labels[Protocol] == Kubernetes && k8sMetricsPreHandler(labels) {
			dimensions = fillUpBkInfo(labels)
		} else {
			continue
		}

		// 丢弃业务id和实例id为0的指标
		if dimensions["bk_inst_id"] == 0 || dimensions["bk_biz_id"] == 0 {
			continue
		}

		if dimensions["bk_data_id"] != "" {
			kafkaTopic = fmt.Sprintf("0bkmonitor_%v0", dimensions["bk_data_id"])
		} else {
			kafkaTopic = topic(labels)
		}

		for _, sample := range ts.Samples {
			if !filter(metricName, labels) {
				objectsFiltered.Add(float64(1))
				continue
			}

			// 数据清洗
			data, err := processData(metricName, dimensions, sample)
			if err != nil {
				serializeFailed.Add(float64(1))
				logrus.WithError(err).Errorln("couldn't marshal timeseries")
			}

			serializeTotal.Add(float64(1))
			result[kafkaTopic] = append(result[kafkaTopic], data)
		}
	}

	return result, nil
}

// JSONSerializer represents a metrics serializer that writes JSON
type JSONSerializer struct {
}

func (s *JSONSerializer) Marshal(metric map[string]interface{}) ([]byte, error) {
	return json.Marshal(metric)
}

func NewJSONSerializer() (*JSONSerializer, error) {
	return &JSONSerializer{}, nil
}

// AvroJSONSerializer represents a metrics serializer that writes Avro-JSON
type AvroJSONSerializer struct {
	codec *goavro.Codec
}

func (s *AvroJSONSerializer) Marshal(metric map[string]interface{}) ([]byte, error) {
	return s.codec.TextualFromNative(nil, metric)
}

// NewAvroJSONSerializer builds a new instance of the AvroJSONSerializer
func NewAvroJSONSerializer(schemaPath string) (*AvroJSONSerializer, error) {
	schema, err := ioutil.ReadFile(schemaPath)
	if err != nil {
		logrus.WithError(err).Errorln("couldn't read avro schema")
		return nil, err
	}

	codec, err := goavro.NewCodec(string(schema))
	if err != nil {
		logrus.WithError(err).Errorln("couldn't create avro codec")
		return nil, err
	}

	return &AvroJSONSerializer{
		codec: codec,
	}, nil
}

func topic(labels map[string]string) string {
	var buf bytes.Buffer
	if err := topicTemplate.Execute(&buf, labels); err != nil {
		return ""
	}
	return buf.String()
}

func filter(name string, labels map[string]string) bool {
	if len(match) == 0 {
		return true
	}
	mf, ok := match[name]
	if !ok {
		return false
	}

	for _, m := range mf.Metric {
		if len(m.Label) == 0 {
			return true
		}

		labelMatch := true
		for _, label := range m.Label {
			val, ok := labels[label.GetName()]
			if !ok || val != label.GetValue() {
				labelMatch = false
				break
			}
		}

		if labelMatch {
			return true
		}
	}
	return false
}
