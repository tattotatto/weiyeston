package storage

import (
	"bytes"
	"context"
	"fmt"
	"io"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// S3Provider S3兼容存储驱动（支持阿里云OSS/腾讯COS/MinIO/AWS S3）
type S3Provider struct {
	client    *s3.Client
	bucket    string
	publicURL string // 自定义CDN域名，为空时使用S3默认URL
}

// S3Config S3存储配置
type S3Config struct {
	Endpoint  string // S3兼容Endpoint
	Bucket    string // 存储桶名称
	Region    string // 区域
	AccessKey string // 访问密钥
	SecretKey string // 秘密密钥
	PublicURL string // 自定义CDN域名（可选）
}

// NewS3Provider 创建S3兼容存储驱动
func NewS3Provider(cfg S3Config) (*S3Provider, error) {
	resolver := aws.EndpointResolverWithOptionsFunc(func(service, region string, options ...interface{}) (aws.Endpoint, error) {
		return aws.Endpoint{
			URL:               cfg.Endpoint,
			SigningRegion:     cfg.Region,
			HostnameImmutable: true,
		}, nil
	})

	awsCfg, err := config.LoadDefaultConfig(context.Background(),
		config.WithRegion(cfg.Region),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(cfg.AccessKey, cfg.SecretKey, "")),
		config.WithEndpointResolverWithOptions(resolver),
	)
	if err != nil {
		return nil, fmt.Errorf("加载AWS配置失败: %w", err)
	}

	client := s3.NewFromConfig(awsCfg, func(o *s3.Options) {
		o.UsePathStyle = true // 兼容MinIO等使用路径风格的S3服务
	})

	return &S3Provider{
		client:    client,
		bucket:    cfg.Bucket,
		publicURL: cfg.PublicURL,
	}, nil
}

// Upload 上传文件到S3
func (p *S3Provider) Upload(ctx context.Context, key string, reader io.Reader, contentType string) (string, error) {
	// 读取全部内容到 buffer（S3 PutObject 需要内容长度）
	data, err := io.ReadAll(reader)
	if err != nil {
		return "", fmt.Errorf("读取文件内容失败: %w", err)
	}

	_, err = p.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(p.bucket),
		Key:         aws.String(key),
		Body:        bytes.NewReader(data),
		ContentType: aws.String(contentType),
	})
	if err != nil {
		return "", fmt.Errorf("上传到S3失败: %w", err)
	}

	return p.GetURL(key), nil
}

// Delete 从S3删除文件
func (p *S3Provider) Delete(ctx context.Context, key string) error {
	_, err := p.client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(p.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return fmt.Errorf("从S3删除文件失败: %w", err)
	}
	return nil
}

// GetURL 获取文件访问URL
func (p *S3Provider) GetURL(key string) string {
	if p.publicURL != "" {
		return p.publicURL + "/" + key
	}
	// 回退：返回相对路径格式，实际项目应配置publicURL
	return "/" + key
}
