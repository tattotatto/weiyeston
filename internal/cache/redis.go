// Package cache Redis 缓存封装
// 提供 Get/Set/Del/Incr/Expire 常用操作，统一错误处理
package cache

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
)

// Client Redis 客户端封装
type Client struct {
	rdb *redis.Client
}

// New 创建缓存客户端
func New(rdb *redis.Client) *Client {
	return &Client{rdb: rdb}
}

// Get 获取键对应的值，key 不存在时返回空字符串和 nil error
func (c *Client) Get(ctx context.Context, key string) (string, error) {
	val, err := c.rdb.Get(ctx, key).Result()
	if err == redis.Nil {
		return "", nil
	}
	return val, err
}

// Set 设置键值对，可指定过期时间
func (c *Client) Set(ctx context.Context, key string, value string, expiration time.Duration) error {
	return c.rdb.Set(ctx, key, value, expiration).Err()
}

// Del 删除一个或多个键
func (c *Client) Del(ctx context.Context, keys ...string) error {
	return c.rdb.Del(ctx, keys...).Err()
}

// Incr 原子递增计数器，返回递增后的值
func (c *Client) Incr(ctx context.Context, key string) (int64, error) {
	return c.rdb.Incr(ctx, key).Result()
}

// Expire 设置键的过期时间
func (c *Client) Expire(ctx context.Context, key string, expiration time.Duration) error {
	return c.rdb.Expire(ctx, key, expiration).Err()
}

// Raw 返回底层的 redis.Client，供需要高级操作时使用
func (c *Client) Raw() *redis.Client {
	return c.rdb
}
