# 🚀 Go-Interaction-Service (高并发互动中台服务)

本项目是一个基于 Go 语言构建的**高并发、高可用**的互动中台服务。以“点赞/取消赞”与“实时热榜”为核心业务场景，重点解决了**高并发流量削峰**、**防恶意刷赞**以及**数据最终一致性**等后端核心痛点。

项目中深度融入了 **安全开发 (SecDevOps)** 理念，实现了从网关层到数据层的多维立体防御，可直接作为企业级微服务的核心模块接入。

## 🛠️ 技术栈 (Tech Stack)

* **核心语言**: Go (Golang)
* **Web 框架**: Gin
* **存储引擎**: MySQL 8.0 + GORM (ORM)
* **缓存与中间件**: Redis (go-redis/v9)
* **并发控制**: 原生 Goroutine + Channel (Pipeline 模式)

## ✨ 核心架构亮点 (Key Features)

### 1. 极致的高并发处理 (削峰填谷)
* **读写分离架构**：读请求（总赞数、热榜）100% 命中 Redis 缓存，实现 O(1) 或 O(logN) 级别的毫秒级响应。
* **Channel 异步削峰**：写请求（点赞）在 Redis 更新状态后，立即封装为 Task 丢入本地定长 `Channel`，主干协程瞬间返回，极大提升 API 吞吐量。
* **Pipeline 批量落盘**：后台常驻 `Flusher` 消费者协程，利用 `select` 多路复用监听定时器与管道，实现“满 N 条”或“满 N 秒”自动触发 `CreateInBatches` 批量写入 MySQL，彻底解放数据库 IO。

### 2. 纵深安全防御体系 (Security First)
* **网关层防刷 (Rate Limiting)**：基于 Redis `INCR` 与 `EXPIRE` 纯手写的高性能滑动/固定窗口限流器，有效防御脚本爆破与恶意重放攻击 (触发 HTTP 429)。
* **协议安全防护**：全局挂载 Security Headers 中间件，强制开启 `nosniff` 与 XSS 阻断，免疫基础 Web 攻击。
* **并发漏洞防御 (Race Condition)**：利用 Redis 单线程 `SADD` 指令的原子性，彻底根除高并发下的重复刷赞逻辑漏洞。
* **防 SQL 注入**：数据持久层严格采用 GORM 预编译参数绑定机制，免疫 SQL 注入。

### 3. 企业级高可用保障 (High Availability)
* **优雅停机 (Graceful Shutdown)**：通过 `os/signal` 捕获 K8s/OS 驱逐信号，阻塞主进程直到 `Channel` 内所有滞留的点赞任务被强制刷入 MySQL，实现服务重启/发布时的**数据零丢失**。
* **冲突自愈 (UPSERT)**：利用 GORM `clause.OnConflict` 与 MySQL 联合唯一索引，优雅解决高频点击导致的批量落盘主键冲突问题，保障数据最终一致性。
* **高性能排行榜**：利用 Redis 跳表 (`ZSet`) 数据结构，实时更新并极速拉取全站 Top N 热门文章。

## 🚀 快速启动 (Quick Start)

### 环境依赖
* Go 1.20+
* MySQL 8.0
* Redis 6.0+

### 运行服务
```bash
# 1. 克隆项目
git clone [https://github.com/stywzn/Go-Interaction-Service.git](https://github.com/stywzn/Go-Interaction-Service.git)
cd Go-Interaction-Service

# 2. 安装依赖
go mod tidy

# 3. 启动服务
go run cmd/api/main.go