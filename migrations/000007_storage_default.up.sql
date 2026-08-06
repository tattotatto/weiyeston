-- 000007_storage_default.up.sql
-- 默认存储配置 — 本地存储驱动
-- 2026-08-04

INSERT INTO system_configs (account_id, key, value, type, description)
VALUES
    (NULL, 'storage.driver',      'local',     'string', '存储驱动: local/s3'),
    (NULL, 'storage.local_path',  './uploads', 'string', '本地存储路径'),
    (NULL, 'storage.s3_endpoint', '',          'string', 'S3兼容Endpoint'),
    (NULL, 'storage.s3_bucket',   '',          'string', 'S3存储桶名称'),
    (NULL, 'storage.s3_region',   '',          'string', 'S3区域'),
    (NULL, 'storage.s3_key',      '',          'string', 'S3访问密钥'),
    (NULL, 'storage.s3_secret',   '',          'string', 'S3秘密密钥'),
    (NULL, 'storage.public_url',  '',          'string', '自定义CDN域名')
ON CONFLICT (COALESCE(account_id, 0), key) DO NOTHING;
