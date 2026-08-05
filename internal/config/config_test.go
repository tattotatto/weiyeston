package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestLoadFromFile 测试从 config.yaml 文件加载配置
func TestLoadFromFile(t *testing.T) {
	t.Run("加载完整配置文件", func(t *testing.T) {
		configPath := filepath.Join("..", "..", "config.yaml")
		cfg, err := Load(configPath)

		require.NoError(t, err, "加载配置文件不应返回错误")
		require.NotNil(t, cfg, "配置对象不应为 nil")

		// 验证 server 配置
		assert.Equal(t, 8080, cfg.Server.Port, "端口应为 8080")
		assert.Equal(t, "debug", cfg.Server.Mode, "模式应为 debug")
		assert.Equal(t, 30*time.Second, cfg.Server.ReadTimeout, "读超时应为 30s")
		assert.Equal(t, 60*time.Second, cfg.Server.WriteTimeout, "写超时应为 60s")

		// 验证 database 配置
		assert.Equal(t, "localhost", cfg.Database.Host, "数据库主机应为 localhost")
		assert.Equal(t, 5432, cfg.Database.Port, "数据库端口应为 5432")
		assert.Equal(t, "postgres", cfg.Database.User, "数据库用户应为 postgres")
		assert.Equal(t, "weiyeston", cfg.Database.DBName, "数据库名应为 weiyeston")
		assert.Equal(t, "disable", cfg.Database.SSLMode, "SSL 模式应为 disable")
		assert.Equal(t, 25, cfg.Database.MaxOpenConns, "最大连接数应为 25")
		assert.Equal(t, 5, cfg.Database.MaxIdleConns, "最大空闲连接数应为 5")
		assert.Equal(t, 5*time.Minute, cfg.Database.ConnMaxLifetime, "连接最大生命周期应为 5m")

		// 验证 redis 配置
		assert.Equal(t, "localhost:6379", cfg.Redis.Addr, "Redis 地址应为 localhost:6379")
		assert.Equal(t, "", cfg.Redis.Password, "Redis 密码应为空")
		assert.Equal(t, 0, cfg.Redis.DB, "Redis DB 编号应为 0")
		assert.Equal(t, 10, cfg.Redis.PoolSize, "Redis 连接池大小应为 10")

		// 验证 jwt 配置
		assert.NotEmpty(t, cfg.JWT.Secret, "JWT Secret 不应为空")
		assert.Equal(t, 24*time.Hour, cfg.JWT.Expiration, "JWT 过期时间应为 24h")
		assert.Equal(t, "weiyeston-v2", cfg.JWT.Issuer, "JWT 签发者应为 weiyeston-v2")

		// 验证 wechat 配置
		assert.Equal(t, "https://your-domain.com", cfg.Wechat.ServerURL, "微信服务器 URL 应正确")

		// 验证 upload 配置
		assert.Equal(t, "local", cfg.Upload.Driver, "上传驱动应为 local")
		assert.Equal(t, "./uploads", cfg.Upload.LocalPath, "本地路径应为 ./uploads")
		assert.Equal(t, 20, cfg.Upload.MaxSizeMB, "最大上传大小应为 20MB")

		// 验证 log 配置
		assert.Equal(t, "info", cfg.Log.Level, "日志级别应为 info")
		assert.Equal(t, "console", cfg.Log.Format, "日志格式应为 console")
		assert.Equal(t, "stdout", cfg.Log.Output, "日志输出应为 stdout")

		// 验证 AI 配置
		assert.Equal(t, "deepseek", cfg.AI.Provider, "AI 提供商应为 deepseek")
		assert.Equal(t, "https://api.deepseek.com", cfg.AI.BaseURL, "AI BaseURL 应正确")
		assert.Equal(t, "deepseek-chat", cfg.AI.Model, "AI 模型应为 deepseek-chat")
		assert.Equal(t, 60*time.Second, cfg.AI.Timeout, "AI 超时应为 60s")

		// 验证 CORS 配置
		assert.Contains(t, cfg.CORS.AllowedOrigins, "http://localhost:5173")
		assert.Contains(t, cfg.CORS.AllowedOrigins, "http://localhost:8080")
		assert.Contains(t, cfg.CORS.AllowedMethods, "GET")
		assert.Contains(t, cfg.CORS.AllowedMethods, "POST")
		assert.Contains(t, cfg.CORS.AllowedMethods, "PUT")
		assert.Contains(t, cfg.CORS.AllowedMethods, "DELETE")
		assert.Contains(t, cfg.CORS.AllowedHeaders, "Authorization")
		assert.Contains(t, cfg.CORS.AllowedHeaders, "Content-Type")
	})

	t.Run("加载不存在的配置文件", func(t *testing.T) {
		_, err := Load("nonexistent_config.yaml")
		require.Error(t, err, "加载不存在的文件应返回错误")
		assert.Contains(t, err.Error(), "读取配置文件失败", "错误信息应包含读取失败")
	})

	t.Run("加载格式错误的配置文件", func(t *testing.T) {
		// 创建一个临时无效 YAML 文件
		tmpDir := t.TempDir()
		invalidPath := filepath.Join(tmpDir, "invalid.yaml")
		err := os.WriteFile(invalidPath, []byte(": invalid yaml: :::"), 0644)
		require.NoError(t, err)

		_, err = Load(invalidPath)
		require.Error(t, err, "加载格式错误的文件应返回错误")
	})
}

