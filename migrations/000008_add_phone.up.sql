-- 000008_add_phone.up.sql
-- Add phone column to tenants table (for password recovery)
ALTER TABLE tenants ADD COLUMN IF NOT EXISTS phone VARCHAR(20);
