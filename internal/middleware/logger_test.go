package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
)

// TestLoggerMiddleware_RequestLogFormat 测试请求日志格式
func TestLoggerMiddleware_RequestLogFormat(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("请求成功时应记录方法、路径、状态码、耗时", func(t *testing.T) {
		core, recorded := observer.New(zapcore.InfoLevel)
		logger := zap.New(core)

		r := gin.New()
		r.Use(Logger(logger))
		r.GET("/api/v1/test", func(c *gin.Context) {
			time.Sleep(5 * time.Millisecond)
			c.String(http.StatusOK, "ok")
		})

		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodGet, "/api/v1/test", nil)
		req.RemoteAddr = "127.0.0.1:12345"
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		// 验证日志记录
		logs := recorded.All()
		assert.GreaterOrEqual(t, len(logs), 1, "应至少有一条日志记录")
		logEntry := logs[len(logs)-1]
		fields := logEntry.ContextMap()

		// 验证基本字段
		assert.Contains(t, fields, "method", "日志应包含 method 字段")
		assert.Contains(t, fields, "path", "日志应包含 path 字段")
		assert.Contains(t, fields, "status", "日志应包含 status 字段")
		assert.Contains(t, fields, "latency", "日志应包含 latency 字段")

		assert.Equal(t, "GET", fields["method"])
		assert.Equal(t, "/api/v1/test", fields["path"])
		assert.Equal(t, int64(200), fields["status"])
	})

	t.Run("请求失败时应记录错误状态码", func(t *testing.T) {
		core, recorded := observer.New(zapcore.InfoLevel)
		logger := zap.New(core)

		r := gin.New()
		r.Use(Logger(logger))
		r.GET("/api/v1/not-found", func(c *gin.Context) {
			c.JSON(http.StatusNotFound, gin.H{"msg": "not found"})
		})

		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodGet, "/api/v1/not-found", nil)
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)

		logs := recorded.All()
		assert.GreaterOrEqual(t, len(logs), 1, "应至少有一条日志记录")
		logEntry := logs[len(logs)-1]
		fields := logEntry.ContextMap()
		assert.Equal(t, int64(404), fields["status"],
			"日志应记录 404 状态码")
	})

	t.Run("请求应记录客户端 IP", func(t *testing.T) {
		core, recorded := observer.New(zapcore.InfoLevel)
		logger := zap.New(core)

		r := gin.New()
		r.Use(Logger(logger))
		r.GET("/api/v1/test", func(c *gin.Context) {
			c.String(http.StatusOK, "ok")
		})

		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodGet, "/api/v1/test", nil)
		req.RemoteAddr = "192.168.1.100:54321"
		req.Header.Set("X-Forwarded-For", "10.0.0.1")
		r.ServeHTTP(w, req)

		logs := recorded.All()
		assert.GreaterOrEqual(t, len(logs), 1, "应至少有一条日志记录")
		logEntry := logs[len(logs)-1]
		fields := logEntry.ContextMap()
		// 应包含 client_ip 或 ip 字段
		hasIP := fields["client_ip"] != nil || fields["ip"] != nil
		assert.True(t, hasIP, "日志应包含客户端 IP 信息")
	})

	t.Run("请求应记录 User-Agent", func(t *testing.T) {
		core, recorded := observer.New(zapcore.InfoLevel)
		logger := zap.New(core)

		r := gin.New()
		r.Use(Logger(logger))
		r.GET("/api/v1/test", func(c *gin.Context) {
			c.String(http.StatusOK, "ok")
		})

		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodGet, "/api/v1/test", nil)
		req.Header.Set("User-Agent", "Mozilla/5.0 TestBrowser")
		r.ServeHTTP(w, req)

		logs := recorded.All()
		assert.GreaterOrEqual(t, len(logs), 1, "应至少有一条日志记录")
		logEntry := logs[len(logs)-1]
		fields := logEntry.ContextMap()
		hasUA := fields["user_agent"] != nil || fields["ua"] != nil
		assert.True(t, hasUA, "日志应包含 User-Agent 信息")
	})

	t.Run("请求耗时应以毫秒或适当的单位记录", func(t *testing.T) {
		core, recorded := observer.New(zapcore.InfoLevel)
		logger := zap.New(core)

		r := gin.New()
		r.Use(Logger(logger))
		r.GET("/api/v1/slow", func(c *gin.Context) {
			time.Sleep(50 * time.Millisecond)
			c.String(http.StatusOK, "slow")
		})

		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodGet, "/api/v1/slow", nil)
		r.ServeHTTP(w, req)

		logs := recorded.All()
		assert.GreaterOrEqual(t, len(logs), 1, "应至少有一条日志记录")
		logEntry := logs[len(logs)-1]
		fields := logEntry.ContextMap()
		latency, ok := fields["latency"]
		assert.True(t, ok, "日志应包含 latency 字段")
		// latency 应为数值类型（毫秒或 Duration）
		switch v := latency.(type) {
		case float64:
			assert.Greater(t, v, float64(0), "latency 应大于 0")
		case int64:
			assert.Greater(t, v, int64(0), "latency 应大于 0")
		default:
			assert.NotNil(t, v, "latency 应有值")
		}
	})
}

