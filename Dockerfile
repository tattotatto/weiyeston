# syntax=docker/dockerfile:1

# =============================================================================
# 微盈通 V2 - 多阶段 Docker 构建
# =============================================================================

# ============ 构建阶段 ============
FROM golang:1.23-alpine AS builder

RUN apk add --no-cache git ca-certificates

WORKDIR /build

# 缓存依赖下载
COPY go.mod go.sum ./
RUN go mod download

# 复制源码并构建
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o weiyeston ./cmd/server

# ============ 运行阶段 ============
FROM alpine:3.21

RUN apk add --no-cache ca-certificates tzdata curl && \
    adduser -D -h /app weiyeston

WORKDIR /app

COPY --from=builder /build/weiyeston .
COPY --from=builder /build/config.prod.yaml ./config.yaml
COPY --from=builder /build/templates ./templates

RUN mkdir -p /app/uploads && chown -R weiyeston:weiyeston /app

USER weiyeston

EXPOSE 8080

HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD curl -f http://localhost:8080/api/health || exit 1

ENTRYPOINT ["./weiyeston"]
CMD ["--config=config.yaml"]
