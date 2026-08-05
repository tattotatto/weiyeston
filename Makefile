# Makefile — 常用开发/构建/部署命令

.PHONY: help dev dev-db dev-server dev-web build build-server build-web test lint clean migrate

help: ## 显示帮助信息
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'

# ============ 开发 ============
dev: ## 启动完整开发环境 (Docker Compose)
	docker compose up -d
	@echo ">>> Postgres: localhost:5432"
	@echo ">>> Redis:    localhost:6379"
	@echo ">>> Server:   http://localhost:8080"
	@echo ">>> Web:      http://localhost:5173"

dev-db: ## 仅启动 PostgreSQL + Redis
	docker compose up -d postgres redis

dev-server: ## 本地运行 Go 服务（配合外部 PG/Redis）
	go run ./cmd/server --config=config.dev.yaml

dev-web: ## 本地运行前端开发服务器
	cd web/admin && npm run dev

# ============ 构建 ============
build: build-server build-web ## 构建后端 + 前端

build-server: ## 编译 Go 后端（单文件）
	go build -ldflags="-s -w" -o bin/weiyeston ./cmd/server

build-web: ## 构建前端静态文件
	cd web/admin && npm ci && npm run build

# ============ 测试 ============
test: ## 运行所有测试
	go test -v -race -coverprofile=coverage.out ./...
	cd web/admin && npm run test

test-server: ## 运行 Go 测试
	go test -v -race -coverprofile=coverage.out ./...

test-web: ## 运行前端测试
	cd web/admin && npm run test

lint: ## 代码检查
	golangci-lint run ./...
	cd web/admin && npm run lint

# ============ 数据库 ============
migrate-up: ## 执行数据库迁移
	@bash scripts/migrate.sh up

migrate-down: ## 回滚数据库迁移
	@bash scripts/migrate.sh down

migrate-create: ## 创建新迁移文件 (usage: make migrate-create NAME=add_something)
	migrate create -ext sql -dir migrations -seq $(NAME)

# ============ 清理 ============
clean: ## 清理构建产物
	rm -rf bin/ .air/ web/admin/dist/ coverage.out
