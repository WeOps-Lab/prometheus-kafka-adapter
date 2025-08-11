#!/bin/bash
set -x

# 获取参数
ARCH_PARAM=${1:-}  # 第一个参数作为架构
VERSION=${VERSION:-latest}  # 版本号从环境变量获取，默认为 latest

# 设置 Golang 和 Alpine 版本
GOLANG_VERSION=1.19.1-alpine
ALPINE_VERSION=3.16

# 镜像名称和仓库配置
IMAGE_NAME=weops-kafka-adapter
REGISTRY=docker-bkrepo.cwoa.net/ce1b09/weops-docker

# 构建函数
build_for_arch() {
    local arch=$1
    local platform=""

    # 选择 Docker 平台参数
    if [ "$arch" == "amd64" ]; then
        platform=linux/amd64
    elif [ "$arch" == "arm64" ]; then
        platform=linux/arm64
    else
        echo "Unsupported architecture: $arch"
        return 1
    fi

    echo "Building for architecture: $arch, platform: $platform"

    # 构建 Docker 镜像
    docker build --platform=$platform -t $IMAGE_NAME:$arch --build-arg ARCH=$arch .

    if [ $? -ne 0 ]; then
        echo "Failed to build for $arch"
        return 1
    fi

    # 打 Tag
    local full_version=$VERSION-$arch
    docker tag $IMAGE_NAME:$arch $REGISTRY/$IMAGE_NAME:$full_version

    # 推送到仓库
    docker push $REGISTRY/$IMAGE_NAME:$full_version

    if [ $? -eq 0 ]; then
        echo "Successfully built and pushed $arch version: $REGISTRY/$IMAGE_NAME:$full_version"
    else
        echo "Failed to push $arch version"
        return 1
    fi
}

# 主构建逻辑
if [ -z "$ARCH_PARAM" ]; then
    # 如果没有传递架构参数，构建所有支持的架构
    echo "No architecture specified, building for all supported architectures..."

    ARCHITECTURES=("amd64" "arm64")

    for arch in "${ARCHITECTURES[@]}"; do
        echo "========================================="
        echo "Building for $arch..."
        echo "========================================="

        build_for_arch $arch

        if [ $? -ne 0 ]; then
            echo "Build failed for $arch, continuing with other architectures..."
        fi
    done

    echo "========================================="
    echo "Multi-architecture build completed!"
    echo "========================================="

else
    # 如果传递了架构参数，只构建指定的架构
    echo "Building for specified architecture: $ARCH_PARAM"
    build_for_arch $ARCH_PARAM

    if [ $? -eq 0 ]; then
        echo "Build completed successfully for $ARCH_PARAM"
    else
        echo "Build failed for $ARCH_PARAM"
        exit 1
    fi
fi
