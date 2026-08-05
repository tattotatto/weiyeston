package middleware

import (
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// RateLimitConfig 限流配置
type RateLimitConfig struct {
	Enabled   bool   `mapstructure:"enabled"    yaml:"enabled"`
	Rate      int    `mapstructure:"rate"       yaml:"rate"`       // 每秒允许的请求数
	Burst     int    `mapstructure:"burst"      yaml:"burst"`      // 突发请求数
	KeyPrefix string `mapstructure:"key_prefix" yaml:"key_prefix"` // Redis key 前缀
}

// tokenBucket 简单的内存令牌桶
type tokenBucket struct {
	mu         sync.Mutex
	tokens     float64
	lastRefill time.Time
	lastAccess time.Time
	rate       float64 // 每秒补充的令牌数
	burst      float64 // 最大令牌数
}

// allow 检查是否允许请求，返回 (允许, 等待秒数)
func (tb *tokenBucket) allow() (bool, float64) {
	tb.mu.Lock()
	defer tb.mu.Unlock()

	now := time.Now()
	elapsed := now.Sub(tb.lastRefill).Seconds()

	// 补充令牌
	tb.tokens += elapsed * tb.rate
	if tb.tokens > tb.burst {
		tb.tokens = tb.burst
	}
	tb.lastRefill = now
	tb.lastAccess = now

	if tb.tokens >= 1.0 {
		tb.tokens -= 1.0
		return true, 0
	}

	// 计算需要等待的时间
	waitSeconds := (1.0 - tb.tokens) / tb.rate
	return false, waitSeconds
}

// bucketStore 存储每个 IP 的令牌桶
type bucketStore struct {
	mu      sync.Mutex
	buckets map[string]*tokenBucket
	config  RateLimitConfig
}

// getOrCreate 获取或创建 IP 对应的令牌桶
func (bs *bucketStore) getOrCreate(key string) *tokenBucket {
	bs.mu.Lock()
	defer bs.mu.Unlock()

	if bucket, ok := bs.buckets[key]; ok {
		return bucket
	}

	burst := float64(bs.config.Burst)
	if burst <= 0 {
		burst = 1
	}

	bucket := &tokenBucket{
		tokens:     burst, // 初始令牌数为 burst
		lastRefill: time.Now(),
		lastAccess: time.Now(),
		rate:       float64(bs.config.Rate),
		burst:      burst,
	}
	bs.buckets[key] = bucket
	return bucket
}

// cleanup 清理超过 idleTimeout 未访问的令牌桶
func (bs *bucketStore) cleanup(idleTimeout time.Duration) {
	bs.mu.Lock()
	defer bs.mu.Unlock()

	now := time.Now()
	for key, bucket := range bs.buckets {
		bucket.mu.Lock()
		idle := now.Sub(bucket.lastAccess)
		bucket.mu.Unlock()
		if idle > idleTimeout {
			delete(bs.buckets, key)
		}
	}
}

// RateLimit 限流中间件
// 基于令牌桶算法，限制每个 IP/用户的请求频率（T0 阶段使用内存实现）
func RateLimit(config RateLimitConfig) gin.HandlerFunc {
	store := &bucketStore{
		buckets: make(map[string]*tokenBucket),
		config:  config,
	}

	// 后台 goroutine 定期清理超过 5 分钟未使用的令牌桶
	go func() {
		ticker := time.NewTicker(1 * time.Minute)
		defer ticker.Stop()
		for range ticker.C {
			store.cleanup(5 * time.Minute)
		}
	}()

	return func(c *gin.Context) {
		// 限流禁用时跳过
		if !config.Enabled {
			c.Next()
			return
		}

		// 生成限流 key（基于客户端 IP）
		key := config.KeyPrefix + ":" + c.ClientIP()

		bucket := store.getOrCreate(key)
		allowed, waitSeconds := bucket.allow()

		if !allowed {
			c.Header("Retry-After", strconv.Itoa(int(waitSeconds)+1))
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"code": 429,
				"msg":  "请求过于频繁，请稍后重试",
			})
			return
		}

		c.Next()
	}
}