// TestEnvVarOverride 测试环境变量覆盖（WEIYESTON_ 前缀）
func TestEnvVarOverride(t *testing.T) {
	t.Run("环境变量覆盖 server.port", func(t *testing.T) {
		os.Setenv("WEIYESTON_SERVER_PORT", "9090")
		defer os.Unsetenv("WEIYESTON_SERVER_PORT")

		configPath := filepath.Join("..", "..", "config.yaml")
		cfg, err := Load(configPath)
		require.NoError(t, err)

		assert.Equal(t, 9090, cfg.Server.Port, "环境变量应覆盖 server.port 为 9090")
	})

	t.Run("环境变量覆盖 server.mode", func(t *testing.T) {
		os.Setenv("WEIYESTON_SERVER_MODE", "release")
		defer os.Unsetenv("WEIYESTON_SERVER_MODE")

		configPath := filepath.Join("..", "..", "config.yaml")
		cfg, err := Load(configPath)
		require.NoError(t, err)

		assert.Equal(t, "release", cfg.Server.Mode, "环境变量应覆盖 server.mode 为 release")
	})

	t.Run("环境变量覆盖 database.host", func(t *testing.T) {
		os.Setenv("WEIYESTON_DATABASE_HOST", "db.example.com")
		defer os.Unsetenv("WEIYESTON_DATABASE_HOST")

		configPath := filepath.Join("..", "..", "config.yaml")
		cfg, err := Load(configPath)
		require.NoError(t, err)

		assert.Equal(t, "db.example.com", cfg.Database.Host, "环境变量应覆盖 database.host")
	})

	t.Run("环境变量覆盖 database.password", func(t *testing.T) {
		os.Setenv("WEIYESTON_DATABASE_PASSWORD", "secure-pass-123")
		defer os.Unsetenv("WEIYESTON_DATABASE_PASSWORD")

		configPath := filepath.Join("..", "..", "config.yaml")
		cfg, err := Load(configPath)
		require.NoError(t, err)

		assert.Equal(t, "secure-pass-123", cfg.Database.Password, "环境变量应覆盖 database.password")
	})

	t.Run("环境变量覆盖 redis.addr", func(t *testing.T) {
		os.Setenv("WEIYESTON_REDIS_ADDR", "redis.example.com:6380")
		defer os.Unsetenv("WEIYESTON_REDIS_ADDR")

		configPath := filepath.Join("..", "..", "config.yaml")
		cfg, err := Load(configPath)
		require.NoError(t, err)

		assert.Equal(t, "redis.example.com:6380", cfg.Redis.Addr, "环境变量应覆盖 redis.addr")
	})

	t.Run("环境变量覆盖 jwt.secret", func(t *testing.T) {
		os.Setenv("WEIYESTON_JWT_SECRET", "super-secret-key")
		defer os.Unsetenv("WEIYESTON_JWT_SECRET")

		configPath := filepath.Join("..", "..", "config.yaml")
		cfg, err := Load(configPath)
		require.NoError(t, err)

		assert.Equal(t, "super-secret-key", cfg.JWT.Secret, "环境变量应覆盖 jwt.secret")
	})

	t.Run("环境变量覆盖 wechat.component_app_id", func(t *testing.T) {
		os.Setenv("WEIYESTON_WECHAT_COMPONENT_APP_ID", "wx-test-app-id")
		defer os.Unsetenv("WEIYESTON_WECHAT_COMPONENT_APP_ID")

		configPath := filepath.Join("..", "..", "config.yaml")
		cfg, err := Load(configPath)
		require.NoError(t, err)

		assert.Equal(t, "wx-test-app-id", cfg.Wechat.ComponentAppID, "环境变量应覆盖 wechat.component_app_id")
	})

	t.Run("环境变量覆盖 ai.api_key", func(t *testing.T) {
		os.Setenv("WEIYESTON_AI_API_KEY", "sk-test-key-123")
		defer os.Unsetenv("WEIYESTON_AI_API_KEY")

		configPath := filepath.Join("..", "..", "config.yaml")
		cfg, err := Load(configPath)
		require.NoError(t, err)

		assert.Equal(t, "sk-test-key-123", cfg.AI.APIKey, "环境变量应覆盖 ai.api_key")
	})
}

