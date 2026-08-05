package middleware

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/weiyeston/weiyeston-v2/internal/config"
)

// generateTestToken 生成测试用的 JWT token
func generateTestToken(secret string, expiration time.Duration, issuer string, userID int64) (string, error) {
	claims := jwt.MapClaims{
		"sub": userID,
		"iat": time.Now().Unix(),
		"exp": time.Now().Add(expiration).Unix(),
		"iss": issuer,
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}

// TestAuthMiddleware_ValidToken 测试有效 token
func TestAuthMiddleware_ValidToken(t *testing.T) {
	gin.SetMode(gin.TestMode)

	secret := "test-secret-key"
	jwtConfig := config.JWTConfig{
		Secret:     secret,
		Expiration: 24 * time.Hour,
		Issuer:     "weiyeston-v2",
	}

	t.Run("有效 token 应通过认证并设置用户上下文", func(t *testing.T) {
		token, err := generateTestToken(secret, 24*time.Hour, "weiyeston-v2", 1)
		require.NoError(t, err)

		r := gin.New()
		r.Use(Auth(jwtConfig))
		r.GET("/protected", func(c *gin.Context) {
			// 验证上下文中已设置用户信息
			userID, exists := c.Get("user_id")
			assert.True(t, exists, "上下文应包含 user_id")
			assert.Equal(t, int64(1), userID)

			c.JSON(http.StatusOK, gin.H{"ok": true})
		})

		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodGet, "/protected", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("有效 token 后 c.Next() 被调用", func(t *testing.T) {
		token, err := generateTestToken(secret, 24*time.Hour, "weiyeston-v2", 42)
		require.NoError(t, err)

		r := gin.New()
		r.Use(Auth(jwtConfig))
		r.GET("/next-test", func(c *gin.Context) {
			c.String(http.StatusOK, "passed")
		})

		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodGet, "/next-test", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, "passed", w.Body.String())
	})
}

// TestAuthMiddleware_ExpiredToken 测试过期 token
func TestAuthMiddleware_ExpiredToken(t *testing.T) {
	gin.SetMode(gin.TestMode)

	secret := "test-secret-key"
	jwtConfig := config.JWTConfig{
		Secret:     secret,
		Expiration: 24 * time.Hour,
		Issuer:     "weiyeston-v2",
	}

	t.Run("过期 token 应返回 401", func(t *testing.T) {
		// 生成一个已过期的 token（过期时间为过去 1 小时）
		claims := jwt.MapClaims{
			"sub": 1,
			"iat": time.Now().Add(-2 * time.Hour).Unix(),
			"exp": time.Now().Add(-1 * time.Hour).Unix(),
			"iss": "weiyeston-v2",
		}
		expiredToken := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
		expiredTokenStr, err := expiredToken.SignedString([]byte(secret))
		require.NoError(t, err)

		r := gin.New()
		r.Use(Auth(jwtConfig))
		r.GET("/protected", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"ok": true})
		})

		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodGet, "/protected", nil)
		req.Header.Set("Authorization", "Bearer "+expiredTokenStr)
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code,
			"过期 token 应返回 401")

		var response map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &response)
		assert.Contains(t, response["msg"], "过期", "错误信息应包含'过期'")
	})

	t.Run("即将过期的 token 仍应通过认证", func(t *testing.T) {
		// token 将在 5 秒后过期（避免测试竞态）
		token, err := generateTestToken(secret, 5*time.Second, "weiyeston-v2", 1)
		require.NoError(t, err)

		r := gin.New()
		r.Use(Auth(jwtConfig))
		r.GET("/protected", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"ok": true})
		})

		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodGet, "/protected", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code,
			"即将过期(5s)但尚未过期的 token 应通过认证")
	})
}

