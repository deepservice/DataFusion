# 构建阶段
FROM golang:1.18-alpine AS builder

# 安装必要的工具
RUN apk add --no-cache git make

WORKDIR /app

ENV GOPROXY=https://goproxy.cn
ENV GOSUMDB=sum.golang.google.cn

# 复制依赖文件
COPY go.mod go.sum ./
RUN go mod download

# 复制源代码
COPY . .

# 编译
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o worker cmd/worker/main.go

# 运行阶段
FROM alpine:latest

# 安装 chromium（用于 RPA 采集）和时区数据
RUN apk add --no-cache \
    chromium \
    nss \
    freetype \
    harfbuzz \
    ca-certificates \
    ttf-freefont \
    tzdata

# 设置时区
ENV TZ=Asia/Shanghai

# 设置 Chromium 环境变量
ENV CHROME_BIN=/usr/bin/chromium-browser \
    CHROME_PATH=/usr/lib/chromium/

WORKDIR /app

# 从构建阶段复制二进制文件
COPY --from=builder /app/worker .

# 创建配置目录和数据目录
RUN mkdir -p /app/config /app/data

# 运行 Worker
CMD ["./worker", "-config", "/app/config/worker.yaml"]
