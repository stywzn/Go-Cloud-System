.PHONY: infra-up infra-down run-all stop-all run-web

# 一键启动所有底层中间件
infra-up:
	docker compose up -d
	@echo "Infrastructure is up! RabbitMQ UI: http://localhost:15672, MinIO UI: http://localhost:9001"

# 一键关闭中间件
infra-down:
	docker compose down

# 一键启动三个核心微服务在后台运行 (修复了相对路径读取配置的问题)
run-all:
	@echo "Starting Gateway Service..."
	cd gateway && go run ./cmd/gateway/main.go &
	@echo "Starting Interaction Service..."
	cd interaction && go run ./cmd/api/main.go &
	@echo "Starting Storage Service..."
	cd storage && go run ./cmd/server/main.go &
	@echo "All services are running in background. Use 'make stop-all' to terminate."

# 启动前端服务
run-web:
	@echo "Starting Web Frontend..."
	cd web && go run server.go

# 一键杀死所有运行的微服务
stop-all:
	pkill -f "go run ./cmd/gateway/main.go" || true
	pkill -f "go run ./cmd/api/main.go" || true
	pkill -f "go run ./cmd/server/main.go" || true
	pkill -f "go run server.go" || true
	@echo "All microservices stopped."

# 一键启动完整系统 (包括前端)
run-full: run-all run-web