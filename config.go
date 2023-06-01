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
	"fmt"
	_ "github.com/go-sql-driver/mysql"
	"github.com/patrickmn/go-cache"
	dto "github.com/prometheus/client_model/go"
	"github.com/prometheus/common/expfmt"
	"github.com/robfig/cron/v3"
	"gopkg.in/yaml.v2"
	"os"
	"strconv"
	"strings"
	"sync"
	"text/template"
	"time"

	"github.com/sirupsen/logrus"
)

var (
	bkAppWeopsId           = "weops_saas"
	bkAppSecret            = ""
	bkAppMonitorCenterId   = "monitorcenter_saas"
	bkAppPaasHost          = "http://paas.weops.com"
	kafkaBrokerList        = "kafka:9092"
	kafkaTopic             = "metrics"
	topicTemplate          *template.Template
	match                  = make(map[string]*dto.MetricFamily, 0)
	basicauth              = false
	basicauthUsername      = ""
	basicauthPassword      = ""
	kafkaCompression       = "none"
	kafkaBatchNumMessages  = "10000"
	kafkaSslClientCertFile = ""
	kafkaSslClientKeyFile  = ""
	kafkaSslClientKeyPass  = ""
	kafkaSslCACertFile     = ""
	kafkaSecurityProtocol  = ""
	kafkaSaslMechanism     = ""
	kafkaSaslUsername      = ""
	kafkaSaslPassword      = ""
	serializer             Serializer
	bkCache                *cache.Cache
	bkObjRelaCache         *cache.Cache
	bkInstCache            *cache.Cache
	bkSetBizCache          *cache.Cache
	bkObjSetCache          *cache.Cache
	cacheExpiration        = int64(300)
	mutex                  = sync.Mutex{}

	metricsFilePath = "metrics.yaml"
	podWorkloadMap  = make(map[int]int)
	setIdBizIdMap   = make(map[int]int)
	c               = cron.New()
)

func init() {
	logrus.SetFormatter(&logrus.JSONFormatter{})
	logrus.SetOutput(os.Stdout)

	bkCache = cache.New(5*time.Minute, 10*time.Minute)
	bkObjRelaCache = cache.New(5*time.Minute, 10*time.Minute)
	bkInstCache = cache.New(5*time.Minute, 10*time.Minute)
	bkSetBizCache = cache.New(5*time.Minute, 10*time.Minute)
	bkObjSetCache = cache.New(5*time.Minute, 10*time.Minute)

	if value := os.Getenv("BKAPP_WEOPS_APP_ID"); value != "" {
		bkAppWeopsId = value
	}

	if value := os.Getenv("BKAPP_MONITORCENTER_APP_ID"); value != "" {
		bkAppMonitorCenterId = value
	}

	if value := os.Getenv("BKAPP_WEOPS_APP_SECRET"); value != "" {
		bkAppSecret = value
	}

	if value := os.Getenv("BKAPP_PAAS_HOST"); value != "" {
		bkAppPaasHost = value
	}

	if value := os.Getenv("LOG_LEVEL"); value != "" {
		logrus.SetLevel(parseLogLevel(value))
	}

	if value := os.Getenv("KAFKA_BROKER_LIST"); value != "" {
		kafkaBrokerList = value
	}

	if value := os.Getenv("KAFKA_TOPIC"); value != "" {
		kafkaTopic = value
	}

	if value := os.Getenv("BASIC_AUTH_USERNAME"); value != "" {
		basicauth = true
		basicauthUsername = value
	}

	if value := os.Getenv("BASIC_AUTH_PASSWORD"); value != "" {
		basicauthPassword = value
	}

	if value := os.Getenv("KAFKA_COMPRESSION"); value != "" {
		kafkaCompression = value
	}

	if value := os.Getenv("KAFKA_BATCH_NUM_MESSAGES"); value != "" {
		kafkaBatchNumMessages = value
	}

	if value := os.Getenv("KAFKA_SSL_CLIENT_CERT_FILE"); value != "" {
		kafkaSslClientCertFile = value
	}

	if value := os.Getenv("KAFKA_SSL_CLIENT_KEY_FILE"); value != "" {
		kafkaSslClientKeyFile = value
	}

	if value := os.Getenv("KAFKA_SSL_CLIENT_KEY_PASS"); value != "" {
		kafkaSslClientKeyPass = value
	}

	if value := os.Getenv("KAFKA_SSL_CA_CERT_FILE"); value != "" {
		kafkaSslCACertFile = value
	}

	if value := os.Getenv("KAFKA_SECURITY_PROTOCOL"); value != "" {
		kafkaSecurityProtocol = strings.ToLower(value)
	}

	if value := os.Getenv("KAFKA_SASL_MECHANISM"); value != "" {
		kafkaSaslMechanism = value
	}

	if value := os.Getenv("KAFKA_SASL_USERNAME"); value != "" {
		kafkaSaslUsername = value
	}

	if value := os.Getenv("KAFKA_SASL_PASSWORD"); value != "" {
		kafkaSaslPassword = value
	}

	if value := os.Getenv("MATCH"); value != "" {
		matchList, err := parseMatchList(value)
		if err != nil {
			logrus.WithError(err).Fatalln("couldn't parse the match rules")
		}
		match = matchList
	}

	if value := os.Getenv("METRICS_FILE"); value != "" {
		metricsFilePath = value
	}

	var err error
	serializer, err = parseSerializationFormat(os.Getenv("SERIALIZATION_FORMAT"))
	if err != nil {
		logrus.WithError(err).Fatalln("couldn't create a metrics serializer")
	}

	topicTemplate, err = parseTopicTemplate(kafkaTopic)
	if err != nil {
		logrus.WithError(err).Fatalln("couldn't parse the topic template")
	}

	// 缓存时长
	if value := os.Getenv("CACHE_EXPIRATION"); value != "" {
		intValue, err := strconv.ParseInt(value, 10, 64)
		if err != nil {
			logrus.WithError(err).Fatalln("parse cache expiration error")
		}
		cacheExpiration = intValue
	}

	parseK8sMetricsFile(metricsFilePath)

	//初始化获取cmdb全量信息
	setUpCmdbInfo()

	//定时执行获取cmdb全量信息
	_, err = c.AddFunc(fmt.Sprintf("@every %vs", cacheExpiration), setUpCmdbInfo)
	if err != nil {
		logrus.WithError(err).Fatal("set up cmdb info error")
	}
	c.Start()
}

