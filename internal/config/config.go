// Package config 配置结构体定义 + 加载逻辑
package config

import (
	"fmt"
	"strings"
	"time"

	"github.com/spf13/viper"
)

// ========== 顶层配置 ==========

// Config 应用总配置
type Config struct {
	Server   ServerConfig   `mapstructure:"server"   yaml:"server"`
	Database DatabaseConfig `mapstructure:"database" yaml:"database"`
	Redis    RedisConfig    `mapstructure:"redis"    yaml:"redis"`
	JWT      JWTConfig      `mapstructure:"jwt"      yaml:"jwt"`
	Wechat   WechatConfig   `mapstructure:"wechat"   yaml:"wechat"`
	Upload   UploadConfig   `mapstructure:"upload"   yaml:"upload"`
	Log      LogConfig      `mapstructure:"log"      yaml:"log"`
	AI       AIConfig       `mapstructure:"ai"       yaml:"ai"`
	CORS     CORSConfig     `mapstructure:"cors"     yaml:"cors"`
}

// ========== 服务端配置 ==========

// ServerConfig HTTP 服务端配置
type ServerConfig struct {
	Port         int           `mapstructure:"port"          yaml:"port"`
	Mode         string        `mapstructure:"mode"          yaml:"mode"`
	ReadTimeout  time.Duration `mapstructure:"read_timeout"  yaml:"read_timeout"`
	WriteTimeout time.Duration `mapstructure:"write_timeout" yaml:"write_timeout"`
}

// ========== 数据库配置 ==========

// DatabaseConfig PostgreSQL 连接配置
type DatabaseConfig struct {
	Host            string        `mapstructure:"host"              yaml:"host"`
	Port            int           `mapstructure:"port"              yaml:"port"`
	User            string        `mapstructure:"user"              yaml:"user"`
	Password        string        `mapstructure:"password"          yaml:"password"`
	DBName          string        `mapstructure:"dbname"            yaml:"dbname"`
	SSLMode         string        `mapstructure:"sslmode"           yaml:"sslmode"`
	MaxOpenConns    int           `mapstructure:"max_open_conns"    yaml:"max_open_conns"`
	MaxIdleConns    int           `mapstructure:"max_idle_conns"    yaml:"max_idle_conns"`
	ConnMaxLifetime time.Duration `mapstructure:"conn_max_lifetime" yaml:"conn_max_lifetime"`
	ConnMaxIdleTime time.Duration `mapstructure:"conn_max_idle_time" yaml:"conn_max_idle_time"`
}

// ========== Redis 配置 ==========

// RedisConfig Redis 连接配置
type RedisConfig struct {
	Addr     string `mapstructure:"addr"      yaml:"addr"`
	Password string `mapstructure:"password"  yaml:"password"`
	DB       int    `mapstructure:"db"        yaml:"db"`
	PoolSize int    `mapstructure:"pool_size" yaml:"pool_size"`
}

// ========== JWT 配置 ==========

// JWTConfig JWT 认证配置
type JWTConfig struct {
	Secret            string        `mapstructure:"secret"             yaml:"secret"`
	Expiration        time.Duration `mapstructure:"expiration"         yaml:"expiration"`
	AccessExpiration  time.Duration `mapstructure:"access_expiration"  yaml:"access_expiration"`
	RefreshExpiration time.Duration `mapstructure:"refresh_expiration" yaml:"refresh_expiration"`
	Issuer            string        `mapstructure:"issuer"             yaml:"issuer"`
}

// GetAccessExpiration 获取 Access Token 过期时间
// 优先使用 AccessExpiration，未配置时回退到 Expiration
func (j *JWTConfig) GetAccessExpiration() time.Duration {
	if j.AccessExpiration > 0 {
		return j.AccessExpiration
	}
	return j.Expiration
}

// GetRefreshExpiration 获取 Refresh Token 过期时间
// 优先使用 RefreshExpiration，未配置时回退到 7 * Expiration
func (j *JWTConfig) GetRefreshExpiration() time.Duration {
	if j.RefreshExpiration > 0 {
		return j.RefreshExpiration
	}
	return 7 * 24 * time.Hour
}

// ========== 微信配置 ==========

// WechatConfig 微信公众号第三方平台配置
type WechatConfig struct {
	ComponentAppID     string `mapstructure:"component_app_id"     yaml:"component_app_id"`
	ComponentAppSecret string `mapstructure:"component_app_secret" yaml:"component_app_secret"`
	Token              string `mapstructure:"token"                yaml:"token"`
	EncodingAESKey     string `mapstructure:"encoding_aes_key"     yaml:"encoding_aes_key"`
	ServerURL          string `mapstructure:"server_url"           yaml:"server_url"`
}

// ========== 上传配置 ==========

// UploadConfig 文件上传配置
type UploadConfig struct {
	Driver     string `mapstructure:"driver"      yaml:"driver"` // local / s3
	LocalPath  string `mapstructure:"local_path"  yaml:"local_path"`
	MaxSizeMB  int    `mapstructure:"max_size_mb" yaml:"max_size_mb"`
	S3Endpoint string `mapstructure:"s3_endpoint" yaml:"s3_endpoint"`
	S3Bucket   string `mapstructure:"s3_bucket"   yaml:"s3_bucket"`
	S3Region   string `mapstructure:"s3_region"   yaml:"s3_region"`
	S3Key      string `mapstructure:"s3_key"      yaml:"s3_key"`
	S3Secret   string `mapstructure:"s3_secret"   yaml:"s3_secret"`
	PublicURL  string `mapstructure:"public_url"  yaml:"public_url"` // 自定义CDN域名
}