// TestAuthMiddleware_InvalidToken 测试无效 token
func TestAuthMiddleware_InvalidToken(t *testing.T) {
	gin.SetMode(gin.TestMode)

	secret := "test-secret-key"
	jwtConfig := config.JWTConfig{
		Secret:     secret,
		Expiration: 24 * time.Hour,
		Issuer:     "weiyeston-v2",
	}

	t.Run("错误的签名应返回 401", func(t *testing.T) {
		// 使用不同的密钥签名
		wrongSecretToken, err := generateTestToken("wrong-secret", 24*time.Hour, "weiyeston-v2", 1)
		require.NoError(t, err)

		r := gin.New()
		r.Use(Auth(jwtConfig))
		r.GET("/protected", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"ok": true})
		})

		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodGet, "/protected", nil)
		req.Header.Set("Authorization", "Bearer "+wrongSecretToken)
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code,
			"错误签名的 token 应返回 401")
	})

	t.Run("格式错误的 token 应返回 401", func(t *testing.T) {
		r := gin.New()
		r.Use(Auth(jwtConfig))
		r.GET("/protected", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"ok": true})
		})

		malformedTokens := []string{
			"not-a-valid-jwt-token",
			"eyJhbGciOiJIUzI1NiJ9.xxx.yyy",
			"a.b.c.d.e",
			"",
		}

		for _, mt := range malformedTokens {
			t.Run("token="+truncateToken(mt), func(t *testing.T) {
				w := httptest.NewRecorder()
				req, _ := http.NewRequest(http.MethodGet, "/protected", nil)
				if mt != "" {
					req.Header.Set("Authorization", "Bearer "+mt)
				}
				r.ServeHTTP(w, req)

				assert.Equal(t, http.StatusUnauthorized, w.Code,
					"格式错误的 token 应返回 401")
			})
		}
	})

	t.Run("不包含 Bearer 前缀应返回 401", func(t *testing.T) {
		validToken, err := generateTestToken(secret, 24*time.Hour, "weiyeston-v2", 1)
		require.NoError(t, err)

		r := gin.New()
		r.Use(Auth(jwtConfig))
		r.GET("/protected", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"ok": true})
		})

		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodGet, "/protected", nil)
		req.Header.Set("Authorization", validToken) // 没有 Bearer 前缀
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code,
			"缺少 Bearer 前缀应返回 401")
	})

	t.Run("使用不同的签名算法应返回 401", func(t *testing.T) {
		// 使用 HS384 而不是 HS256
		claims := jwt.MapClaims{
			"sub": 1,
			"exp": time.Now().Add(24 * time.Hour).Unix(),
			"iss": "weiyeston-v2",
		}
		token := jwt.NewWithClaims(jwt.SigningMethodHS384, claims)
		tokenStr, err := token.SignedString([]byte(secret))
		require.NoError(t, err)

		r := gin.New()
		r.Use(Auth(jwtConfig))
		r.GET("/protected", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"ok": true})
		})

		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodGet, "/protected", nil)
		req.Header.Set("Authorization", "Bearer "+tokenStr)
		r.ServeHTTP(w, req)

		// 不同签名算法应被拒绝（安全考虑）
		assert.Equal(t, http.StatusUnauthorized, w.Code,
			"不同签名算法的 token 应返回 401")
	})

	t.Run("缺少 sub 字段的 token 应返回 401", func(t *testing.T) {
		claims := jwt.MapClaims{
			"exp": time.Now().Add(24 * time.Hour).Unix(),
			"iss": "weiyeston-v2",
			// 缺少 sub 字段
		}
		token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
		tokenStr, err := token.SignedString([]byte(secret))
		require.NoError(t, err)

		r := gin.New()
		r.Use(Auth(jwtConfig))
		r.GET("/protected", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"ok": true})
		})

		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodGet, "/protected", nil)
		req.Header.Set("Authorization", "Bearer "+tokenStr)
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code,
			"缺少 sub 的 token 应返回 401")
	})
}

