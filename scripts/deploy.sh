#!/bin/bash

# 部署脚本
set -e

echo "🚀 开始部署 Go Cloud System..."

# 检查参数
ENVIRONMENT=${1:-dev}
VERSION=${2:-latest}

echo "🌍 部署环境: $ENVIRONMENT"
echo "📦 应用版本: $VERSION"

# 检查环境文件
ENV_FILE=".env.$ENVIRONMENT"
if [ ! -f "$ENV_FILE" ]; then
    echo "❌ 环境文件 $ENV_FILE 不存在"
    echo "💡 请复制 .env.example 到 $ENV_FILE 并配置相应参数"
    exit 1
fi

# 加载环境变量
export $(cat $ENV_FILE | xargs)
export VERSION=$VERSION

# 停止现有服务
echo "🛑 停止现有服务..."
docker compose -f docker-compose.prod.yml down --remove-orphans

# 清理旧镜像 (可选)
if [ "$3" = "clean" ]; then
    echo "🧹 清理旧镜像..."
    docker image prune -f
fi

# 拉取最新镜像
echo "📥 拉取最新镜像..."
docker compose -f docker-compose.prod.yml pull

# 启动服务
echo "🔄 启动服务..."
docker compose -f docker-compose.prod.yml up -d

# 等待服务启动
echo "⏳ 等待服务启动..."
sleep 30

# 健康检查
echo "🏥 健康检查..."
check_service() {
    local service_name=$1
    local url=$2
    local max_attempts=30
    local attempt=1

    while [ $attempt -le $max_attempts ]; do
        if curl -f -s "$url" > /dev/null; then
            echo "✅ $service_name 健康检查通过"
            return 0
        fi
        echo "⏳ $service_name 健康检查中... ($attempt/$max_attempts)"
        sleep 2
        ((attempt++))
    done
    
    echo "❌ $service_name 健康检查失败"
    return 1
}

# 检查各个服务
check_service "Gateway" "http://localhost:8080/healthz"
check_service "Interaction" "http://localhost:8082/healthz"
check_service "Frontend" "http://localhost:8084"

# 显示服务状态
echo ""
echo "📊 服务状态:"
docker compose -f docker-compose.prod.yml ps

echo ""
echo "🌐 访问地址:"
echo "  前端界面: http://localhost:8084"
echo "  API网关: http://localhost:8080"
echo "  Nginx: http://localhost"
echo "  Grafana: http://localhost:3000 (admin/admin)"
echo "  Prometheus: http://localhost:9090"
echo "  RabbitMQ: http://localhost:15672"
echo "  MinIO: http://localhost:9001"

echo ""
echo "📝 查看日志:"
echo "  docker compose -f docker-compose.prod.yml logs -f"

echo ""
echo "🛑 停止服务:"
echo "  docker compose -f docker-compose.prod.yml down"

echo ""
echo "✅ 部署完成！"
