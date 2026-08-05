# syntax=docker/dockerfile:1
# 微盈通 V2 - 多阶段 Docker 生产构建

# ============ 前端构建 ============
FROM node:20-alpine AS frontend-builder
WORKDIR /web
COPY web/admin/package.json web/admin/package-lock.json ./
RUN npm config set registry https://registry.npmmirror.com && npm ci
COPY web/admin/ ./
RUN npm run build

# ============ Go 后端构建 ============
FROM golang:1.24-alpine AS backend-builder
RUN apk add --no-cache git ca-certificates
ENV GOPROXY=https://goproxy.cn,direct
ENV GONOSUMCHECK=*
ENV GOFLAGS=-mod=mod
WORKDIR /build
COPY . .
RUN go mod download && CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o weiyeston ./cmd/server

# ============ 最终运行镜像 ============
FROM alpine:3.21

RUN apk add --no-cache ca-certificates tzdata curl bash && \
    adduser -D -h /app weiyeston

WORKDIR /app

COPY --from=backend-builder /build/weiyeston .
COPY --from=frontend-builder /web/dist ./web/admin
COPY --from=backend-builder /build/templates ./templates
COPY --from=backend-builder /build/migrations ./migrations
COPY --from=backend-builder /build/config.prod.yaml ./config.yaml

COPY scripts/docker-entrypoint.sh ./entrypoint.sh
RUN chmod +x ./entrypoint.sh && chown -R weiyeston:weiyeston /app

RUN mkdir -p /app/uploads && chown weiyeston:weiyeston /app/uploads

USER weiyeston
EXPOSE 8090

HEALTHCHECK --interval=30s --timeout=3s --start-period=10s --retries=3 \
    CMD curl -f http://localhost:8090/api/health || exit 1

ENTRYPOINT ["./entrypoint.sh"]
