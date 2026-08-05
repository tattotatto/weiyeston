-- 默认管理员用户 (用户名: admin  密码: admin123)
-- 仅当不存在时插入
INSERT INTO tenants (username, password_hash, nickname, role, email, status, created_at, updated_at)
SELECT 'admin', '$2b$12$GawRqaF/ve0/oa4DfVVFQ.NoNhllbF0yyMDqlQ6ogq2rHIBgIWFyO', '超级管理员', 'admin', 'admin@weiyeston.com', 1, NOW(), NOW()
WHERE NOT EXISTS (SELECT 1 FROM tenants WHERE username = 'admin');