// TestLoggerMiddleware_LogLevels 测试不同状态的日志级别
func TestLoggerMiddleware_LogLevels(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("4xx 请求应记录为 warn 级别", func(t *testing.T) {
		core, recorded := observer.New(zapcore.WarnLevel)
		logger := zap.New(core)

		r := gin.New()
		r.Use(Logger(logger))
		r.GET("/api/v1/forbidden", func(c *gin.Context) {
			c.JSON(http.StatusForbidden, gin.H{"msg": "forbidden"})
		})

		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodGet, "/api/v1/forbidden", nil)
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusForbidden, w.Code)

		logs := recorded.All()
		assert.GreaterOrEqual(t, len(logs), 1, "4xx 请求应产生 warn 级别日志")
	})

	t.Run("5xx 请求应记录为 error 级别", func(t *testing.T) {
		core, recorded := observer.New(zapcore.ErrorLevel)
		logger := zap.New(core)

		r := gin.New()
		r.Use(Logger(logger))
		r.GET("/api/v1/error", func(c *gin.Context) {
			c.JSON(http.StatusInternalServerError, gin.H{"msg": "internal error"})
		})

		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodGet, "/api/v1/error", nil)
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusInternalServerError, w.Code)

		logs := recorded.All()
		assert.GreaterOrEqual(t, len(logs), 1, "5xx 请求应产生 error 级别日志")
	})

	t.Run("2xx 请求应记录为 info 级别", func(t *testing.T) {
		core, recorded := observer.New(zapcore.InfoLevel)
		logger := zap.New(core)

		r := gin.New()
		r.Use(Logger(logger))
		r.GET("/api/v1/success", func(c *gin.Context) {
			c.String(http.StatusOK, "success")
		})

		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodGet, "/api/v1/success", nil)
		r.ServeHTTP(w, req)

		logs := recorded.All()
		assert.GreaterOrEqual(t, len(logs), 1, "2xx 请求应产生 info 日志")
	})
}

// TestLoggerMiddleware_RequestID 测试请求 ID 字段
func TestLoggerMiddleware_RequestID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("请求日志应包含 request_id 或 trace_id", func(t *testing.T) {
		core, recorded := observer.New(zapcore.InfoLevel)
		logger := zap.New(core)

		r := gin.New()
		r.Use(Logger(logger))
		r.GET("/api/v1/test", func(c *gin.Context) {
			c.String(http.StatusOK, "ok")
		})

		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodGet, "/api/v1/test", nil)
		r.ServeHTTP(w, req)

		logs := recorded.All()
		assert.GreaterOrEqual(t, len(logs), 1, "应至少有一条日志记录")
		logEntry := logs[len(logs)-1]
		fields := logEntry.ContextMap()

		hasRequestID := fields["request_id"] != nil ||
			fields["trace_id"] != nil ||
			fields["req_id"] != nil
		assert.True(t, hasRequestID, "日志应包含 request_id 或 trace_id")
	})
}

// TestLoggerMiddleware_MultipleRequests 测试多个请求的日志顺序
func TestLoggerMiddleware_MultipleRequests(t *testing.T) {
	gin.SetMode(gin.TestMode)

	core, recorded := observer.New(zapcore.InfoLevel)
	logger := zap.New(core)

	r := gin.New()
	r.Use(Logger(logger))
	r.GET("/api/v1/echo", func(c *gin.Context) {
		c.String(http.StatusOK, "echo")
	})

	// 发送多个请求
	for i := 0; i < 3; i++ {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodGet, "/api/v1/echo", nil)
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
	}

	logs := recorded.All()
	assert.GreaterOrEqual(t, len(logs), 3,
		"每个请求应各产生一条日志")
}
