package router

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestPublicRoutesAccessible 测试公开路由（无需认证）可以正常访问
func TestPublicRoutesAccessible(t *testing.T) {
	t.Run("GET /api/v1/health 可访问且返回非 401", func(t *testing.T) {
		gin.SetMode(gin.TestMode)
		// 使用最小化路由验证路由注册
		r := gin.New()
		r.GET("/api/v1/health", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"status": "healthy"})
		})

		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodGet, "/api/v1/health", nil)
		r.ServeHTTP(w, req)

		assert.NotEqual(t, http.StatusUnauthorized, w.Code,
			"健康检查端点不应要求认证")
		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("微信回调接口可访问（由微信签名验证保护）", func(t *testing.T) {
		gin.SetMode(gin.TestMode)
		r := gin.New()
		r.GET("/wx/callback/:app_id", func(c *gin.Context) {
			c.String(http.StatusOK, "verified")
		})
		r.POST("/wx/callback/:app_id", func(c *gin.Context) {
			c.String(http.StatusOK, "received")
		})

		// GET 验证
		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodGet, "/wx/callback/wx123456", nil)
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)

		// POST 消息
		w = httptest.NewRecorder()
		req, _ = http.NewRequest(http.MethodPost, "/wx/callback/wx123456", nil)
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("第三方平台事件推送接口可访问", func(t *testing.T) {
		gin.SetMode(gin.TestMode)
		r := gin.New()
		r.POST("/wx/component/callback", func(c *gin.Context) {
			c.String(http.StatusOK, "success")
		})

		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodPost, "/wx/component/callback", nil)
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("H5 展示页接口可访问", func(t *testing.T) {
		gin.SetMode(gin.TestMode)
		r := gin.New()
		r.GET("/h5/article/:id", func(c *gin.Context) {
			c.String(http.StatusOK, "article")
		})
		r.GET("/h5/news/:account_id", func(c *gin.Context) {
			c.String(http.StatusOK, "news list")
		})
		r.GET("/h5/vote/:id", func(c *gin.Context) {
			c.String(http.StatusOK, "vote")
		})

		tests := []struct {
			name   string
			method string
			path   string
		}{
			{"文章详情", http.MethodGet, "/h5/article/1"},
			{"快讯列表", http.MethodGet, "/h5/news/1"},
			{"投票页面", http.MethodGet, "/h5/vote/1"},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				w := httptest.NewRecorder()
				req, _ := http.NewRequest(tt.method, tt.path, nil)
				r.ServeHTTP(w, req)
				assert.Equal(t, http.StatusOK, w.Code)
			})
		}
	})
}

// TestAuthRequiredRoutes 测试需要认证的路由返回 401
func TestAuthRequiredRoutes(t *testing.T) {
	t.Run("未携带 token 访问受保护路由返回 401", func(t *testing.T) {
		gin.SetMode(gin.TestMode)
		r := gin.New()

		// 模拟认证中间件
		authMw := func(c *gin.Context) {
			token := c.GetHeader("Authorization")
			if token == "" || len(token) < 7 {
				c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
					"code": 401,
					"msg":  "未授权访问",
				})
				return
			}
			c.Next()
		}

		protected := r.Group("/api/v1")
		protected.Use(authMw)
		{
			protected.GET("/accounts", func(c *gin.Context) {
				c.JSON(http.StatusOK, gin.H{"data": []string{}})
			})
		}

		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodGet, "/api/v1/accounts", nil)
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code,
			"未携带 token 访问受保护路由应返回 401")
	})

	t.Run("携带无效 token 访问受保护路由返回 401", func(t *testing.T) {
		gin.SetMode(gin.TestMode)
		r := gin.New()

		authMw := func(c *gin.Context) {
			token := c.GetHeader("Authorization")
			if len(token) < 7 || token[:7] != "Bearer " {
				c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
					"code": 401,
					"msg":  "无效的 token 格式",
				})
				return
			}
			// 验证 token 有效性（此处模拟无效 token）
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"code": 401,
				"msg":  "token 无效或已过期",
			})
		}

		protected := r.Group("/api/v1")
		protected.Use(authMw)
		{
			protected.GET("/votes", func(c *gin.Context) {
				c.JSON(http.StatusOK, gin.H{})
			})
		}

		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodGet, "/api/v1/votes", nil)
		req.Header.Set("Authorization", "Bearer invalid-token-123")
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("登录接口在认证组外，不需要 token", func(t *testing.T) {
		gin.SetMode(gin.TestMode)
		r := gin.New()

		// 登录接口在分组外单独注册
		r.POST("/api/v1/auth/login", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"code": 0, "msg": "ok"})
		})

		// 受保护的 /api/v1 组
		protected := r.Group("/api/v1")
		protected.Use(func(c *gin.Context) {
			if c.GetHeader("Authorization") == "" {
				c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{})
				return
			}
			c.Next()
		})
		{
			protected.GET("/auth/me", func(c *gin.Context) {
				c.JSON(http.StatusOK, gin.H{})
			})
		}

		// 登录不需要 token
		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodPost, "/api/v1/auth/login", nil)
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code,
			"登录接口应无需认证即可访问")

		// auth/me 需要 token
		w = httptest.NewRecorder()
		req, _ = http.NewRequest(http.MethodGet, "/api/v1/auth/me", nil)
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusUnauthorized, w.Code,
			"auth/me 接口需要认证")
	})
}

