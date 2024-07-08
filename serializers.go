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
	"errors"
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
			logrus.WithField("Metrics excluding the protocol dimension: ", labels).Debugln()
			weopsProtocolMetricsFiltered.Add(float64(1))
			continue
		} else {
			weopsProtocolMetricsInputed.WithLabelValues(labels[Protocol]).Add(float64(1))
		}

		// 提取维度信息
		dimensions := make(map[string]interface{})

		// 过滤指标
		if !shouldProcess(labels) {
			logrus.WithField("The metrics do not meet the WEOPS conditions: ", labels).Debugln()
			weopsMetricsFiltered.WithLabelValues(labels[Protocol]).Add(float64(1))
			continue
		}

		dimensions = fillUpBkInfo(labels)
		if dimensions == nil {
			logrus.WithField("Dropping metrics because of null dimension: ", labels).Debugln()
			weopsMetricsDropped.WithLabelValues(labels[Protocol], labels["__name__"]).Add(float64(1))
			continue
		}

		t, err := getTopic(dimensions)
		if err != nil {
			logrus.WithField("Empty data id: ", labels).Debugln()
			continue
		}

		for _, sample := range ts.Samples {
			if !filter(labels["__name__"], labels) {
				objectsFiltered.Add(float64(1))
				continue
			}

			bkSource := dimensions["protocol"] == Vector
			delete(dimensions, Protocol)
			deleteUselessDimension(&dimensions, commonDimensionFilter, false)

			data, err := formatMetricsData(labels["__name__"], dimensions, sample, bkSource)
			if err != nil {
				serializeFailed.Add(float64(1))
				logrus.WithError(err).Errorln("couldn't marshal timeseries")
				continue
			}

			serializeTotal.Add(float64(1))
			weopsTopicSerializeTotal.WithLabelValues(t).Add(float64(1))
			result[t] = append(result[t], data)
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

// shouldProcess 判断是否需要处理该指标
func shouldProcess(labels map[string]string) bool {
	return (labels[Protocol] == Kubernetes && k8sMetricsPreHandler(labels)) || labels[Protocol] == SNMP || labels[Protocol] == IPMI || labels[Source] == Automate || labels[Protocol] == Vector
}

// getTopic 提取topic并删除无用的维度信息
func getTopic(dimensions map[string]interface{}) (string, error) {
	if dataID, ok := dimensions["bk_data_id"].(string); ok && dataID != "" {
		t := fmt.Sprintf("0bkmonitor_%v0", dataID)
		for _, key := range []string{"bk_data_id", "job"} {
			delete(dimensions, key)
		}
		return t, nil
	}
	return "", errors.New("dataID is empty or not found")
}