// TestDefaults 测试默认值
func TestDefaults(t *testing.T) {
	t.Run("缺少 server 配置时使用默认值", func(t *testing.T) {
		tmpDir := t.TempDir()
		minimalPath := filepath.Join(tmpDir, "minimal.yaml")
		content := []byte(`
server:
  port: 8080
database:
  host: localhost
  port: 5432
  user: postgres
  password: postgres
  dbname: testdb
  sslmode: disable
redis:
  addr: localhost:6379
jwt:
  secret: test-secret
wechat:
  server_url: "https://test.com"
`)
		err := os.WriteFile(minimalPath, content, 0644)
		require.NoError(t, err)

		cfg, err := Load(minimalPath)
		require.NoError(t, err)

		// 验证未配置的字段有合理的默认值
		assert.Equal(t, "debug", cfg.Server.Mode, "server.mode 默认值应为 debug")
		assert.Greater(t, cfg.Server.Port, 0, "server.port 应有默认值")
		assert.Equal(t, 0, cfg.Redis.DB, "redis.db 默认值应为 0")
		assert.Equal(t, 10, cfg.Redis.PoolSize, "redis.pool_size 默认值应为 10")
		assert.Equal(t, "info", cfg.Log.Level, "log.level 默认值应为 info")
	})

	t.Run("upload 驱动默认为 local", func(t *testing.T) {
		tmpDir := t.TempDir()
		minimalPath := filepath.Join(tmpDir, "minimal2.yaml")
		content := []byte(`
server:
  port: 8080
database:
  host: localhost
  port: 5432
  user: postgres
  password: postgres
  dbname: testdb
  sslmode: disable
redis:
  addr: localhost:6379
jwt:
  secret: test-secret
wechat:
  server_url: "https://test.com"
upload:
  local_path: ./uploads
  max_size_mb: 10
`)
		err := os.WriteFile(minimalPath, content, 0644)
		require.NoError(t, err)

		cfg, err := Load(minimalPath)
		require.NoError(t, err)

		assert.Equal(t, "local", cfg.Upload.Driver, "upload.driver 默认值应为 local")
	})
}

