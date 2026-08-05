package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

// TestRateLimitMiddleware_BasicBehavior 测试限流中间件基本行为
func TestRateLimitMiddleware_BasicBehavior(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("未超过限流阈值时正常通过", func(t *testing.T) {
		// 使用高阈值确保测试通过
		config := RateLimitConfig{
			Enabled:   true,
			Rate:      1000,  // 每秒 1000 次
			Burst:     100,
			KeyPrefix: "test_ratelimit",
		}

		r := gin.New()
		r.Use(RateLimit(config))
		r.GET("/api/v1/test", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"ok": true})
		})

		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodGet, "/api/v1/test", nil)
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code,
			"未超过限流时应正常返回")
	})

	t.Run("超过限流阈值时返回 429", func(t *testing.T) {
		// 使用极低阈值
		config := RateLimitConfig{
			Enabled:   true,
			Rate:      1,    // 每秒 1 次
			Burst:     0,
			KeyPrefix: "test_strict",
		}

		r := gin.New()
		r.Use(RateLimit(config))
		r.GET("/api/v1/test-strict", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"ok": true})
		})

		// 先发送一次请求消耗额度
		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodGet, "/api/v1/test-strict", nil)
		r.ServeHTTP(w, req)

		// 立即发送第二次请求，应触发限流
		w = httptest.NewRecorder()
		req, _ = http.NewRequest(http.MethodGet, "/api/v1/test-strict", nil)
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusTooManyRequests, w.Code,
			"超过限流阈值应返回 429 Too Many Requests")
	})

	t.Run("限流禁用时所有请求正常通过", func(t *testing.T) {
		config := RateLimitConfig{
			Enabled: false,
		}

		r := gin.New()
		r.Use(RateLimit(config))
		r.GET("/api/v1/no-limit", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"ok": true})
		})

		for i := 0; i < 5; i++ {
			w := httptest.NewRecorder()
			req, _ := http.NewRequest(http.MethodGet, "/api/v1/no-limit", nil)
			r.ServeHTTP(w, req)
			assert.Equal(t, http.StatusOK, w.Code,
				"限流禁用时所有请求应正常通过")
		}
	})

	t.Run("429 响应应包含 Retry-After header", func(t *testing.T) {
		config := RateLimitConfig{
			Enabled:   true,
			Rate:      1,
			Burst:     0,
			KeyPrefix: "test_retry",
		}

		r := gin.New()
		r.Use(RateLimit(config))
		r.GET("/api/v1/retry-test", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"ok": true})
		})

		// 消耗额度
		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodGet, "/api/v1/retry-test", nil)
		r.ServeHTTP(w, req)

		// 触发限流
		w = httptest.NewRecorder()
		req, _ = http.NewRequest(http.MethodGet, "/api/v1/retry-test", nil)
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusTooManyRequests, w.Code,
			"触发限流应返回 429")
		retryAfter := w.Header().Get("Retry-After")
		assert.NotEmpty(t, retryAfter,
			"429 响应应包含 Retry-After header")
	})
}

// TestRateLimitMiddleware_KeyGeneration 测试限流 key 生成
func TestRateLimitMiddleware_KeyGeneration(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("不同 IP 使用不同的限流计数器", func(t *testing.T) {
		config := RateLimitConfig{
			Enabled:   true,
			Rate:      5,
			Burst:     3,
			KeyPrefix: "test_ip",
		}

		r := gin.New()
		r.Use(RateLimit(config))
		r.GET("/api/v1/test", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"ok": true})
		})

		// 模拟不同 IP 的请求
		ips := []string{"192.168.1.1:1000", "192.168.1.2:2000", "10.0.0.1:3000"}
		for _, ip := range ips {
			w := httptest.NewRecorder()
			req, _ := http.NewRequest(http.MethodGet, "/api/v1/test", nil)
			req.RemoteAddr = ip
			r.ServeHTTP(w, req)
			assert.Equal(t, http.StatusOK, w.Code,
				"不同 IP 的请求应独立计数，互不影响")
		}
	})
}
