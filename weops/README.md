# Weops-adapter
prometheus向adapter写入监控指标，adapter将指标清洗，送入蓝鲸监控链路(kafka)

## 配置

### 环境变量
基本环境变量配置：
- `KAFKA_BROKER_LIST`：定义 Kafka 的端点和端口，默认为 `kafka:9092`。
- `KAFKA_TOPIC`：定义要使用的 Kafka 主题，默认为 `metrics`。
- `KAFKA_COMPRESSION`：定义要使用的压缩类型，默认为 `none`。
- `KAFKA_BATCH_NUM_MESSAGES`：定义要批量写入的消息数，默认为 `10000`。
- `SERIALIZATION_FORMAT`：定义序列化格式，可以是 `json`、`avro-json`，默认为 `json`。
- `PORT`：定义要监听的 HTTP 端口，默认为 `8080`，由 [gin](https://github.com/gin-gonic/gin) 直接使用。
- `BASIC_AUTH_USERNAME`：用于接收端点的基本身份验证用户名，默认为无基本身份验证。
- `BASIC_AUTH_PASSWORD`：用于接收端点的基本身份验证密码，默认为无基本身份验证。
- `LOG_LEVEL`：为 [`logrus`](https://github.com/sirupsen/logrus) 定义日志级别，可以是 `debug`、`info`、`warn`、`error`、`fatal` 或 `panic`，默认为 `info`。
- `GIN_MODE`：管理 [gin](https://github.com/gin-gonic/gin) 调试日志记录，可以是 `debug` 或 `release`。

Weops环境变量配置:
- `BKAPP_PAAS_HOST`: 蓝鲸Paas访问地址，默认为 `http://paas.weops.com`。
- `BKAPP_WEOPS_APP_ID`: Weops访问地址，默认为 `weops_saas`。
- `BKAPP_WEOPS_APP_SECRET`: Weops Saas秘钥，可从开发者中心获取。
- `METRICS_FILE`: k8s指标列表文件，默认已内置于容器，默认为 `metrics.yaml`。



要通过 SSL 连接到 Kafka，请定义以下其他环境变量：
- `KAFKA_SSL_CLIENT_CERT_FILE`：Kafka SSL 客户端证书文件，默认为 `""`
- `KAFKA_SSL_CLIENT_KEY_FILE`：Kafka SSL 客户端证书密钥文件，默认为 `""`
- `KAFKA_SSL_CLIENT_KEY_PASS`：Kafka SSL 客户端证书密钥密码（可选），默认为 `""`
- `KAFKA_SSL_CA_CERT_FILE`：Kafka SSL 经纪人 CA 证书文件，默认为 `""`

要通过 SASL/SCRAM 身份验证连接到 Kafka，请定义以下其他环境变量：
- `KAFKA_SECURITY_PROTOCOL`：用于与代理通信的 Kafka 客户端使用的协议，如果要使用 SASL，则必须设置为 plain 或带有 SSL
- `KAFKA_SASL_MECHANISM`：用于身份验证的 SASL 机制，默认为 `""`
- `KAFKA_SASL_USERNAME`：用于 PLAIN 和 SASL-SCRAM-.. 机制的 SASL 用户名，默认为 `""`
- `KAFKA_SASL_PASSWORD`：用于 PLAIN 和 SASL-SCRAM-.. 机制的 SASL 密码，默认为 `""`

### prometheus配置

Prometheus needs to have a `remote_write` url configured, pointing to the '/receive' endpoint of the host and port where the prometheus-kafka-adapter service is running. For example:

```yaml
remote_write:
  - url: "http://prometheus-kafka-adapter:8080/receive"
```
