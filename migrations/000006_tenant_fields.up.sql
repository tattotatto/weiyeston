-- Add tenant status, VIP fields
ALTER TABLE tenants ADD COLUMN IF NOT EXISTS status SMALLINT NOT NULL DEFAULT 0;
ALTER TABLE tenants ADD COLUMN IF NOT EXISTS vip_end_time TIMESTAMPTZ;
ALTER TABLE tenants ADD COLUMN IF NOT EXISTS vip_level VARCHAR(20) NOT NULL DEFAULT 'trial';

-- Auto-approve existing admin user
UPDATE tenants SET status = 1 WHERE username = 'admin';