// ========== 日志配置 ==========

// LogConfig 日志配置
type LogConfig struct {
	Level  string `mapstructure:"level"  yaml:"level"`  // debug/info/warn/error
	Format string `mapstructure:"format" yaml:"format"` // json/console
	Output string `mapstructure:"output" yaml:"output"` // stdout/file
	File   string `mapstructure:"file"   yaml:"file"`
}

// ========== AI 配置 ==========

// AIConfig AI 接口配置
type AIConfig struct {
	Provider string        `mapstructure:"provider" yaml:"provider"` // openai/deepseek/qwen
	BaseURL  string        `mapstructure:"base_url" yaml:"base_url"`
	APIKey   string        `mapstructure:"api_key"  yaml:"api_key"`
	Model    string        `mapstructure:"model"    yaml:"model"`
	Timeout  time.Duration `mapstructure:"timeout"  yaml:"timeout"`
}

// ========== CORS 配置 ==========

// CORSConfig 跨域配置
type CORSConfig struct {
	AllowedOrigins []string `mapstructure:"allowed_origins" yaml:"allowed_origins"`
	AllowedMethods []string `mapstructure:"allowed_methods" yaml:"allowed_methods"`
	AllowedHeaders []string `mapstructure:"allowed_headers" yaml:"allowed_headers"`
}

// ========== 配置加载 ==========

// Load 从指定路径加载配置文件，支持环境变量覆盖（WEIYESTON_ 前缀）
func Load(path string) (*Config, error) {
	v := viper.New()
	v.SetConfigFile(path)

	// 设置默认值
	setDefaults(v)

	// 读取配置文件
	if err := v.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("读取配置文件失败: %w", err)
	}

	// 启用环境变量覆盖：WEIYESTON_DATABASE_HOST → database.host
	v.SetEnvPrefix("WEIYESTON")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("解析配置失败: %w", err)
	}

	return &cfg, nil
}

// setDefaults 为 viper 设置合理的默认值
func setDefaults(v *viper.Viper) {
	// server 默认值
	v.SetDefault("server.mode", "debug")
	v.SetDefault("server.read_timeout", "30s")
	v.SetDefault("server.write_timeout", "60s")

	// database 默认值
	v.SetDefault("database.port", 5432)
	v.SetDefault("database.sslmode", "disable")
	v.SetDefault("database.max_open_conns", 25)
	v.SetDefault("database.max_idle_conns", 5)
	v.SetDefault("database.conn_max_lifetime", "5m")
	v.SetDefault("database.conn_max_idle_time", "1m")

	// redis 默认值
	v.SetDefault("redis.db", 0)
	v.SetDefault("redis.pool_size", 10)

	// jwt 默认值
	v.SetDefault("jwt.expiration", "24h")
	v.SetDefault("jwt.issuer", "weiyeston-v2")

	// upload 默认值
	v.SetDefault("upload.driver", "local")
	v.SetDefault("upload.max_size_mb", 20)
	v.SetDefault("upload.public_url", "")

	// log 默认值
	v.SetDefault("log.level", "info")
	v.SetDefault("log.format", "console")
	v.SetDefault("log.output", "stdout")

	// ai 默认值
	v.SetDefault("ai.provider", "deepseek")
	v.SetDefault("ai.timeout", "60s")
}

// Validate 验证配置的必填字段
func (c *Config) Validate() error {
	var errs []string

	if c.Database.Host == "" {
		errs = append(errs, "database.host 是必填字段")
	}
	if c.JWT.Secret == "" {
		errs = append(errs, "jwt.secret 是必填字段")
	}
	if c.Redis.Addr == "" {
		errs = append(errs, "redis.addr 是必填字段")
	}
	if c.Wechat.ServerURL == "" {
		errs = append(errs, "wechat.server_url 是必填字段")
	}

	// 第三方平台配置：仅平台授权功能启用时需要
	// 如果不使用平台授权（仅手动接入），这些字段可留空
	if c.Wechat.ComponentAppID != "" {
		if c.Wechat.ComponentAppSecret == "" {
			errs = append(errs, "wechat.component_app_secret 是必填字段（已配置 component_app_id）")
		}
		if c.Wechat.Token == "" {
			errs = append(errs, "wechat.token 是必填字段（已配置 component_app_id）")
		}
		if c.Wechat.EncodingAESKey == "" {
			errs = append(errs, "wechat.encoding_aes_key 是必填字段（已配置 component_app_id）")
		}
		if len(c.Wechat.EncodingAESKey) != 0 && len(c.Wechat.EncodingAESKey) != 43 {
			errs = append(errs, "wechat.encoding_aes_key 必须是 43 位字符串")
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("配置验证失败: %s", strings.Join(errs, "; "))
	}

	return nil
}
