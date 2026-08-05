#!/bin/bash
# scripts/migrate.sh — 数据库迁移脚本
# 用法: ./scripts/migrate.sh [up|down|version]

set -euo pipefail

DIRECTION="${1:-up}"
DB_URL="${DATABASE_URL:-postgres://postgres:postgres@localhost:5432/weiyeston?sslmode=disable}"
MIGRATIONS_DIR="./migrations"

echo ">>> 执行迁移: ${DIRECTION}"

migrate \
  -source "file://${MIGRATIONS_DIR}" \
  -database "${DB_URL}" \
  "${DIRECTION}"

echo ">>> 迁移完成"
