-- 000003_component_ticket.down.sql
-- T3: 微信第三方平台 -- 回滚

DROP TABLE IF EXISTS component_verify_tickets;

ALTER TABLE wechat_accounts DROP COLUMN IF EXISTS authorizer_appid;
ALTER TABLE wechat_accounts DROP COLUMN IF EXISTS func_info;
ALTER TABLE wechat_accounts DROP COLUMN IF EXISTS service_type_info;
ALTER TABLE wechat_accounts DROP COLUMN IF EXISTS verify_type_info;
ALTER TABLE wechat_accounts DROP COLUMN IF EXISTS nick_name;
ALTER TABLE wechat_accounts DROP COLUMN IF EXISTS head_img;
ALTER TABLE wechat_accounts DROP COLUMN IF EXISTS user_name;
ALTER TABLE wechat_accounts DROP COLUMN IF EXISTS alias;
ALTER TABLE wechat_accounts DROP COLUMN IF EXISTS principal_name;
ALTER TABLE wechat_accounts DROP COLUMN IF EXISTS qrcode_url;
ALTER TABLE wechat_accounts DROP COLUMN IF EXISTS signature;
