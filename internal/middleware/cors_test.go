package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"

	"github.com/weiyeston/weiyeston-v2/internal/config"
)

// TestCORSMiddleware_OptionsPreflight 测试 OPTIONS 预检请求
func TestCORSMiddleware_OptionsPreflight(t *testing.T) {
	gin.SetMode(gin.TestMode)

	corsConfig := config.CORSConfig{
		AllowedOrigins: []string{"http://localhost:5173", "http://localhost:8080"},
		AllowedMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders: []string{"Authorization", "Content-Type", "X-Requested-With"},
	}

	t.Run("OPTIONS 预检请求返回正确的 CORS 头", func(t *testing.T) {
		r := gin.New()
		r.Use(CORS(corsConfig))
		r.GET("/api/v1/test", func(c *gin.Context) {
			c.String(http.StatusOK, "ok")
		})

		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodOptions, "/api/v1/test", nil)
		req.Header.Set("Origin", "http://localhost:5173")
		req.Header.Set("Access-Control-Request-Method", "GET")
		r.ServeHTTP(w, req)

		// 验证响应状态
		assert.Equal(t, http.StatusNoContent, w.Code,
			"OPTIONS 预检请求应返回 204 No Content")

		// 验证 CORS 响应头
		assert.Equal(t, "http://localhost:5173", w.Header().Get("Access-Control-Allow-Origin"),
			"应返回允许的 Origin")
		assert.NotEmpty(t, w.Header().Get("Access-Control-Allow-Methods"),
			"应返回 Allow-Methods")
		assert.NotEmpty(t, w.Header().Get("Access-Control-Allow-Headers"),
			"应返回 Allow-Headers")
	})

	t.Run("OPTIONS 预检请求返回 Access-Control-Max-Age", func(t *testing.T) {
		r := gin.New()
		r.Use(CORS(corsConfig))
		r.POST("/api/v1/submit", func(c *gin.Context) {
			c.String(http.StatusOK, "ok")
		})

		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodOptions, "/api/v1/submit", nil)
		req.Header.Set("Origin", "http://localhost:5173")
		req.Header.Set("Access-Control-Request-Method", "POST")
		r.ServeHTTP(w, req)

		maxAge := w.Header().Get("Access-Control-Max-Age")
		assert.NotEmpty(t, maxAge, "应返回 Access-Control-Max-Age 头")
	})

	t.Run("不在允许列表中的 Origin 不返回 Allow-Origin 头", func(t *testing.T) {
		r := gin.New()
		r.Use(CORS(corsConfig))
		r.GET("/api/v1/test", func(c *gin.Context) {
			c.String(http.StatusOK, "ok")
		})

		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodOptions, "/api/v1/test", nil)
		req.Header.Set("Origin", "http://evil.example.com")
		req.Header.Set("Access-Control-Request-Method", "GET")
		r.ServeHTTP(w, req)

		// 不应返回 Allow-Origin 或者不应匹配恶意 Origin
		origin := w.Header().Get("Access-Control-Allow-Origin")
		assert.NotEqual(t, "http://evil.example.com", origin,
			"不在允许列表中的 Origin 不应被允许")
	})
}

// TestCORSMiddleware_ActualRequest 测试实际请求的 CORS 头
func TestCORSMiddleware_ActualRequest(t *testing.T) {
	gin.SetMode(gin.TestMode)

	corsConfig := config.CORSConfig{
		AllowedOrigins: []string{"http://localhost:5173", "http://localhost:8080"},
		AllowedMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders: []string{"Authorization", "Content-Type", "X-Requested-With"},
	}

	t.Run("GET 请求返回正确的 CORS 头", func(t *testing.T) {
		r := gin.New()
		r.Use(CORS(corsConfig))
		r.GET("/api/v1/test", func(c *gin.Context) {
			c.String(http.StatusOK, "ok")
		})

		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodGet, "/api/v1/test", nil)
		req.Header.Set("Origin", "http://localhost:5173")
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, "http://localhost:5173", w.Header().Get("Access-Control-Allow-Origin"),
			"响应应包含正确的 Allow-Origin")
	})

	t.Run("POST 请求返回正确的 CORS 头", func(t *testing.T) {
		r := gin.New()
		r.Use(CORS(corsConfig))
		r.POST("/api/v1/create", func(c *gin.Context) {
			c.String(http.StatusCreated, "created")
		})

		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodPost, "/api/v1/create", nil)
		req.Header.Set("Origin", "http://localhost:8080")
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusCreated, w.Code)
		assert.Equal(t, "http://localhost:8080", w.Header().Get("Access-Control-Allow-Origin"))
	})

	t.Run("PUT 请求返回正确的 CORS 头", func(t *testing.T) {
		r := gin.New()
		r.Use(CORS(corsConfig))
		r.PUT("/api/v1/update", func(c *gin.Context) {
			c.String(http.StatusOK, "updated")
		})

		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodPut, "/api/v1/update", nil)
		req.Header.Set("Origin", "http://localhost:5173")
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, "http://localhost:5173", w.Header().Get("Access-Control-Allow-Origin"))
	})

	t.Run("DELETE 请求返回正确的 CORS 头", func(t *testing.T) {
		r := gin.New()
		r.Use(CORS(corsConfig))
		r.DELETE("/api/v1/delete", func(c *gin.Context) {
			c.String(http.StatusOK, "deleted")
		})

		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodDelete, "/api/v1/delete", nil)
		req.Header.Set("Origin", "http://localhost:8080")
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, "http://localhost:8080", w.Header().Get("Access-Control-Allow-Origin"))
	})
}

