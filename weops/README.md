# Weops-kafka-adapter
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
- `LOG_SKIP_RECEIVE`: 填`True`则不现实/receive请求部分的日志。
- `PPROF_ENABLED`: 启动时启用pprof性能分析，可设置为`true`或`false`，默认为`false`。

### pprof性能分析

Weops-kafka-adapter内置了Go的pprof性能分析工具，可以通过HTTP接口动态控制开启和关闭：

#### 控制接口

- `POST /pprof/enable` - 启用pprof性能分析
- `POST /pprof/disable` - 禁用pprof性能分析  
- `GET /pprof/status` - 查看当前pprof状态

#### 分析接口（启用后可用）

- `GET /debug/pprof/` - 性能分析首页
- `GET /debug/pprof/profile` - CPU性能分析
- `GET /debug/pprof/heap` - 堆内存分析
- `GET /debug/pprof/goroutine` - 协程分析
- `GET /debug/pprof/allocs` - 内存分配分析
- `GET /debug/pprof/block` - 阻塞分析
- `GET /debug/pprof/mutex` - 互斥锁分析

#### 使用示例

```bash
# 启用性能分析
curl -X POST http://localhost:8080/pprof/enable

# 查看状态
curl http://localhost:8080/pprof/status

# 获取CPU性能分析（30秒）
curl http://localhost:8080/debug/pprof/profile?seconds=30 -o profile.pprof

# 使用go tool分析
go tool pprof profile.pprof

# 或直接从URL分析
go tool pprof http://localhost:8080/debug/pprof/profile

# 分析完成后禁用
curl -X POST http://localhost:8080/pprof/disable
```

**安全提醒**：pprof默认处于禁用状态，仅在需要调试性能问题时临时启用，因为它可能暴露运行时敏感信息。

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

Prometheus 需要配置一个 `remote_write` URL，指向运行 weops-adapter 服务的主机和端口的 /receive 端点。例如：
```yaml
remote_write:
  - url: "http://weops-adapter:8080/receive"
```

### 运行示例
```shell
docker run -d --restart=always --net=host \
-e KAFKA_BROKER_LIST=$BK_KAFKA_IP:9092 \
-e BASIC_AUTH_USERNAME=admin \
-e BASIC_AUTH_PASSWORD=admin \
-e PORT=8080 \
-e BKAPP_PAAS_HOST=http://paas.service.consul \
-e BKAPP_WEOPS_APP_ID=weops_saas \
-e BKAPP_WEOPS_APP_SECRET=6a38236d-8e79-4c48-a977-504b0d286904 \
--name=weops-kafka-adapter \
docker-bkrepo.cwoa.net/ce1b09/weops-docker/weops-kafka-adapter:v1.0.0
```

### 版本日志

#### v1.0.0
- 初始化

#### v1.0.1
- 修复api请求地址变量问题

#### v1.0.2
- 加速构建镜像

#### v1.0.3
- 适配automate更多类型的自定义上报

#### v1.0.4
- 修复weops部署后才可部署adapter的问题
- 优化日志

#### v1.0.5
- 优化日志，k8s部分排查

#### v1.0.6
- 优化日志
- 修复请求301
- 修复自定义指标对weops saas接口请求过多问题

#### v1.0.7
- k8s指标kube_node_status_condition and kube_pod_status_phase动态维度恢复

#### v1.1.0
- 新增蓝鲸计算指标功能
- 增加对vector数据来源的监控数据处理

#### v1.1.1
- 修复并发使用变量问题

#### v1.1.2
- 增加k8s cluster指标

#### v1.1.3
- 只保留固定的ipmi指标

#### v1.1.4
- 修复ipmi指标obj id不正确问题

#### v1.1.5
- 修复ipmi指标obj id不正确问题

#### v1.2.0
- 过滤prometheus特殊值-Inf、+Inf、NaN
- 简化链路处理逻辑

#### v1.2.1
- 为k8s pod、node指标保留cluster_name维度
- 修复pod指标缺失instance_name、cluster_name维度

#### v1.2.2
- 修复vmware、云类采集指标被过滤的问题
- 兼容protocol=cloud protocol=automate

#### v1.2.3
- 适配k8s node实例名修改

#### v1.2.4
- node、pod监控指标适配k8s node实例名修改

#### v1.2.5
- json解析cmdb响应内容优化
- 新增pprof