package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

// TestTenantMiddleware_ContextInjection 测试租户上下文注入
func TestTenantMiddleware_ContextInjection(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("租户中间件从 token 中解析 tenant_id 并注入上下文", func(t *testing.T) {
		r := gin.New()

		// 模拟认证中间件已设置 user_id
		r.Use(func(c *gin.Context) {
			c.Set("user_id", int64(1))
			c.Next()
		})

		// 租户中间件
		r.Use(Tenant())

		r.GET("/api/v1/test", func(c *gin.Context) {
			tenantID, exists := c.Get("tenant_id")
			assert.True(t, exists, "上下文应包含 tenant_id")
			assert.NotNil(t, tenantID)
			c.JSON(http.StatusOK, gin.H{"tenant_id": tenantID})
		})

		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodGet, "/api/v1/test", nil)
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("无 user_id 时租户中间件不应阻塞请求", func(t *testing.T) {
		r := gin.New()
		r.Use(Tenant())
		r.GET("/api/v1/public", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"ok": true})
		})

		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodGet, "/api/v1/public", nil)
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code,
			"无 user_id 时租户中间件应允许请求通过")
	})
}

// TestTenantMiddleware_HeaderOverride 测试通过 header 覆盖租户
func TestTenantMiddleware_HeaderOverride(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("可通过 X-Tenant-ID header 指定租户", func(t *testing.T) {
		r := gin.New()
		r.Use(func(c *gin.Context) {
			c.Set("user_id", int64(1))
			c.Next()
		})
		r.Use(Tenant())
		r.GET("/api/v1/test", func(c *gin.Context) {
			tenantID, _ := c.Get("tenant_id")
			c.JSON(http.StatusOK, gin.H{"tenant_id": tenantID})
		})

		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodGet, "/api/v1/test", nil)
		req.Header.Set("X-Tenant-ID", "5")
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})
}
