# syntax=docker/dockerfile:1
# 微盈通 V2 - 多阶段 Docker 生产构建
# 包含: Go 后端构建 + React 前端构建 + 最终运行镜像

# ============ 前端构建 ============
FROM node:20-alpine AS frontend-builder
WORKDIR /web
COPY web/admin/package.json web/admin/package-lock.json ./
RUN npm ci --registry=https://registry.npmmirror.com
COPY web/admin/ ./
RUN npm run build

# ============ Go 后端构建 ============
FROM golang:1.24-alpine AS backend-builder
RUN apk add --no-cache git ca-certificates
WORKDIR /build
COPY go.mod go.sum ./
RUN go mod tidy && go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o weiyeston ./cmd/server

# ============ 最终运行镜像 ============
FROM alpine:3.21

RUN apk add --no-cache ca-certificates tzdata curl bash && \
    adduser -D -h /app weiyeston

WORKDIR /app

# Go 二进制
COPY --from=backend-builder /build/weiyeston .

# 前端静态文件
COPY --from=frontend-builder /web/dist ./web/admin

# Go 模板
COPY --from=backend-builder /build/templates ./templates

# 数据库迁移脚本
COPY --from=backend-builder /build/migrations ./migrations

# 配置文件
COPY --from=backend-builder /build/config.prod.yaml ./config.yaml

# 入口脚本
COPY scripts/docker-entrypoint.sh ./entrypoint.sh
RUN chmod +x ./entrypoint.sh && chown -R weiyeston:weiyeston /app

RUN mkdir -p /app/uploads && chown weiyeston:weiyeston /app/uploads

USER weiyeston
EXPOSE 8080

HEALTHCHECK --interval=30s --timeout=3s --start-period=10s --retries=3 \
    CMD curl -f http://localhost:8080/api/health || exit 1

ENTRYPOINT ["./entrypoint.sh"]
