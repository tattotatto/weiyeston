-- 000002_add_role.up.sql
-- 为 tenants 表增加 role 字段，支持平台管理员与普通租户的权限区分

ALTER TABLE tenants ADD COLUMN IF NOT EXISTS role VARCHAR(20) NOT NULL DEFAULT 'user';

-- role 约束：仅允许 admin 和 user
ALTER TABLE tenants DROP CONSTRAINT IF EXISTS chk_tenants_role;
ALTER TABLE tenants ADD CONSTRAINT chk_tenants_role CHECK (role IN ('admin', 'user'));

-- 按 role 查询索引
CREATE INDEX IF NOT EXISTS idx_tenants_role ON tenants(role) WHERE deleted_at IS NULL;

COMMENT ON COLUMN tenants.role IS '角色: admin=平台管理员, user=普通租户';
