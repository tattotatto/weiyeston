// Package storage 文件存储抽象层
package storage

import (
	"context"
	"io"
)

// Provider 存储驱动接口
type Provider interface {
	Upload(ctx context.Context, key string, reader io.Reader, contentType string) (string, error)
	Delete(ctx context.Context, key string) error
	GetURL(key string) string
}
