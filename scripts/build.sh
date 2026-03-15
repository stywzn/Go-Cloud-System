#!/bin/bash

# 构建脚本
set -e

echo "🚀 开始构建 Go Cloud System..."

# 检查Docker是否安装
if ! command -v docker &> /dev/null; then
    echo "❌ Docker 未安装，请先安装 Docker"
    exit 1
fi

# 检查Docker Compose是否安装
if ! command -v docker compose &> /dev/null; then
    echo "❌ Docker Compose 未安装，请先安装 Docker Compose"
    exit 1
fi

# 设置环境变量
export DOCKER_BUILDKIT=1
export COMPOSE_DOCKER_CLI_BUILD=1

# 获取版本信息
VERSION=${1:-latest}
REGISTRY=${2:-localhost:5000}

echo "📦 构建版本: $VERSION"
echo "🏷️  镜像仓库: $REGISTRY"

# 构建应用镜像
echo "🔨 构建应用镜像..."
docker build -t $REGISTRY/gcs-app:$VERSION .
docker tag $REGISTRY/gcs-app:$VERSION $REGISTRY/gcs-app:latest

# 推送到镜像仓库 (可选)
if [ "$3" = "push" ]; then
    echo "📤 推送镜像到仓库..."
    docker push $REGISTRY/gcs-app:$VERSION
    docker push $REGISTRY/gcs-app:latest
fi

echo "✅ 构建完成！"
echo ""
echo "📋 镜像列表:"
docker images | grep gcs-app

echo ""
echo "🚀 启动命令:"
echo "  开发环境: docker compose up -d"
echo "  生产环境: docker compose -f docker-compose.prod.yml up -d"
