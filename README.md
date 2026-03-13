# Go-Cloud-System (高并发分布式存储与抽奖互动系统)

基于 Go (Gin) 构建的微服务架构项目，包含统一 API 网关、高并发抽奖引擎与分布式文件存储中台。

## 🚀 核心架构与技术栈
* **微服务架构**: Gateway (网关) / Interaction (交互) / Storage (存储) / Compute (计算)
* **核心技术**: Go 1.26, Gin, GORM
* **中间件**: MySQL, Redis, RabbitMQ
* **高并发保障**: 基于 Redis Lua 脚本的原子性库存扣减与 O(1) 幂等性防刷拦截
* **异步解耦**: 引入 RabbitMQ 实现交易链路削峰填谷与数据最终一致性
* **压测性能**: 单机并发测试下，秒杀接口 QPS 达 16,000+，P99 延迟 < 50ms。

## 🛠️ 快速开始

1. 启动底层中间件 (MySQL/Redis/MQ/MinIO):
   ```bash
   docker-compose up -d

2. 一键启动所有微服务:
    ```Bash
    make run-all

3. 停止所有微服务:
    ```Bash
    make stop-all   
