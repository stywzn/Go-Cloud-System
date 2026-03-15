# 多阶段构建 - 构建阶段
FROM golang:1.25-alpine AS builder

# 设置工作目录
WORKDIR /app

# 安装必要的包
RUN apk add --no-cache git ca-certificates tzdata

# 复制go mod文件
COPY go.work go.work.sum ./
COPY gateway/go.mod gateway/go.sum ./gateway/
COPY interaction/go.mod interaction/go.sum ./interaction/
COPY storage/go.mod storage/go.sum ./storage/
COPY pkg/go.mod pkg/go.sum ./pkg/

# 下载依赖
RUN go mod download

# 复制源代码
COPY . .

# 构建各个服务
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o gateway ./gateway/cmd/gateway/main.go
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o interaction ./interaction/cmd/api/main.go
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o storage ./storage/cmd/server/main.go

# 运行阶段 - 使用轻量级镜像
FROM alpine:latest

# 安装ca-certificates用于HTTPS请求
RUN apk --no-cache add ca-certificates tzdata

# 设置时区
ENV TZ=Asia/Shanghai

WORKDIR /root/

# 从构建阶段复制二进制文件
COPY --from=builder /app/gateway .
COPY --from=builder /app/interaction .
COPY --from=builder /app/storage .

# 复制配置文件
COPY gateway/configs ./configs/
COPY docker-compose.yml .

# 创建启动脚本
RUN echo '#!/bin/sh' > start.sh && \
    echo './gateway &' >> start.sh && \
    echo './interaction &' >> start.sh && \
    echo './storage &' >> start.sh && \
    echo 'wait' >> start.sh && \
    chmod +x start.sh

# 暴露端口
EXPOSE 8080 8081 8082 8083 8084

# 启动服务
CMD ["./start.sh"]
