# 数据库迁移

使用 golang-migrate/migrate 管理 PostgreSQL 数据库 schema 版本。

## 文件命名规范

```
格式：{6位序号}_{描述}.{direction}.sql
```

## 迁移命令

```bash
# CLI 方式
migrate -source file://migrations -database "postgres://user:pass@localhost:5432/weiyeston?sslmode=disable" up
migrate -source file://migrations -database "postgres://user:pass@localhost:5432/weiyeston?sslmode=disable" down

# Makefile 快捷方式
make migrate-up
make migrate-down

# 创建新迁移
make migrate-create NAME=add_new_table
```

## 已有迁移

| 序号 | 描述 | 说明 |
|------|------|------|
| 000001 | init | 初始化所有核心表结构 |
