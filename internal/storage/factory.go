package storage

import (
	"fmt"
	"strings"
)

// Config 存储配置
type Config struct {
	Driver     string // local | s3
	LocalPath  string // 本地存储路径
	BaseURL    string // 本地存储URL前缀
	S3Endpoint string // S3 Endpoint
	S3Bucket   string // S3 Bucket
	S3Region   string // S3 Region
	S3Key      string // S3 Access Key
	S3Secret   string // S3 Secret Key
	PublicURL  string // 自定义CDN域名
}

// NewProvider 根据配置创建存储驱动
func NewProvider(cfg Config) (Provider, error) {
	driver := strings.ToLower(strings.TrimSpace(cfg.Driver))
	if driver == "" {
		driver = "local"
	}

	switch driver {
	case "local":
		basePath := cfg.LocalPath
		if basePath == "" {
			basePath = "./uploads"
		}
		baseURL := cfg.BaseURL
		if baseURL == "" {
			baseURL = "/uploads"
		}
		return NewLocalProvider(basePath, baseURL), nil

	case "s3":
		if cfg.S3Endpoint == "" {
			return nil, fmt.Errorf("S3 Endpoint 不能为空")
		}
		if cfg.S3Bucket == "" {
			return nil, fmt.Errorf("S3 Bucket 不能为空")
		}
		if cfg.S3Region == "" {
			return nil, fmt.Errorf("S3 Region 不能为空")
		}
		return NewS3Provider(S3Config{
			Endpoint:  cfg.S3Endpoint,
			Bucket:    cfg.S3Bucket,
			Region:    cfg.S3Region,
			AccessKey: cfg.S3Key,
			SecretKey: cfg.S3Secret,
			PublicURL: cfg.PublicURL,
		})

	default:
		return nil, fmt.Errorf("不支持的存储驱动: %s", driver)
	}
}
