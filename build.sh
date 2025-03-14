#!/bin/bash
set -x

# 设置默认架构
ARCH=${ARCH:-${1:-amd64}}  # 优先使用环境变量，其次使用第一个参数，最后默认 amd64
VERSION=${VERSION:-latest}  # 版本号从外部传入，默认为 latest


# 设置 Golang 和 Alpine 版本
GOLANG_VERSION=1.19.1-alpine
ALPINE_VERSION=3.16

# 选择 Docker 平台参数
if [ "$ARCH" == "amd64" ]; then
    PLATFORM=linux/amd64
elif [ "$ARCH" == "arm64" ]; then
    PLATFORM=linux/arm64
else
    echo "Unsupported architecture: $ARCH"
    exit 1
fi

# 镜像名称
IMAGE_NAME=weops-kafka-adapter
FULL_VERSION=$VERSION-$ARCH
REGISTRY=docker-bkrepo.cwoa.net/ce1b09/weops-docker

# 构建 Docker 镜像
docker build --platform=$PLATFORM -t $IMAGE_NAME:latest --build-arg ARCH=$ARCH .

# 打 Tag
docker tag $IMAGE_NAME:latest $REGISTRY/$IMAGE_NAME:$FULL_VERSION

# 推送到仓库
docker push $REGISTRY/$IMAGE_NAME:$FULL_VERSION

#docker run -d --restart=always --net=host \
#-e KAFKA_BROKER_LIST=10.10.26.237:9092 \
#-e BASIC_AUTH_USERNAME=admin \
#-e BASIC_AUTH_PASSWORD=admin \
#-e PORT=8080 \
#-e BKAPP_PAAS_HOST=http://paas.service.consul \
#-e BKAPP_WEOPS_APP_ID=weops_saas \
#-e BKAPP_WEOPS_APP_SECRET=7297613c-de40-477a-b13c-f86b3cc3cb99 \
#-e LOG_SKIP_RECEIVE=True \
#-e LOG_LEVEL=info \
#--name=kafka-adapter \
#docker-bkrepo.cwoa.net/ce1b09/weops-docker/weops-kafka-adapter:v1.2.0