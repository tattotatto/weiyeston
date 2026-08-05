#!/bin/bash
# 微盈通 V2 - Docker 入口脚本
# 执行数据库迁移 → 启动服务

set -e

echo "=== 微盈通 V2 ==="
echo "等待数据库就绪..."

# 等待 PostgreSQL 就绪 (最多 30 秒)
for i in $(seq 1 30); do
    if curl -s "http://${WEIYESTON_DATABASE_HOST:-postgres}:5432/" > /dev/null 2>&1; then
        break
    fi
    # pg_isready 也尝试
    if command -v pg_isready > /dev/null 2>&1; then
        if pg_isready -h "${WEIYESTON_DATABASE_HOST:-postgres}" -p "${WEIYESTON_DATABASE_PORT:-5432}" -U "${WEIYESTON_DATABASE_USER:-postgres}" -t 3 > /dev/null 2>&1; then
            break
        fi
    fi
    echo "  等待数据库... (${i}/30)"
    sleep 1
done

echo "数据库就绪，执行迁移..."
# 迁移脚本使用 golang-migrate 格式，通过内部 migration 模块执行
# 这里直接启动服务，由服务内部 RunMigrations 处理
# 如果需要在启动前单独运行迁移，取消下面注释:
# /app/migrate --config=/app/config.yaml

echo "启动微盈通 V2 服务..."
exec /app/weiyeston --config=/app/config.yaml
