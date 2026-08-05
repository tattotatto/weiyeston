-- 000003_component_ticket.up.sql
-- T3: 微信第三方平台 -- 数据库变更
-- 2026-08-05

-- 1. 新增 component_verify_tickets 表
CREATE TABLE IF NOT EXISTS component_verify_tickets (
    id              BIGSERIAL       PRIMARY KEY,
    component_appid VARCHAR(50)     NOT NULL,
    ticket          TEXT            NOT NULL,
    received_at     TIMESTAMPTZ     NOT NULL DEFAULT NOW(),
    created_at      TIMESTAMPTZ     NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_tickets_appid_time
    ON component_verify_tickets(component_appid, received_at DESC);

COMMENT ON TABLE component_verify_tickets IS '微信每10分钟推送的 component_verify_ticket 持久化记录';

-- 2. wechat_accounts 增加第三方平台授权相关字段
ALTER TABLE wechat_accounts ADD COLUMN IF NOT EXISTS authorizer_appid   VARCHAR(50);
ALTER TABLE wechat_accounts ADD COLUMN IF NOT EXISTS func_info          JSONB;
ALTER TABLE wechat_accounts ADD COLUMN IF NOT EXISTS service_type_info  SMALLINT;
ALTER TABLE wechat_accounts ADD COLUMN IF NOT EXISTS verify_type_info   SMALLINT;
ALTER TABLE wechat_accounts ADD COLUMN IF NOT EXISTS nick_name          VARCHAR(100);
ALTER TABLE wechat_accounts ADD COLUMN IF NOT EXISTS head_img           VARCHAR(500);
ALTER TABLE wechat_accounts ADD COLUMN IF NOT EXISTS user_name          VARCHAR(50);
ALTER TABLE wechat_accounts ADD COLUMN IF NOT EXISTS alias              VARCHAR(100);
ALTER TABLE wechat_accounts ADD COLUMN IF NOT EXISTS principal_name     VARCHAR(100);
ALTER TABLE wechat_accounts ADD COLUMN IF NOT EXISTS qrcode_url         VARCHAR(500);
ALTER TABLE wechat_accounts ADD COLUMN IF NOT EXISTS signature          VARCHAR(500);

-- 3. authorizer_appid 唯一索引（仅对非空且未删除的记录）
CREATE UNIQUE INDEX IF NOT EXISTS idx_accounts_authorizer_appid
    ON wechat_accounts(authorizer_appid)
    WHERE authorizer_appid IS NOT NULL AND deleted_at IS NULL;

COMMENT ON COLUMN wechat_accounts.authorizer_appid  IS '授权方AppId（第三方平台授权场景）';
COMMENT ON COLUMN wechat_accounts.func_info         IS 'JSON数组: 授权的接口权限集';
COMMENT ON COLUMN wechat_accounts.service_type_info IS '0=订阅号 1=升级订阅号 2=服务号';
COMMENT ON COLUMN wechat_accounts.verify_type_info  IS '-1=未认证 0=微信认证';
COMMENT ON COLUMN wechat_accounts.nick_name         IS '授权方昵称（微信官方名称）';
COMMENT ON COLUMN wechat_accounts.head_img          IS '授权方头像';
COMMENT ON COLUMN wechat_accounts.user_name         IS '授权方原始微信号';
COMMENT ON COLUMN wechat_accounts.alias             IS '授权方别名';
COMMENT ON COLUMN wechat_accounts.principal_name    IS '公众号主体名称';
COMMENT ON COLUMN wechat_accounts.qrcode_url         IS '公众号二维码图片URL（微信侧同步）';
COMMENT ON COLUMN wechat_accounts.signature         IS '公众号功能介绍';