// TestAuthMiddleware_NoToken 测试无 token
func TestAuthMiddleware_NoToken(t *testing.T) {
	gin.SetMode(gin.TestMode)

	jwtConfig := config.JWTConfig{
		Secret:     "test-secret-key",
		Expiration: 24 * time.Hour,
		Issuer:     "weiyeston-v2",
	}

	t.Run("无 Authorization header 应返回 401", func(t *testing.T) {
		r := gin.New()
		r.Use(Auth(jwtConfig))
		r.GET("/protected", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"ok": true})
		})

		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodGet, "/protected", nil)
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code,
			"无 Authorization header 应返回 401")
	})

	t.Run("Authorization header 为空字符串应返回 401", func(t *testing.T) {
		r := gin.New()
		r.Use(Auth(jwtConfig))
		r.GET("/protected", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"ok": true})
		})

		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodGet, "/protected", nil)
		req.Header.Set("Authorization", "")
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code,
			"Authorization header 为空应返回 401")
	})

	t.Run("Authorization header 仅有 Bearer 前缀无 token 应返回 401", func(t *testing.T) {
		r := gin.New()
		r.Use(Auth(jwtConfig))
		r.GET("/protected", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"ok": true})
		})

		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodGet, "/protected", nil)
		req.Header.Set("Authorization", "Bearer ")
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})
}

// TestAuthMiddleware_SuccessBehavior 测试认证成功后的行为
func TestAuthMiddleware_SuccessBehavior(t *testing.T) {
	gin.SetMode(gin.TestMode)

	secret := "test-secret-key"
	jwtConfig := config.JWTConfig{
		Secret:     secret,
		Expiration: 24 * time.Hour,
		Issuer:     "weiyeston-v2",
	}

	t.Run("认证成功后将 user_id 注入上下文", func(t *testing.T) {
		token, err := generateTestToken(secret, 24*time.Hour, "weiyeston-v2", 100)
		require.NoError(t, err)

		var capturedUserID interface{}

		r := gin.New()
		r.Use(Auth(jwtConfig))
		r.GET("/check-user", func(c *gin.Context) {
			capturedUserID, _ = c.Get("user_id")
			c.JSON(http.StatusOK, gin.H{"ok": true})
		})

		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodGet, "/check-user", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, int64(100), capturedUserID,
			"认证成功后 user_id 应为 token 中的 sub")
	})

	t.Run("认证成功后将 tenant_id 注入上下文（容错：token 无 tenant 时设为 0）", func(t *testing.T) {
		token, err := generateTestToken(secret, 24*time.Hour, "weiyeston-v2", 200)
		require.NoError(t, err)

		r := gin.New()
		r.Use(Auth(jwtConfig))
		r.GET("/check-tenant", func(c *gin.Context) {
			tenantID, exists := c.Get("tenant_id")
			if exists {
				c.JSON(http.StatusOK, gin.H{"tenant_id": tenantID})
			} else {
				c.JSON(http.StatusOK, gin.H{"tenant_id": 0})
			}
		})

		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodGet, "/check-tenant", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})
}

// TestAuthMiddleware_ResponseFormat 测试 401 响应格式
func TestAuthMiddleware_ResponseFormat(t *testing.T) {
	gin.SetMode(gin.TestMode)

	jwtConfig := config.JWTConfig{
		Secret:     "test-secret-key",
		Expiration: 24 * time.Hour,
		Issuer:     "weiyeston-v2",
	}

	t.Run("401 响应应为 JSON 格式", func(t *testing.T) {
		r := gin.New()
		r.Use(Auth(jwtConfig))
		r.GET("/protected", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"ok": true})
		})

		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodGet, "/protected", nil)
		r.ServeHTTP(w, req)

		assert.Equal(t, "application/json; charset=utf-8", w.Header().Get("Content-Type"),
			"401 响应 Content-Type 应为 JSON")

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err, "响应体应为有效 JSON")

		assert.Contains(t, response, "code", "响应应包含 code 字段")
		assert.Contains(t, response, "msg", "响应应包含 msg 字段")
	})
}

// truncateToken 截断 token 用于测试名称
func truncateToken(t string) string {
	if len(t) > 20 {
		return t[:20] + "..."
	}
	if t == "" {
		return "empty"
	}
	return t
}