// TestCORSMiddleware_WildcardOrigin 测试通配符 Origin 支持
func TestCORSMiddleware_WildcardOrigin(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("配置 * 通配符时允许任意 Origin", func(t *testing.T) {
		corsConfig := config.CORSConfig{
			AllowedOrigins: []string{"*"},
			AllowedMethods: []string{"GET", "POST"},
			AllowedHeaders: []string{"Content-Type"},
		}

		r := gin.New()
		r.Use(CORS(corsConfig))
		r.GET("/api/v1/test", func(c *gin.Context) {
			c.String(http.StatusOK, "ok")
		})

		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodGet, "/api/v1/test", nil)
		req.Header.Set("Origin", "https://any-domain.com")
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		allowOrigin := w.Header().Get("Access-Control-Allow-Origin")
		assert.NotEmpty(t, allowOrigin, "应返回 Allow-Origin 头")
	})
}

// TestCORSMiddleware_NoOriginHeader 测试无 Origin header 的请求
func TestCORSMiddleware_NoOriginHeader(t *testing.T) {
	gin.SetMode(gin.TestMode)

	corsConfig := config.CORSConfig{
		AllowedOrigins: []string{"http://localhost:5173"},
		AllowedMethods: []string{"GET", "POST"},
		AllowedHeaders: []string{"Content-Type"},
	}

	t.Run("无 Origin header 的请求正常通过", func(t *testing.T) {
		r := gin.New()
		r.Use(CORS(corsConfig))
		r.GET("/api/v1/test", func(c *gin.Context) {
			c.String(http.StatusOK, "ok")
		})

		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodGet, "/api/v1/test", nil)
		// 不设置 Origin header
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code,
			"无 Origin header 的请求应正常通过")
	})
}

// TestCORSMiddleware_AllowedHeaders 测试允许的请求头
func TestCORSMiddleware_AllowedHeaders(t *testing.T) {
	gin.SetMode(gin.TestMode)

	corsConfig := config.CORSConfig{
		AllowedOrigins: []string{"http://localhost:5173"},
		AllowedMethods: []string{"GET", "POST"},
		AllowedHeaders: []string{"Authorization", "Content-Type", "X-Requested-With"},
	}

	t.Run("OPTIONS 预检返回正确的 Allow-Headers", func(t *testing.T) {
		r := gin.New()
		r.Use(CORS(corsConfig))
		r.GET("/api/v1/test", func(c *gin.Context) {
			c.String(http.StatusOK, "ok")
		})

		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodOptions, "/api/v1/test", nil)
		req.Header.Set("Origin", "http://localhost:5173")
		req.Header.Set("Access-Control-Request-Method", "GET")
		req.Header.Set("Access-Control-Request-Headers", "Authorization, Content-Type")
		r.ServeHTTP(w, req)

		allowedHeaders := w.Header().Get("Access-Control-Allow-Headers")
		assert.Contains(t, allowedHeaders, "Authorization",
			"Allow-Headers 应包含 Authorization")
		assert.Contains(t, allowedHeaders, "Content-Type",
			"Allow-Headers 应包含 Content-Type")
	})
}

// TestCORSMiddleware_ExposeHeaders 测试暴露的响应头
func TestCORSMiddleware_ExposeHeaders(t *testing.T) {
	gin.SetMode(gin.TestMode)

	corsConfig := config.CORSConfig{
		AllowedOrigins: []string{"http://localhost:5173"},
		AllowedMethods: []string{"GET"},
		AllowedHeaders: []string{"Content-Type"},
	}

	t.Run("响应可能包含 Access-Control-Expose-Headers", func(t *testing.T) {
		r := gin.New()
		r.Use(CORS(corsConfig))
		r.GET("/api/v1/test", func(c *gin.Context) {
			c.Header("X-Custom-Header", "value")
			c.String(http.StatusOK, "ok")
		})

		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodGet, "/api/v1/test", nil)
		req.Header.Set("Origin", "http://localhost:5173")
		r.ServeHTTP(w, req)

		// 验证响应正常且 CORS 头已设置
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, "http://localhost:5173", w.Header().Get("Access-Control-Allow-Origin"))
	})
}

// TestCORSMiddleware_EmptyConfig 测试空 CORS 配置的兜底行为
func TestCORSMiddleware_EmptyConfig(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("空允许列表时不应设置 Allow-Origin", func(t *testing.T) {
		corsConfig := config.CORSConfig{
			AllowedOrigins: []string{},
			AllowedMethods: []string{},
			AllowedHeaders: []string{},
		}

		r := gin.New()
		r.Use(CORS(corsConfig))
		r.GET("/api/v1/test", func(c *gin.Context) {
			c.String(http.StatusOK, "ok")
		})

		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodGet, "/api/v1/test", nil)
		req.Header.Set("Origin", "http://localhost:5173")
		r.ServeHTTP(w, req)

		// 空配置不允许任何 Origin
		origin := w.Header().Get("Access-Control-Allow-Origin")
		assert.Empty(t, origin, "空允许列表不应设置 Allow-Origin")
	})
}
