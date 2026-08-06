-- 000007_storage_default.down.sql
-- 移除默认存储配置

DELETE FROM system_configs
WHERE account_id IS NULL
  AND key IN (
    'storage.driver',
    'storage.local_path',
    'storage.s3_endpoint',
    'storage.s3_bucket',
    'storage.s3_region',
    'storage.s3_key',
    'storage.s3_secret',
    'storage.public_url'
  );
