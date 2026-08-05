-- 000001_init.down.sql
-- 微盈通 V2 回滚脚本 — 按外键依赖逆序删除所有表
-- 2026-08-04

-- 先删除触发器函数（依赖于所有表上创建的触发器）
DROP FUNCTION IF EXISTS update_timestamp() CASCADE;

-- 按依赖顺序逆序删除（先删子表，再删父表）
DROP TABLE IF EXISTS vote_records;
DROP TABLE IF EXISTS vote_options;
DROP TABLE IF EXISTS votes;

DROP TABLE IF EXISTS quicknews_comments;
DROP TABLE IF EXISTS quicknews_likes;
DROP TABLE IF EXISTS quicknews_photos;
DROP TABLE IF EXISTS quicknews_news;
DROP TABLE IF EXISTS quicknews_users;
DROP TABLE IF EXISTS quicknews_channels;

DROP TABLE IF EXISTS cms_articles;
DROP TABLE IF EXISTS cms_channels;

DROP TABLE IF EXISTS auto_reply_rules;

DROP TABLE IF EXISTS materials;

DROP TABLE IF EXISTS system_configs;

DROP TABLE IF EXISTS wechat_accounts;
DROP TABLE IF EXISTS tenants;

-- 可选：删除扩展（如果不再需要）
-- DROP EXTENSION IF EXISTS "pg_trgm";
-- DROP EXTENSION IF EXISTS "pgcrypto";