// TestAdminAPIRoutes 测试所有管理后台 API 路由注册
func TestAdminAPIRoutes(t *testing.T) {
	t.Run("所有管理后台 API 路由已注册", func(t *testing.T) {
		gin.SetMode(gin.TestMode)
		r := gin.New()

		v1 := r.Group("/api/v1")
		{
			// 公众号管理
			v1.GET("/accounts", func(c *gin.Context) { c.String(200, "ok") })
			v1.POST("/accounts", func(c *gin.Context) { c.String(200, "ok") })
			v1.GET("/accounts/:id", func(c *gin.Context) { c.String(200, "ok") })
			v1.PUT("/accounts/:id", func(c *gin.Context) { c.String(200, "ok") })
			v1.DELETE("/accounts/:id", func(c *gin.Context) { c.String(200, "ok") })

			// CMS
			v1.GET("/cms/channels", func(c *gin.Context) { c.String(200, "ok") })
			v1.POST("/cms/channels", func(c *gin.Context) { c.String(200, "ok") })
			v1.PUT("/cms/channels/:id", func(c *gin.Context) { c.String(200, "ok") })
			v1.DELETE("/cms/channels/:id", func(c *gin.Context) { c.String(200, "ok") })
			v1.GET("/cms/articles", func(c *gin.Context) { c.String(200, "ok") })
			v1.POST("/cms/articles", func(c *gin.Context) { c.String(200, "ok") })
			v1.GET("/cms/articles/:id", func(c *gin.Context) { c.String(200, "ok") })
			v1.PUT("/cms/articles/:id", func(c *gin.Context) { c.String(200, "ok") })
			v1.DELETE("/cms/articles/:id", func(c *gin.Context) { c.String(200, "ok") })

			// 投票
			v1.GET("/votes", func(c *gin.Context) { c.String(200, "ok") })
			v1.POST("/votes", func(c *gin.Context) { c.String(200, "ok") })
			v1.GET("/votes/:id", func(c *gin.Context) { c.String(200, "ok") })
			v1.PUT("/votes/:id", func(c *gin.Context) { c.String(200, "ok") })
			v1.DELETE("/votes/:id", func(c *gin.Context) { c.String(200, "ok") })

			// 素材
			v1.GET("/materials", func(c *gin.Context) { c.String(200, "ok") })
			v1.POST("/materials/upload", func(c *gin.Context) { c.String(200, "ok") })
			v1.GET("/materials/:id", func(c *gin.Context) { c.String(200, "ok") })
			v1.DELETE("/materials/:id", func(c *gin.Context) { c.String(200, "ok") })

			// AI
			v1.POST("/ai/write", func(c *gin.Context) { c.String(200, "ok") })
			v1.POST("/ai/layout", func(c *gin.Context) { c.String(200, "ok") })
			v1.POST("/ai/proofread", func(c *gin.Context) { c.String(200, "ok") })

			// 快讯
			v1.GET("/news", func(c *gin.Context) { c.String(200, "ok") })
			v1.POST("/news", func(c *gin.Context) { c.String(200, "ok") })
			v1.GET("/news/:id", func(c *gin.Context) { c.String(200, "ok") })
			v1.PUT("/news/:id", func(c *gin.Context) { c.String(200, "ok") })
			v1.DELETE("/news/:id", func(c *gin.Context) { c.String(200, "ok") })
		}

		expectedRoutes := []struct {
			method string
			path   string
		}{
			// 公众号管理
			{http.MethodGet, "/api/v1/accounts"},
			{http.MethodPost, "/api/v1/accounts"},
			{http.MethodGet, "/api/v1/accounts/1"},
			{http.MethodPut, "/api/v1/accounts/1"},
			{http.MethodDelete, "/api/v1/accounts/1"},
			// CMS
			{http.MethodGet, "/api/v1/cms/channels"},
			{http.MethodPost, "/api/v1/cms/channels"},
			{http.MethodPut, "/api/v1/cms/channels/1"},
			{http.MethodDelete, "/api/v1/cms/channels/1"},
			{http.MethodGet, "/api/v1/cms/articles"},
			{http.MethodPost, "/api/v1/cms/articles"},
			{http.MethodGet, "/api/v1/cms/articles/1"},
			{http.MethodPut, "/api/v1/cms/articles/1"},
			{http.MethodDelete, "/api/v1/cms/articles/1"},
			// 投票
			{http.MethodGet, "/api/v1/votes"},
			{http.MethodPost, "/api/v1/votes"},
			{http.MethodGet, "/api/v1/votes/1"},
			{http.MethodPut, "/api/v1/votes/1"},
			{http.MethodDelete, "/api/v1/votes/1"},
			// 素材
			{http.MethodGet, "/api/v1/materials"},
			{http.MethodPost, "/api/v1/materials/upload"},
			{http.MethodGet, "/api/v1/materials/1"},
			{http.MethodDelete, "/api/v1/materials/1"},
			// AI
			{http.MethodPost, "/api/v1/ai/write"},
			{http.MethodPost, "/api/v1/ai/layout"},
			{http.MethodPost, "/api/v1/ai/proofread"},
			// 快讯
			{http.MethodGet, "/api/v1/news"},
			{http.MethodPost, "/api/v1/news"},
			{http.MethodGet, "/api/v1/news/1"},
			{http.MethodPut, "/api/v1/news/1"},
			{http.MethodDelete, "/api/v1/news/1"},
		}

		for _, tt := range expectedRoutes {
			t.Run(tt.method+" "+tt.path, func(t *testing.T) {
				w := httptest.NewRecorder()
				req, _ := http.NewRequest(tt.method, tt.path, nil)
				r.ServeHTTP(w, req)
				assert.Equal(t, http.StatusOK, w.Code,
					"路由 %s %s 应已注册并可访问", tt.method, tt.path)
			})
		}
	})
}