// TestValidation 测试配置验证
func TestValidation(t *testing.T) {
	t.Run("缺少 database.host 应返回错误", func(t *testing.T) {
		tmpDir := t.TempDir()
		invalidPath := filepath.Join(tmpDir, "no_db_host.yaml")
		content := []byte(`
server:
  port: 8080
database:
  port: 5432
  user: postgres
  password: postgres
  dbname: testdb
  sslmode: disable
redis:
  addr: localhost:6379
jwt:
  secret: test-secret
wechat:
  server_url: "https://test.com"
`)
		err := os.WriteFile(invalidPath, content, 0644)
		require.NoError(t, err)

		cfg, err := Load(invalidPath)
		// 配置加载本身成功，但验证应发现缺失必填字段
		if err == nil {
			err = cfg.Validate()
			require.Error(t, err, "缺少必填字段应返回验证错误")
			assert.Contains(t, err.Error(), "database.host", "错误信息应提及缺失的字段")
		}
	})

	t.Run("缺少 jwt.secret 应返回错误", func(t *testing.T) {
		tmpDir := t.TempDir()
		invalidPath := filepath.Join(tmpDir, "no_jwt_secret.yaml")
		content := []byte(`
server:
  port: 8080
database:
  host: localhost
  port: 5432
  user: postgres
  password: postgres
  dbname: testdb
  sslmode: disable
redis:
  addr: localhost:6379
jwt:
  expiration: 24h
wechat:
  server_url: "https://test.com"
`)
		err := os.WriteFile(invalidPath, content, 0644)
		require.NoError(t, err)

		cfg, err := Load(invalidPath)
		if err == nil {
			err = cfg.Validate()
			require.Error(t, err, "缺少 jwt.secret 应返回验证错误")
			assert.Contains(t, err.Error(), "jwt.secret", "错误信息应提及缺失的字段")
		}
	})

	t.Run("缺少 redis.addr 应返回错误", func(t *testing.T) {
		tmpDir := t.TempDir()
		invalidPath := filepath.Join(tmpDir, "no_redis_addr.yaml")
		content := []byte(`
server:
  port: 8080
database:
  host: localhost
  port: 5432
  user: postgres
  password: postgres
  dbname: testdb
  sslmode: disable
redis:
  password: ""
jwt:
  secret: test-secret
wechat:
  server_url: "https://test.com"
`)
		err := os.WriteFile(invalidPath, content, 0644)
		require.NoError(t, err)

		cfg, err := Load(invalidPath)
		if err == nil {
			err = cfg.Validate()
			require.Error(t, err, "缺少 redis.addr 应返回验证错误")
			assert.Contains(t, err.Error(), "redis.addr", "错误信息应提及缺失的字段")
		}
	})

	t.Run("缺少 wechat.server_url 应返回错误", func(t *testing.T) {
		tmpDir := t.TempDir()
		invalidPath := filepath.Join(tmpDir, "no_wechat_url.yaml")
		content := []byte(`
server:
  port: 8080
database:
  host: localhost
  port: 5432
  user: postgres
  password: postgres
  dbname: testdb
  sslmode: disable
redis:
  addr: localhost:6379
jwt:
  secret: test-secret
wechat:
  component_app_id: ""
`)
		err := os.WriteFile(invalidPath, content, 0644)
		require.NoError(t, err)

		cfg, err := Load(invalidPath)
		if err == nil {
			err = cfg.Validate()
			require.Error(t, err, "缺少 wechat.server_url 应返回验证错误")
			assert.Contains(t, err.Error(), "wechat.server_url", "错误信息应提及缺失的字段")
		}
	})
}

// TestConfigStructTypes 测试配置结构体类型正确性
func TestConfigStructTypes(t *testing.T) {
	t.Run("Config 结构体包含所有必要的嵌套配置", func(t *testing.T) {
		configPath := filepath.Join("..", "..", "config.yaml")
		cfg, err := Load(configPath)
		require.NoError(t, err)

		// 确保所有子配置不为 nil
		assert.NotNil(t, cfg)
		// ServerConfig 字段存在
		assert.IsType(t, ServerConfig{}, cfg.Server)
		assert.IsType(t, DatabaseConfig{}, cfg.Database)
		assert.IsType(t, RedisConfig{}, cfg.Redis)
		assert.IsType(t, JWTConfig{}, cfg.JWT)
		assert.IsType(t, WechatConfig{}, cfg.Wechat)
		assert.IsType(t, UploadConfig{}, cfg.Upload)
		assert.IsType(t, LogConfig{}, cfg.Log)
		assert.IsType(t, AIConfig{}, cfg.AI)
		assert.IsType(t, CORSConfig{}, cfg.CORS)
	})

	t.Run("指针字段可以正确处理", func(t *testing.T) {
		configPath := filepath.Join("..", "..", "config.yaml")
		cfg, err := Load(configPath)
		require.NoError(t, err)

		// s3_endpoint、s3_bucket 等可能为空字符串
		assert.IsType(t, "", cfg.Upload.S3Endpoint)
		assert.IsType(t, "", cfg.Upload.S3Key)
		assert.IsType(t, "", cfg.Upload.S3Secret)
	})
}
