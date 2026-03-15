# 🚀 Go Cloud System 部署指南

## 📋 部署概述

本文档介绍如何将 Go Cloud System 部署到生产环境，包括 Docker 容器化部署和 Kubernetes 集群部署。

## 🐳 Docker 容器化部署

### 前置要求

- Docker 20.10+
- Docker Compose 2.0+
- 至少 4GB RAM
- 至少 20GB 磁盘空间

### 快速部署

1. **克隆项目**
   ```bash
   git clone https://github.com/your-org/Go-Cloud-System.git
   cd Go-Cloud-System
   ```

2. **配置环境变量**
   ```bash
   cp .env.example .env
   # 编辑 .env 文件，设置密码和配置
   ```

3. **构建镜像**
   ```bash
   chmod +x scripts/build.sh
   ./scripts/build.sh latest localhost:5000
   ```

4. **启动服务**
   ```bash
   # 开发环境
   docker compose up -d
   
   # 生产环境
   docker compose -f docker-compose.prod.yml up -d
   ```

5. **验证部署**
   ```bash
   # 检查服务状态
   docker compose -f docker-compose.prod.yml ps
   
   # 健康检查
   curl http://localhost:8080/healthz
   ```

### 生产环境配置

1. **环境变量配置**
   ```bash
   # .env.prod
   MYSQL_ROOT_PASSWORD=your_secure_password
   MYSQL_USER=gcs_user
   MYSQL_PASSWORD=your_secure_password
   RABBITMQ_USER=admin
   RABBITMQ_PASSWORD=your_secure_password
   MINIO_ROOT_USER=admin
   MINIO_ROOT_PASSWORD=your_secure_password
   ```

2. **SSL证书配置**
   ```bash
   mkdir -p config/nginx/ssl
   # 将SSL证书放入 config/nginx/ssl/
   # cert.pem 和 key.pem
   ```

3. **启动生产服务**
   ```bash
   chmod +x scripts/deploy.sh
   ./scripts/deploy.sh prod latest
   ```

## ☸️ Kubernetes 部署

### 前置要求

- Kubernetes 1.24+
- kubectl 配置正确
- Helm 3.0+ (可选)

### 部署步骤

1. **创建命名空间**
   ```bash
   kubectl apply -f k8s/namespace.yaml
   ```

2. **创建ConfigMap和Secret**
   ```bash
   kubectl create configmap gcs-config --from-env-file=.env.prod -n gcs-system
   kubectl create secret generic gcs-secrets --from-env-file=.env.secrets -n gcs-system
   ```

3. **部署应用**
   ```bash
   kubectl apply -f k8s/deployment.yaml
   ```

4. **部署依赖服务**
   ```bash
   # 使用Helm部署依赖服务
   helm repo add bitnami https://charts.bitnami.com/bitnami
   helm install mysql bitnami/mysql -n gcs-system
   helm install redis bitnami/redis -n gcs-system
   helm install rabbitmq bitnami/rabbitmq -n gcs-system
   ```

5. **验证部署**
   ```bash
   kubectl get pods -n gcs-system
   kubectl get services -n gcs-system
   kubectl get ingress -n gcs-system
   ```

### 使用Helm部署

1. **创建Helm Chart**
   ```bash
   helm create gcs-chart
   # 将配置文件移动到 gcs-chart/templates/
   ```

2. **部署应用**
   ```bash
   helm install gcs gcs-chart -n gcs-system -f gcs-chart/values.prod.yaml
   ```

3. **升级应用**
   ```bash
   helm upgrade gcs gcs-chart -n gcs-system -f gcs-chart/values.prod.yaml
   ```

## 🔄 CI/CD 自动化部署

### GitHub Actions 配置

项目已配置完整的 CI/CD 流水线：

- **代码质量检查**: go vet, go fmt, 单元测试
- **安全扫描**: Trivy, CodeQL
- **镜像构建**: 多阶段构建，安全扫描
- **自动部署**: 测试环境和生产环境

### 部署触发条件

- **测试环境**: `develop` 分支推送
- **生产环境**: 创建 Release

### 手动触发部署