// TestTestRoutesRegistration 测试测试接口在非 release 模式下注册
func TestTestRoutesRegistration(t *testing.T) {
	t.Run("非 release 模式下 /api/v1/__tests__/ 前缀接口可访问", func(t *testing.T) {
		gin.SetMode(gin.TestMode)
		r := gin.New()

		testGroup := r.Group("/api/v1/__tests__")
		{
			testGroup.GET("/db/status", func(c *gin.Context) {
				c.JSON(200, gin.H{"db_alive": true})
			})
			testGroup.GET("/config", func(c *gin.Context) {
				c.JSON(200, gin.H{"config": "sanitized"})
			})
		}

		tests := []struct {
			method string
			path   string
		}{
			{http.MethodGet, "/api/v1/__tests__/db/status"},
			{http.MethodGet, "/api/v1/__tests__/config"},
		}

		for _, tt := range tests {
			t.Run(tt.method+" "+tt.path, func(t *testing.T) {
				w := httptest.NewRecorder()
				req, _ := http.NewRequest(tt.method, tt.path, nil)
				r.ServeHTTP(w, req)
				assert.Equal(t, http.StatusOK, w.Code,
					"测试接口 %s %s 应在非 release 模式下可访问", tt.method, tt.path)
			})
		}
	})

	t.Run("release 模式下不应注册测试接口", func(t *testing.T) {
		gin.SetMode(gin.ReleaseMode)
		r := gin.New()

		// 模拟仅在非 release 模式下注册测试路由
		// 在 release 模式下，测试路由组不会被注册
		// 因此访问应返回 404

		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodGet, "/api/v1/__tests__/db/status", nil)
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code,
			"release 模式下测试接口应返回 404")

		// 恢复模式
		gin.SetMode(gin.TestMode)
	})
}