func parseMatchList(text string) (map[string]*dto.MetricFamily, error) {
	var matchRules []string
	err := yaml.Unmarshal([]byte(text), &matchRules)
	if err != nil {
		return nil, err
	}
	var metricsList []string
	for _, v := range matchRules {
		metricsList = append(metricsList, fmt.Sprintf("%s 0\n", v))
	}

	metricsText := strings.Join(metricsList, "")

	var parser expfmt.TextParser
	metricFamilies, err := parser.TextToMetricFamilies(strings.NewReader(metricsText))
	if err != nil {
		return nil, fmt.Errorf("couldn't parse match rules: %s", err)
	}
	return metricFamilies, nil
}

func parseLogLevel(value string) logrus.Level {
	level, err := logrus.ParseLevel(value)

	if err != nil {
		logrus.WithField("log-level-value", value).Warningln("invalid log level from env var, using info")
		return logrus.ErrorLevel
	}

	return level
}

func parseSerializationFormat(value string) (Serializer, error) {
	switch value {
	case "json":
		return NewJSONSerializer()
	case "avro-json":
		return NewAvroJSONSerializer("schemas/metric.avsc")
	default:
		logrus.WithField("serialization-format-value", value).Warningln("invalid serialization format, using json")
		return NewJSONSerializer()
	}
}

func parseTopicTemplate(tpl string) (*template.Template, error) {
	funcMap := template.FuncMap{
		"replace": func(old, new, src string) string {
			return strings.Replace(src, old, new, -1)
		},
		"substring": func(start, end int, s string) string {
			if start < 0 {
				start = 0
			}
			if end < 0 || end > len(s) {
				end = len(s)
			}
			if start >= end {
				panic("template function - substring: start is bigger (or equal) than end. That will produce an empty string.")
			}
			return s[start:end]
		},
	}
	return template.New("topic").Funcs(funcMap).Parse(tpl)
}

// parseK8sMetricsFile 加载k8s指标
func parseK8sMetricsFile(filePath string) {
	yamlFile, err := os.ReadFile(filePath)
	if err != nil {
		logrus.Errorf("Failed to read %v: %v", filePath, err)
	}

	var metrics MetricsFileData

	err = yaml.Unmarshal(yamlFile, &metrics)
	if err != nil {
		logrus.Errorf("Failed to parse YAML %v: %v", filePath, err)
	}

	for _, nodeMetric := range metrics.NodeMetrics {
		K8sNodeMetrics[nodeMetric] = nodeMetric
	}

	for _, podMetric := range metrics.PodMetrics {
		K8sPodMetrics[podMetric] = podMetric
	}
}

func setUpCmdbInfo() {
	var wg sync.WaitGroup
	// TODO: weops接口取对bk_obj_id、data_id
	processObject := func(obj string) {
		defer wg.Done()
		getObjInstInfo(obj)
		getObjSetInfo(obj)
		getDataId(obj)
		if result, found := bkObjSetCache.Get(fmt.Sprintf("obj_set_%s", obj)); found {
			for _, data := range result.(bkObjSetResponse).Data {
				getBizFromSet(data.BkAsstInstId)
			}
		}
	}

	for obj, _ := range objList {
		wg.Add(1)
		go processObject(obj)
	}
	wg.Wait()

	// pod-workload关联
	podWorkloadRel := getRelationId(K8sPodObjectId, K8sWorkloadObjectId)
	for _, eachRel := range podWorkloadRel.Data {
		podWorkloadMap[eachRel.BkInstId] = eachRel.BkAsstInstId
	}
	bkObjRelaCache.Set("pod_workload_rel_map", podWorkloadMap, time.Duration(cacheExpiration)*time.Second)

	// pod、node的biz_id
	k8sBizObjList := []string{K8sNameSpaceObjectId, K8sClusterObjectId}
	for _, obj := range k8sBizObjList {
		wg.Add(1)
		go func(obj string) {
			defer wg.Done()
			res, found := bkObjSetCache.Get(fmt.Sprintf("obj_set_%v", obj))
			if found {
				SetInfo := res.(bkObjSetResponse)
				for _, eachSet := range SetInfo.Data {
					if bizRes, bizFound := bkSetBizCache.Get(fmt.Sprintf("set_id_biz_id@@%v", eachSet.BkAsstInstId)); bizFound {
						BizId := bizRes.(bizResponse).Data.Info[0].BkBizId
						setId := eachSet.BkInstId
						mutex.Lock()
						setIdBizIdMap[setId] = BizId
						mutex.Unlock()
					}
				}
			}
			bkSetBizCache.Set(fmt.Sprintf("%v_set_id_biz_id", obj), setIdBizIdMap, time.Duration(cacheExpiration)*time.Second)
		}(obj)
	}

	wg.Wait()
}
