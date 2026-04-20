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
	"net/http"
	_ "net/http/pprof"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sirupsen/logrus"
)

// setupPprofRoutes 设置pprof路由，根据pprofEnabled状态决定是否启用
func setupPprofRoutes(r *gin.Engine) {
	// pprof middleware - 检查是否启用
	pprofMiddleware := func() gin.HandlerFunc {
		return func(c *gin.Context) {
			if !pprofEnabled {
				c.JSON(http.StatusForbidden, gin.H{"error": "pprof is disabled"})
				c.Abort()
				return
			}
			c.Next()
		}
	}

	// pprof路由组
	pprofGroup := r.Group("/debug/pprof")
	pprofGroup.Use(pprofMiddleware())
	{
		pprofGroup.GET("/", gin.WrapH(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.Redirect(w, r, "/debug/pprof/", http.StatusMovedPermanently)
		})))
		pprofGroup.GET("/cmdline", gin.WrapH(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.DefaultServeMux.ServeHTTP(w, r)
		})))
		pprofGroup.GET("/profile", gin.WrapH(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.DefaultServeMux.ServeHTTP(w, r)
		})))
		pprofGroup.POST("/symbol", gin.WrapH(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.DefaultServeMux.ServeHTTP(w, r)
		})))
		pprofGroup.GET("/symbol", gin.WrapH(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.DefaultServeMux.ServeHTTP(w, r)
		})))
		pprofGroup.GET("/trace", gin.WrapH(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.DefaultServeMux.ServeHTTP(w, r)
		})))
		pprofGroup.GET("/allocs", gin.WrapH(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.DefaultServeMux.ServeHTTP(w, r)
		})))
		pprofGroup.GET("/block", gin.WrapH(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.DefaultServeMux.ServeHTTP(w, r)
		})))
		pprofGroup.GET("/goroutine", gin.WrapH(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.DefaultServeMux.ServeHTTP(w, r)
		})))
		pprofGroup.GET("/heap", gin.WrapH(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.DefaultServeMux.ServeHTTP(w, r)
		})))
		pprofGroup.GET("/mutex", gin.WrapH(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.DefaultServeMux.ServeHTTP(w, r)
		})))
		pprofGroup.GET("/threadcreate", gin.WrapH(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.DefaultServeMux.ServeHTTP(w, r)
		})))
	}
}

func main() {

	logrus.Info("creating kafka producer")

	kafkaConfig := kafka.ConfigMap{
		"bootstrap.servers":            kafkaBrokerList,
		"compression.codec":            kafkaCompression,
		"batch.num.messages":           kafkaBatchNumMessages,
		"go.batch.producer":            true,
		"go.delivery.reports":          true,
		"queue.buffering.max.messages": kafkaQueueMaxMessages,
		"queue.buffering.max.kbytes":   kafkaQueueMaxKbytes,
	}

	if kafkaSslClientCertFile != "" && kafkaSslClientKeyFile != "" && kafkaSslCACertFile != "" {
		if kafkaSecurityProtocol == "" {
			kafkaSecurityProtocol = "ssl"
		}

		if kafkaSecurityProtocol != "ssl" && kafkaSecurityProtocol != "sasl_ssl" {
			logrus.Fatal("invalid config: kafka security protocol is not ssl based but ssl config is provided")
		}

		kafkaConfig["security.protocol"] = kafkaSecurityProtocol
		kafkaConfig["ssl.ca.location"] = kafkaSslCACertFile              // CA certificate file for verifying the broker's certificate.
		kafkaConfig["ssl.certificate.location"] = kafkaSslClientCertFile // Client's certificate
		kafkaConfig["ssl.key.location"] = kafkaSslClientKeyFile          // Client's key
		kafkaConfig["ssl.key.password"] = kafkaSslClientKeyPass          // Key password, if any.
	}

	if kafkaSaslMechanism != "" && kafkaSaslUsername != "" && kafkaSaslPassword != "" {
		if kafkaSecurityProtocol != "sasl_ssl" && kafkaSecurityProtocol != "sasl_plaintext" {
			logrus.Fatal("invalid config: kafka security protocol is not sasl based but sasl config is provided")
		}

		kafkaConfig["security.protocol"] = kafkaSecurityProtocol
		kafkaConfig["sasl.mechanism"] = kafkaSaslMechanism
		kafkaConfig["sasl.username"] = kafkaSaslUsername
		kafkaConfig["sasl.password"] = kafkaSaslPassword
	}

	producer, err := kafka.NewProducer(&kafkaConfig)

	if err != nil {
		logrus.WithError(err).Fatal("couldn't create kafka producer")
	}

	go consumeKafkaEvents(producer)

	r := gin.New()

	r.Use(gin.LoggerWithConfig(gin.LoggerConfig{
		Formatter: LoggerWithFormatter,
		SkipPaths: func() []string {
			if logSkipReceive {
				return []string{"/receive"}
			}
			return []string{}
		}(),
	}))

	r.GET("/metrics", gin.WrapH(promhttp.Handler()))
	r.GET("/healthz", func(c *gin.Context) { c.JSON(200, gin.H{"status": "UP"}) })

	// pprof 路由 - 根据配置决定是否启用
	setupPprofRoutes(r)

	// pprof 控制接口
	r.POST("/pprof/enable", func(c *gin.Context) {
		pprofEnabled = true
		logrus.Info("pprof enabled")
		c.JSON(200, gin.H{"status": "pprof enabled", "enabled": pprofEnabled})
	})

	r.POST("/pprof/disable", func(c *gin.Context) {
		pprofEnabled = false
		logrus.Info("pprof disabled")
		c.JSON(200, gin.H{"status": "pprof disabled", "enabled": pprofEnabled})
	})

	r.GET("/pprof/status", func(c *gin.Context) {
		c.JSON(200, gin.H{"enabled": pprofEnabled})
	})

	if basicauth {
		authorized := r.Group("/", gin.BasicAuth(gin.Accounts{
			basicauthUsername: basicauthPassword,
		}))
		authorized.POST("/receive", receiveHandler(producer, serializer))
	} else {
		r.POST("/receive", receiveHandler(producer, serializer))
	}
	logrus.Fatal(r.Run())
}

func consumeKafkaEvents(producer *kafka.Producer) {
	for e := range producer.Events() {
		switch ev := e.(type) {
		case *kafka.Message:
			if ev.TopicPartition.Error != nil {
				kafkaDeliveryErrors.Inc()
				logrus.WithError(ev.TopicPartition.Error).Debug("kafka delivery failed")
			} else {
				kafkaDeliverySuccess.Inc()
			}
		case kafka.Error:
			kafkaErrors.Inc()
			logrus.WithError(ev).Error("kafka error")
		}
	}
}