```bash
# 构建并推送镜像
./scripts/build.sh v1.0.0 ghcr.io/your-org/gcs-app push

# 部署到生产环境
./scripts/deploy.sh prod v1.0.0
```

## 📊 监控和日志

### Prometheus 监控

访问地址: http://localhost:9090

监控指标:
- HTTP请求数量和延迟
- 应用内存和CPU使用率
- 数据库连接池状态
- 消息队列积压情况

### Grafana 可视化

访问地址: http://localhost:3000

默认账号: admin/admin

预置面板:
- 应用性能监控
- 基础设施监控
- 业务指标监控

### 日志管理

```bash
# 查看应用日志
docker compose -f docker-compose.prod.yml logs -f app

# 查看特定服务日志
docker compose -f docker-compose.prod.yml logs -f mysql
```

## 🔧 运维操作

### 扩容操作

```bash
# Docker Compose 扩容
docker compose -f docker-compose.prod.yml up -d --scale app=3

# Kubernetes 扩容
kubectl scale deployment gcs-app --replicas=5 -n gcs-system
```

### 滚动更新

```bash
# Docker Compose 滚动更新
docker compose -f docker-compose.prod.yml up -d --no-deps app

# Kubernetes 滚动更新
kubectl set image deployment/gcs-app gcs-app=gcs-app:v1.1.0 -n gcs-system
```

### 回滚操作

```bash
# Docker Compose 回滚
docker compose -f docker-compose.prod.yml down
docker compose -f docker-compose.prod.yml up -d

# Kubernetes 回滚
kubectl rollout undo deployment/gcs-app -n gcs-system
```

## 🔒 安全配置

### 网络安全

- 所有服务都在内部网络中运行
- 只有 Nginx 暴露到外部
- 使用防火墙限制访问

### 数据安全

- 数据库密码使用强密码
- 敏感配置使用 Kubernetes Secret
- 启用 SSL/TLS 加密

### 访问控制

```bash
# 创建服务账号
kubectl create serviceaccount gcs-sa -n gcs-system

# 设置权限
kubectl create clusterrole gcs-role --verb=get,list,watch --resource=pods,services
kubectl create clusterrolebinding gcs-binding --clusterrole=gcs-role --serviceaccount=gcs-system:gcs-sa
```

## 🚨 故障排除

### 常见问题

1. **服务无法启动**
   ```bash
   # 检查日志
   docker compose -f docker-compose.prod.yml logs app
   
   # 检查端口占用
   netstat -tlnp | grep :8080
   ```

2. **数据库连接失败**
   ```bash
   # 检查数据库状态
   docker compose -f docker-compose.prod.yml exec mysql mysql -u root -p
   
   # 检查网络连接
   docker compose -f docker-compose.prod.yml exec app ping mysql
   ```

3. **内存不足**
   ```bash
   # 检查资源使用
   docker stats
   
   # 调整资源限制
   # 编辑 docker-compose.prod.yml 中的 resources 配置
   ```

### 性能调优

1. **数据库优化**
   - 调整 MySQL 配置参数
   - 添加适当的索引
   - 启用查询缓存

2. **缓存优化**
   - 调整 Redis 内存配置
   - 设置合适的过期策略
   - 使用 Redis 集群

3. **应用优化**
   - 调整 Go 运行时参数
   - 优化数据库连接池
   - 启用 Gzip 压缩

## 📋 发布检查清单

### 发布前检查

- [ ] 所有测试通过
- [ ] 安全扫描通过
- [ ] 性能测试通过
- [ ] 配置文件检查
- [ ] 备份策略确认
- [ ] 回滚方案准备

### 发布后检查

- [ ] 服务健康检查
- [ ] 监控指标正常
- [ ] 日志无错误
- [ ] 功能测试验证
- [ ] 性能指标监控

## 📞 支持和联系

- **技术支持**: support@example.com
- **紧急联系**: emergency@example.com
- **文档**: https://docs.example.com
- **监控**: https://monitor.example.com

---

## 🎯 总结

通过以上步骤，你可以将 Go Cloud System 成功部署到生产环境。建议在部署前先在测试环境充分验证，确保所有功能正常运行后再进行生产部署。

记住：**安全第一，监控第二，性能第三**！
