# docker/dev/Dockerfile — Go 开发环境，使用 air 实现热重载
FROM golang:1.23-alpine

RUN apk add --no-cache git make

# 安装 air（Go 热重载工具）
RUN go install github.com/air-verse/air@latest

WORKDIR /app

# 复制 go.mod / go.sum，先下载依赖（利用 Docker 缓存层）
COPY go.mod go.sum ./
RUN go mod download

# air 使用当前目录的 .air.toml 或默认配置
CMD ["air", "-c", ".air.toml"]
