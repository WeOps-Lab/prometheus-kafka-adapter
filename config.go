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
	"database/sql"
	"fmt"
	_ "github.com/go-sql-driver/mysql"
	"github.com/patrickmn/go-cache"
	dto "github.com/prometheus/client_model/go"
	"github.com/prometheus/common/expfmt"
	"gopkg.in/yaml.v2"
	"os"
	"strconv"
	"strings"
	"text/template"
	"time"

	"github.com/sirupsen/logrus"
)

var (
	bkAppWeopsAppId        = "weops_saas"
	bkAppPaasHost          = "http://paas.weops.com"
	weopsDbUser            = "weops"
	weopsDbPass            = "Weops123!"
	weopsDbHost            = "127.0.0.1"
	weopsDbPort            = "3306"
	weopsDbName            = "monitorcenter_saas"
	db                     *sql.DB // 全局变量，保存数据库连接
	dsn                    = ""
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
	cacheExpiration        = int64(300)

	metricsFilePath = "metrics.yaml"
)

func init() {
	logrus.SetFormatter(&logrus.JSONFormatter{})
	logrus.SetOutput(os.Stdout)

	bkCache = cache.New(5*time.Minute, 10*time.Minute)

	if value := os.Getenv("BKAPP_WEOPS_APP_ID"); value != "" {
		bkAppWeopsAppId = value
	}

	if value := os.Getenv("BKAPP_PAAS_HOST"); value != "" {
		bkAppPaasHost = value
	}

	if value := os.Getenv("WEOPS_DB_USER"); value != "" {
		weopsDbUser = value
	}

	if value := os.Getenv("WEOPS_DB_PASSWORD"); value != "" {
		weopsDbPass = value
	}

	if value := os.Getenv("WEOPS_DB_HOST"); value != "" {
		weopsDbHost = value
	}

	if value := os.Getenv("WEOPS_DB_PORT"); value != "" {
		weopsDbPort = value
	}

	if value := os.Getenv("WEOPS_DB_DATABASENAME"); value != "" {
		weopsDbName = value
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

	dsn = fmt.Sprintf("%s:%s@tcp(%s:%s)/%s", weopsDbUser, weopsDbPass, weopsDbHost, weopsDbPort, weopsDbName)
	// 连接数据库
	db, err = sql.Open("mysql", dsn)
	if err != nil {
		logrus.WithError(err).Fatalln("couldn't connect to mysql")
	}

	db.SetConnMaxLifetime(time.Second * 1800)
	db.SetMaxOpenConns(30)
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
		return logrus.InfoLevel
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