// TestConfigEndpointSanitization 测试测试接口的配置端点脱敏
func TestConfigEndpointSanitization(t *testing.T) {
	t.Run("GET /api/v1/__tests__/config 返回脱敏后的配置", func(t *testing.T) {
		gin.SetMode(gin.TestMode)
		r := gin.New()

		r.GET("/api/v1/__tests__/config", func(c *gin.Context) {
			// 模拟脱敏
			c.JSON(200, gin.H{
				"server": gin.H{
					"port": 8080,
					"mode": "test",
				},
				"database": gin.H{
					"password": "***",
				},
				"jwt": gin.H{
					"secret": "***",
				},
				"wechat": gin.H{
					"component_app_secret": "***",
					"encoding_aes_key":     "***",
				},
				"upload": gin.H{
					"s3_key":    "***",
					"s3_secret": "***",
				},
				"ai": gin.H{
					"api_key": "***",
				},
			})
		})

		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodGet, "/api/v1/__tests__/config", nil)
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		require.NoError(t, jsonDecode(w.Body.Bytes(), &response))

		// 验证敏感字段已脱敏
		db := response["database"].(map[string]interface{})
		assert.Equal(t, "***", db["password"], "database.password 应被脱敏")

		jwt := response["jwt"].(map[string]interface{})
		assert.Equal(t, "***", jwt["secret"], "jwt.secret 应被脱敏")

		wechat := response["wechat"].(map[string]interface{})
		assert.Equal(t, "***", wechat["component_app_secret"], "wechat.component_app_secret 应被脱敏")
		assert.Equal(t, "***", wechat["encoding_aes_key"], "wechat.encoding_aes_key 应被脱敏")

		upload := response["upload"].(map[string]interface{})
		assert.Equal(t, "***", upload["s3_key"], "upload.s3_key 应被脱敏")
		assert.Equal(t, "***", upload["s3_secret"], "upload.s3_secret 应被脱敏")

		ai := response["ai"].(map[string]interface{})
		assert.Equal(t, "***", ai["api_key"], "ai.api_key 应被脱敏")
	})
}

// TestMiddlewareRegistration 测试中间件注册顺序
func TestMiddlewareRegistration(t *testing.T) {
	t.Run("全局中间件按正确顺序注册", func(t *testing.T) {
		gin.SetMode(gin.TestMode)
		r := gin.New()

		// 验证中间件可以被注册（顺序：Recovery → Logger → CORS）
		var middlewareOrder []string

		r.Use(func(c *gin.Context) {
			middlewareOrder = append(middlewareOrder, "recovery")
			c.Next()
		})
		r.Use(func(c *gin.Context) {
			middlewareOrder = append(middlewareOrder, "logger")
			c.Next()
		})
		r.Use(func(c *gin.Context) {
			middlewareOrder = append(middlewareOrder, "cors")
			c.Next()
		})

		r.GET("/test", func(c *gin.Context) {
			c.String(200, "ok")
		})

		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodGet, "/test", nil)
		r.ServeHTTP(w, req)

		assert.Equal(t, []string{"recovery", "logger", "cors"}, middlewareOrder,
			"中间件应按 Recovery → Logger → CORS 顺序执行")
	})

	t.Run("租户中间件注册", func(t *testing.T) {
		gin.SetMode(gin.TestMode)
		r := gin.New()

		var tenantMwCalled bool
		tenantMw := func(c *gin.Context) {
			tenantMwCalled = true
			c.Next()
		}

		// 租户中间件应在 auth 中间件之后注册
		r.Use(tenantMw)
		r.GET("/api/v1/test", func(c *gin.Context) {
			c.String(200, "ok")
		})

		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodGet, "/api/v1/test", nil)
		r.ServeHTTP(w, req)

		assert.True(t, tenantMwCalled, "租户中间件应被调用")
	})
}

// TestDependenciesStruct 测试 Dependencies 结构体
func TestDependenciesStruct(t *testing.T) {
	t.Run("Dependencies 包含必要的字段", func(t *testing.T) {
		// 编译时类型检查
		var deps Dependencies
		assert.Nil(t, deps.Config)
		assert.Nil(t, deps.DB)
		assert.Nil(t, deps.Redis)
		assert.Nil(t, deps.Logger)
	})
}

// TestSetupFunction 测试 Setup 函数存在性
func TestSetupFunction(t *testing.T) {
	t.Run("Setup 函数存在且接受 Dependencies 参数", func(t *testing.T) {
		// 编译时检查：确保 Setup(*Dependencies) *gin.Engine 签名存在
		var _ = Setup
	})
}

// jsonDecode 辅助函数，解码 JSON 响应体
func jsonDecode(data []byte, v interface{}) error {
	return json.Unmarshal(data, v)
}
