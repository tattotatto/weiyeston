package storage

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// LocalProvider 本地文件存储
type LocalProvider struct {
	basePath string // 本地存储根目录，如 ./uploads
	baseURL  string // 访问URL前缀，如 /uploads
}

// NewLocalProvider 创建本地存储驱动
func NewLocalProvider(basePath, baseURL string) *LocalProvider {
	if baseURL == "" {
		baseURL = "/uploads"
	}
	return &LocalProvider{basePath: basePath, baseURL: baseURL}
}

// Upload 保存文件到本地目录
func (p *LocalProvider) Upload(ctx context.Context, key string, reader io.Reader, contentType string) (string, error) {
	// 确保目录存在
	dir := filepath.Dir(filepath.Join(p.basePath, key))
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", fmt.Errorf("创建目录失败: %w", err)
	}

	savePath := filepath.Join(p.basePath, key)
	out, err := os.Create(savePath)
	if err != nil {
		return "", fmt.Errorf("创建文件失败: %w", err)
	}
	defer out.Close()

	if _, err := io.Copy(out, reader); err != nil {
		return "", fmt.Errorf("写入文件失败: %w", err)
	}

	return p.GetURL(key), nil
}

// Delete 删除本地文件
func (p *LocalProvider) Delete(ctx context.Context, key string) error {
	filePath := filepath.Join(p.basePath, key)
	if err := os.Remove(filePath); err != nil {
		if os.IsNotExist(err) {
			return nil // 文件不存在视为删除成功
		}
		return fmt.Errorf("删除文件失败: %w", err)
	}
	return nil
}

// GetURL 获取文件访问URL
func (p *LocalProvider) GetURL(key string) string {
	return p.baseURL + "/" + key
}
