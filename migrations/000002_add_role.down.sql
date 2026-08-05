-- 000002_add_role.down.sql

DROP INDEX IF EXISTS idx_tenants_role;
ALTER TABLE tenants DROP CONSTRAINT IF EXISTS chk_tenants_role;
ALTER TABLE tenants DROP COLUMN IF EXISTS role;
