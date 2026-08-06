-- 000009_add_company.up.sql
-- Add company column to tenants table
ALTER TABLE tenants ADD COLUMN IF NOT EXISTS company VARCHAR(200);
