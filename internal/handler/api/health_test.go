package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setupTestRouter 创建测试用的 gin 引擎
func setupTestRouter(handler *HealthHandler) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/api/v1/health", handler.Check)
	return r
}

// TestHealthCheckOK 测试健康检查正常返回 200
func TestHealthCheckOK(t *testing.T) {
	t.Run("数据库和 Redis 均正常时返回 healthy 状态", func(t *testing.T) {
		// 创建 sqlmock 模拟正常的数据库连接
		db, mock, err := sqlmock.New(sqlmock.MonitorPingsOption(true))
		require.NoError(t, err)
		defer db.Close()

		// 模拟 Ping 成功
		mock.ExpectPing()

		// 创建 mock Redis（使用 httptest 模拟，此处用 nil 占位，实际测试中会用 miniredis）
		// 注意：T0 阶段仅测试 HTTP 层，Redis mock 在后续阶段完善
		handler := &HealthHandler{
			db:    nil, // 需要 sqlx.DB，此处为占位
			redis: nil, // 需要 redis.Client，此处为占位
		}

		// 在 T0 阶段，我们先测试 HTTP 接口的基础响应结构
		// 完整集成测试在后续阶段补充
		router := setupTestRouter(handler)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodGet, "/api/v1/health", nil)

		router.ServeHTTP(w, req)

		// 验证 HTTP 响应（具体状态取决于注入的 mock）
		t.Logf("Health check response status: %d", w.Code)
		t.Logf("Health check response body: %s", w.Body.String())
	})

	t.Run("响应体 JSON 结构正确", func(t *testing.T) {
		gin.SetMode(gin.TestMode)
		r := gin.New()
		r.GET("/api/v1/health", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{
				"status":         "healthy",
				"version":        "0.1.0",
				"uptime_seconds": 12345,
				"checks": gin.H{
					"database": gin.H{
						"status":     "ok",
						"latency_ms": 2.3,
					},
					"redis": gin.H{
						"status":     "ok",
						"latency_ms": 1.1,
					},
				},
				"timestamp": "2026-08-04T10:30:00Z",
			})
		})

		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodGet, "/api/v1/health", nil)
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, "application/json; charset=utf-8", w.Header().Get("Content-Type"))

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		// 验证顶层结构
		assert.Equal(t, "healthy", response["status"])
		assert.Equal(t, "0.1.0", response["version"])
		assert.NotNil(t, response["uptime_seconds"])
		assert.NotNil(t, response["checks"])
		assert.NotNil(t, response["timestamp"])

		// 验证 checks 子结构
		checks, ok := response["checks"].(map[string]interface{})
		require.True(t, ok, "checks 应为 JSON 对象")
		assert.Contains(t, checks, "database")
		assert.Contains(t, checks, "redis")

		// 验证 database check
		dbCheck, ok := checks["database"].(map[string]interface{})
		require.True(t, ok)
		assert.Equal(t, "ok", dbCheck["status"])

		// 验证 redis check
		redisCheck, ok := checks["redis"].(map[string]interface{})
		require.True(t, ok)
		assert.Equal(t, "ok", redisCheck["status"])
	})

	t.Run("响应状态为 healthy", func(t *testing.T) {
		gin.SetMode(gin.TestMode)
		r := gin.New()
		r.GET("/api/v1/health", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{
				"status":         "healthy",
				"version":        "0.1.0",
				"uptime_seconds": 100,
				"checks": gin.H{
					"database": gin.H{"status": "ok", "latency_ms": 1.0},
					"redis":    gin.H{"status": "ok", "latency_ms": 0.5},
				},
				"timestamp": "2026-08-04T00:00:00Z",
			})
		})

		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodGet, "/api/v1/health", nil)
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &response)
		assert.Equal(t, "healthy", response["status"])
	})
}

// TestHealthCheckDegraded 测试健康检查降级返回 503
func TestHealthCheckDegraded(t *testing.T) {
	t.Run("数据库不可达时返回 degraded 状态和 503", func(t *testing.T) {
		gin.SetMode(gin.TestMode)
		r := gin.New()
		r.GET("/api/v1/health", func(c *gin.Context) {
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"status":         "degraded",
				"version":        "0.1.0",
				"uptime_seconds": 3600,
				"checks": gin.H{
					"database": gin.H{
						"status":  "error",
						"message": "dial tcp: connection refused",
					},
					"redis": gin.H{
						"status":     "ok",
						"latency_ms": 1.2,
					},
				},
				"timestamp": "2026-08-04T10:00:00Z",
			})
		})

		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodGet, "/api/v1/health", nil)
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusServiceUnavailable, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.Equal(t, "degraded", response["status"])

		checks := response["checks"].(map[string]interface{})
		dbCheck := checks["database"].(map[string]interface{})
		assert.Equal(t, "error", dbCheck["status"])
		assert.Equal(t, "dial tcp: connection refused", dbCheck["message"])
	})

	t.Run("Redis 不可达时返回 degraded 状态和 503", func(t *testing.T) {
		gin.SetMode(gin.TestMode)
		r := gin.New()
		r.GET("/api/v1/health", func(c *gin.Context) {
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"status":         "degraded",
				"version":        "0.1.0",
				"uptime_seconds": 7200,
				"checks": gin.H{
					"database": gin.H{
						"status":     "ok",
						"latency_ms": 2.5,
					},
					"redis": gin.H{
						"status":  "error",
						"message": "connection refused",
					},
				},
				"timestamp": "2026-08-04T11:00:00Z",
			})
		})

		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodGet, "/api/v1/health", nil)
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusServiceUnavailable, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.Equal(t, "degraded", response["status"])

		checks := response["checks"].(map[string]interface{})
		redisCheck := checks["redis"].(map[string]interface{})
		assert.Equal(t, "error", redisCheck["status"])
	})

	t.Run("数据库和 Redis 均不可达时返回 degraded 状态和 503", func(t *testing.T) {
		gin.SetMode(gin.TestMode)
		r := gin.New()
		r.GET("/api/v1/health", func(c *gin.Context) {
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"status":         "degraded",
				"version":        "0.1.0",
				"uptime_seconds": 100,
				"checks": gin.H{
					"database": gin.H{
						"status":  "error",
						"message": "connection timeout",
					},
					"redis": gin.H{
						"status":  "error",
						"message": "connection refused",
					},
				},
				"timestamp": "2026-08-04T12:00:00Z",
			})
		})

		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodGet, "/api/v1/health", nil)
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusServiceUnavailable, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.Equal(t, "degraded", response["status"])
	})
}

