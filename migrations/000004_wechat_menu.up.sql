-- 000004_wechat_menu.up.sql
-- T6: 微信自定义菜单管理
-- 2026-08-05

CREATE TABLE IF NOT EXISTS wechat_menus (
    id              BIGSERIAL       PRIMARY KEY,
    account_id      BIGINT          NOT NULL,               -- 所属公众号
    menu_json       JSONB           NOT NULL DEFAULT '{}',  -- 菜单 JSON 结构
    status          SMALLINT        NOT NULL DEFAULT 0,     -- 0=草稿 1=已发布
    published_at    TIMESTAMPTZ,                            -- 上次发布时间
    deleted_at      TIMESTAMPTZ,                            -- 软删除
    created_at      TIMESTAMPTZ     NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ     NOT NULL DEFAULT NOW(),

    CONSTRAINT fk_wechat_menus_account
        FOREIGN KEY (account_id) REFERENCES wechat_accounts(id)
);

-- 查询某公众号的菜单（草稿+已发布）
CREATE INDEX IF NOT EXISTS idx_wechat_menus_account
    ON wechat_menus(account_id, status) WHERE deleted_at IS NULL;

COMMENT ON TABLE wechat_menus IS '微信自定义菜单';
COMMENT ON COLUMN wechat_menus.menu_json IS '微信菜单 JSON 结构: {button: [{type, name, key/url/sub_button}]}';
COMMENT ON COLUMN wechat_menus.status IS '0=草稿 1=已发布';

DROP TRIGGER IF EXISTS trg_wechat_menus_updated_at ON wechat_menus;
CREATE TRIGGER trg_wechat_menus_updated_at
    BEFORE UPDATE ON wechat_menus
    FOR EACH ROW EXECUTE FUNCTION update_timestamp();