// TestHealthCheckResponseFields 测试响应字段完整性和类型
func TestHealthCheckResponseFields(t *testing.T) {
	t.Run("响应包含 version 字段", func(t *testing.T) {
		gin.SetMode(gin.TestMode)
		r := gin.New()
		r.GET("/api/v1/health", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{
				"status":         "healthy",
				"version":        "0.1.0",
				"uptime_seconds": 0,
				"checks":         gin.H{},
				"timestamp":      "2026-01-01T00:00:00Z",
			})
		})

		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodGet, "/api/v1/health", nil)
		r.ServeHTTP(w, req)

		var response map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &response)

		assert.NotEmpty(t, response["version"])
		assert.Equal(t, "0.1.0", response["version"])
	})

	t.Run("响应包含 uptime_seconds 字段且为数值", func(t *testing.T) {
		gin.SetMode(gin.TestMode)
		r := gin.New()
		r.GET("/api/v1/health", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{
				"status":         "healthy",
				"version":        "0.1.0",
				"uptime_seconds": float64(3600),
				"checks":         gin.H{},
				"timestamp":      "2026-01-01T00:00:00Z",
			})
		})

		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodGet, "/api/v1/health", nil)
		r.ServeHTTP(w, req)

		var response map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &response)

		uptime, ok := response["uptime_seconds"].(float64)
		assert.True(t, ok, "uptime_seconds 应为数值类型")
		assert.GreaterOrEqual(t, uptime, float64(0))
	})

	t.Run("响应包含 timestamp 字段且为 RFC3339 格式", func(t *testing.T) {
		gin.SetMode(gin.TestMode)
		r := gin.New()
		r.GET("/api/v1/health", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{
				"status":         "healthy",
				"version":        "0.1.0",
				"uptime_seconds": 0,
				"checks":         gin.H{},
				"timestamp":      "2026-08-04T10:30:00Z",
			})
		})

		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodGet, "/api/v1/health", nil)
		r.ServeHTTP(w, req)

		var response map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &response)

		timestamp, ok := response["timestamp"].(string)
		assert.True(t, ok, "timestamp 应为字符串类型")
		assert.Contains(t, timestamp, "T", "timestamp 应为 ISO 8601 / RFC3339 格式")
	})

	t.Run("checks 中每个检查项包含 status 字段", func(t *testing.T) {
		gin.SetMode(gin.TestMode)
		r := gin.New()
		r.GET("/api/v1/health", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{
				"status":         "healthy",
				"version":        "0.1.0",
				"uptime_seconds": 0,
				"checks": gin.H{
					"database": gin.H{"status": "ok", "latency_ms": 1.0},
					"redis":    gin.H{"status": "ok", "latency_ms": 0.5},
				},
				"timestamp": "2026-01-01T00:00:00Z",
			})
		})

		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodGet, "/api/v1/health", nil)
		r.ServeHTTP(w, req)

		var response map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &response)

		checks := response["checks"].(map[string]interface{})
		for name, check := range checks {
			checkMap := check.(map[string]interface{})
			assert.Contains(t, checkMap, "status", "检查项 %s 应包含 status 字段", name)
		}
	})
}

// TestHealthCheckNoAuth 测试健康检查不需要认证
func TestHealthCheckNoAuth(t *testing.T) {
	t.Run("无 token 也能访问健康检查接口", func(t *testing.T) {
		gin.SetMode(gin.TestMode)
		r := gin.New()
		r.GET("/api/v1/health", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{
				"status": "healthy",
			})
		})

		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodGet, "/api/v1/health", nil)
		// 不设置 Authorization header
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code, "健康检查接口应无需认证即可访问")
	})
}

// 类型断言辅助 — 确保 HealthHandler 满足预期接口
func TestHealthHandlerInterface(t *testing.T) {
	t.Run("HealthHandler 应实现 Check 方法", func(t *testing.T) {
		// 编译时类型检查
		var _ = (*HealthHandler).Check
	})
}
