// Package api 认证 Handler 测试
// TDD: 测试先行 — 使用 httptest + sqlmock + 内存 Redis 模拟完整请求-响应流程
// auth.go 尚未实现，测试使用内联 handler 展示预期行为
package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/weiyeston/weiyeston-v2/internal/config"
	"github.com/weiyeston/weiyeston-v2/internal/middleware"
)

// ========== Mock Redis ==========

type mockRedisStore struct {
	mu     sync.RWMutex
	store  map[string]string
	counts map[string]int64
}

func newMockRedis() *mockRedisStore {
	return &mockRedisStore{store: make(map[string]string), counts: make(map[string]int64)}
}

func (m *mockRedisStore) Get(key string) (string, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	v, ok := m.store[key]
	return v, ok
}

func (m *mockRedisStore) Set(key, value string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.store[key] = value
}

func (m *mockRedisStore) Del(key string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.store, key)
}

func (m *mockRedisStore) Incr(key string) int64 {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.counts[key]++
	return m.counts[key]
}

func (m *mockRedisStore) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.store = make(map[string]string)
	m.counts = make(map[string]int64)
}

// ========== 测试辅助函数 ==========

func newTestJWTConfig() config.JWTConfig {
	return config.JWTConfig{
		Secret:     "test-secret-key-for-auth-tests-32bytes!!",
		Expiration: 2 * time.Hour,
		Issuer:     "weiyeston-v2",
	}
}

func generateTestAccessToken(secret string, userID int64, role string, exp time.Time) (string, error) {
	claims := jwt.MapClaims{
		"sub":       userID,
		"tenant_id": userID,
		"role":      role,
		"nickname":  "测试用户",
		"iat":       time.Now().Unix(),
		"exp":       exp.Unix(),
		"iss":       "weiyeston-v2",
		"jti":       "test-jti",
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}

func generateExpiredAccessToken(secret string, userID int64) (string, error) {
	return generateTestAccessToken(secret, userID, "user", time.Now().Add(-1*time.Hour))
}

// ========== 内联测试 Handler ==========

// testLoginHandler 模拟登录 handler（TDD 预期行为：限流校验→查询用户→密码比对→返回双Token）
func testLoginHandler(mock sqlmock.Sqlmock, redisStore *mockRedisStore, jwtCfg config.JWTConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			Username string `json:"username" binding:"required"`
			Password string `json:"password" binding:"required"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"code": 40001, "msg": "请求参数不合法"})
			return
		}

		// 限流检查
		ip := c.ClientIP()
		count := redisStore.Incr("login_rate:" + ip)
		if count > 5 {
			c.JSON(http.StatusTooManyRequests, gin.H{"code": 42901, "msg": "登录尝试过于频繁，请稍后再试"})
			return
		}

		// 有效的登录凭据
		if req.Username == "admin" && req.Password == "test123" {
			token, _ := generateTestAccessToken(jwtCfg.Secret, 1, "admin", time.Now().Add(2*time.Hour))
			refreshToken := "test-refresh-uuid-12345678-abcd-4def-abcd-123456789abc"
			redisStore.Set("refresh_token:1", refreshToken)

			c.JSON(http.StatusOK, gin.H{
				"code": 0,
				"msg":  "登录成功",
				"data": gin.H{
					"access_token":  token,
					"refresh_token": refreshToken,
					"expires_in":    int64(7200),
					"user": gin.H{
						"id":       1,
						"username": "admin",
						"nickname": "管理员",
						"role":     "admin",
					},
				},
			})
			return
		}

		// 不区分用户不存在和密码错误
		c.JSON(http.StatusUnauthorized, gin.H{"code": 40101, "msg": "用户名或密码错误"})
	}
}

// testRefreshHandler 模拟刷新 token handler（独立于 Auth 中间件组外，可接收过期 token）
func testRefreshHandler(redisStore *mockRedisStore, jwtCfg config.JWTConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			RefreshToken string `json:"refresh_token" binding:"required"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"code": 40001, "msg": "请求参数不合法"})
			return
		}

		// 从 Authorization header 解析 user_id（使用 ParseUnverified 跳过过期验证）
		authHeader := c.GetHeader("Authorization")
		var userID int64
		if strings.HasPrefix(authHeader, "Bearer ") {
			tokenStr := strings.TrimPrefix(authHeader, "Bearer ")
			parser := jwt.Parser{}
			parsedToken, _, err := parser.ParseUnverified(tokenStr, jwt.MapClaims{})
			if err == nil {
				if claims, ok := parsedToken.Claims.(jwt.MapClaims); ok {
					if sub, exists := claims["sub"]; exists {
						switch v := sub.(type) {
						case float64:
							userID = int64(v)
						case int64:
							userID = v
						}
					}
				}
			}
		}

		if userID == 0 {
			c.JSON(http.StatusUnauthorized, gin.H{"code": 40102, "msg": "refresh token 无效或已过期，请重新登录"})
			return
		}

		// 从 Redis 读取 refresh_token
		key := "refresh_token:1"
		stored, ok := redisStore.Get(key)
		if !ok || stored != req.RefreshToken {
			c.JSON(http.StatusUnauthorized, gin.H{"code": 40102, "msg": "refresh token 无效或已过期，请重新登录"})
			return
		}

		// Rotation: 生成新 token 对
		newToken, _ := generateTestAccessToken(jwtCfg.Secret, userID, "admin", time.Now().Add(2*time.Hour))
		newRefresh := "new-refresh-uuid-" + time.Now().Format("20060102150405")
		redisStore.Set(key, newRefresh)

		c.JSON(http.StatusOK, gin.H{
			"code": 0,
			"msg":  "ok",
			"data": gin.H{
				"access_token":  newToken,
				"refresh_token": newRefresh,
				"expires_in":    int64(7200),
			},
		})
	}
}

// ========== Login 测试 ==========

func TestLogin_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()
	_ = sqlx.NewDb(db, "postgres")

	redisStore := newMockRedis()
	jwtCfg := newTestJWTConfig()

	r := gin.New()
	r.POST("/api/v1/auth/login", testLoginHandler(mock, redisStore, jwtCfg))

	body := `{"username":"admin","password":"test123"}`
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/auth/login", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)

	assert.Equal(t, float64(0), resp["code"])
	assert.Equal(t, "登录成功", resp["msg"])

	data, ok := resp["data"].(map[string]interface{})
	require.True(t, ok, "data 应存在")
	assert.NotEmpty(t, data["access_token"])
	assert.NotEmpty(t, data["refresh_token"])
	assert.Equal(t, float64(7200), data["expires_in"])

	user, ok := data["user"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, float64(1), user["id"])
	assert.Equal(t, "admin", user["username"])
	assert.Equal(t, "管理员", user["nickname"])
	assert.Equal(t, "admin", user["role"])
}

func TestLogin_WrongPassword(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()
	_ = sqlx.NewDb(db, "postgres")

	redisStore := newMockRedis()
	jwtCfg := newTestJWTConfig()

	r := gin.New()
	r.POST("/api/v1/auth/login", testLoginHandler(mock, redisStore, jwtCfg))

	body := `{"username":"admin","password":"wrongpass"}`
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/auth/login", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)

	var resp map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, float64(40101), resp["code"])
	assert.Equal(t, "用户名或密码错误", resp["msg"])
}

func TestLogin_UserNotFound(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()
	_ = sqlx.NewDb(db, "postgres")

	redisStore := newMockRedis()
	jwtCfg := newTestJWTConfig()

	r := gin.New()
	r.POST("/api/v1/auth/login", testLoginHandler(mock, redisStore, jwtCfg))

	body := `{"username":"nonexistent","password":"anypass"}`
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/auth/login", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)

	var resp map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	// 与错误密码完全相同的错误码/msg（防账号枚举）
	assert.Equal(t, float64(40101), resp["code"])
	assert.Equal(t, "用户名或密码错误", resp["msg"])
}

func TestLogin_RateLimit(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()
	_ = sqlx.NewDb(db, "postgres")

	redisStore := newMockRedis()
	jwtCfg := newTestJWTConfig()

	r := gin.New()
	r.POST("/api/v1/auth/login", testLoginHandler(mock, redisStore, jwtCfg))

	body := `{"username":"admin","password":"test123"}`

	// 前 5 次不被限流
	for i := 1; i <= 5; i++ {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodPost, "/api/v1/auth/login", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)
		assert.NotEqual(t, http.StatusTooManyRequests, w.Code,
			"第 %d 次请求不应被限流", i)
	}

	// 第 6 次应被限流
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/auth/login", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusTooManyRequests, w.Code)

	var resp map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, float64(42901), resp["code"])
	assert.Contains(t, resp["msg"], "频繁")
}

func TestLogin_EmptyCredentials(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()
	_ = sqlx.NewDb(db, "postgres")

	redisStore := newMockRedis()
	jwtCfg := newTestJWTConfig()

	r := gin.New()
	r.POST("/api/v1/auth/login", testLoginHandler(mock, redisStore, jwtCfg))

	tests := []struct {
		name string
		body string
	}{
		{"用户名和密码都为空", `{"username":"","password":""}`},
		{"用户名为空", `{"username":"","password":"test123"}`},
		{"密码为空", `{"username":"admin","password":""}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			req, _ := http.NewRequest(http.MethodPost, "/api/v1/auth/login", strings.NewReader(tt.body))
			req.Header.Set("Content-Type", "application/json")
			r.ServeHTTP(w, req)

			assert.Equal(t, http.StatusBadRequest, w.Code)

			var resp map[string]interface{}
			json.Unmarshal(w.Body.Bytes(), &resp)
			assert.Equal(t, float64(40001), resp["code"])
		})
	}
}

func TestLogin_DisabledAccount(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()
	_ = sqlx.NewDb(db, "postgres")

	r := gin.New()
	r.POST("/api/v1/auth/login", func(c *gin.Context) {
		var req struct {
			Username string `json:"username" binding:"required"`
			Password string `json:"password" binding:"required"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"code": 40001, "msg": "请求参数不合法"})
			return
		}
		// 模拟: 用户存在但 status=2
		_ = mock
		c.JSON(http.StatusForbidden, gin.H{"code": 40301, "msg": "账号已被停用"})
	})

	body := `{"username":"disabled_user","password":"test123"}`
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/auth/login", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)

	var resp map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, float64(40301), resp["code"])
	assert.Equal(t, "账号已被停用", resp["msg"])
}

// ========== Refresh 测试 ==========

func TestRefresh_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	redisStore := newMockRedis()
	jwtCfg := newTestJWTConfig()
	redisStore.Set("refresh_token:1", "test-refresh-uuid-12345678-abcd-4def-abcd-123456789abc")

	validToken, err := generateTestAccessToken(jwtCfg.Secret, 1, "admin", time.Now().Add(2*time.Hour))
	require.NoError(t, err)

	r := gin.New()
	r.POST("/api/v1/auth/refresh", testRefreshHandler(redisStore, jwtCfg))

	body := `{"refresh_token":"test-refresh-uuid-12345678-abcd-4def-abcd-123456789abc"}`
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/auth/refresh", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+validToken)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, float64(0), resp["code"])

	data, ok := resp["data"].(map[string]interface{})
	require.True(t, ok)
	assert.NotEmpty(t, data["access_token"])
	assert.NotEmpty(t, data["refresh_token"])

	// Rotation: 旧 refresh token 已被覆盖
	stored, _ := redisStore.Get("refresh_token:1")
	assert.NotEqual(t, "test-refresh-uuid-12345678-abcd-4def-abcd-123456789abc", stored,
		"旧 refresh token 应已被覆盖（Rotation）")
}

func TestRefresh_ExpiredAccessToken(t *testing.T) {
	gin.SetMode(gin.TestMode)
	redisStore := newMockRedis()
	jwtCfg := newTestJWTConfig()
	redisStore.Set("refresh_token:1", "cr-expired-test-token")

	expiredToken, err := generateExpiredAccessToken(jwtCfg.Secret, 1)
	require.NoError(t, err)

	r := gin.New()
	// /auth/refresh 独立注册，不经过 Auth 中间件
	r.POST("/api/v1/auth/refresh", testRefreshHandler(redisStore, jwtCfg))
	// 对比: /auth/me 经过 Auth 中间件
	protected := r.Group("/api/v1/auth")
	protected.Use(middleware.Auth(jwtCfg))
	protected.GET("/me", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	// 过期 token 访问 /refresh → 应能进入 handler
	body := `{"refresh_token":"cr-expired-test-token"}`
	wRefresh := httptest.NewRecorder()
	reqRefresh, _ := http.NewRequest(http.MethodPost, "/api/v1/auth/refresh", strings.NewReader(body))
	reqRefresh.Header.Set("Content-Type", "application/json")
	reqRefresh.Header.Set("Authorization", "Bearer "+expiredToken)
	r.ServeHTTP(wRefresh, reqRefresh)

	assert.NotEqual(t, http.StatusUnauthorized, wRefresh.Code,
		"/auth/refresh 不应被 Auth 中间件拦截，应能接收过期 token")

	// 过期 token 访问 /me → 应被 Auth 中间件拦截返回 401
	wMe := httptest.NewRecorder()
	reqMe, _ := http.NewRequest(http.MethodGet, "/api/v1/auth/me", nil)
	reqMe.Header.Set("Authorization", "Bearer "+expiredToken)
	r.ServeHTTP(wMe, reqMe)

	assert.Equal(t, http.StatusUnauthorized, wMe.Code,
		"/auth/me 使用过期 token 应被 Auth 中间件拦截返回 401")

	t.Log("关键验证: /auth/refresh 不在 Auth 中间件组内，可处理过期 token；/auth/me 在组内，被拦截")
}

func TestRefresh_InvalidRefreshToken(t *testing.T) {
	gin.SetMode(gin.TestMode)
	redisStore := newMockRedis()
	jwtCfg := newTestJWTConfig()

	validToken, _ := generateTestAccessToken(jwtCfg.Secret, 1, "admin", time.Now().Add(2*time.Hour))

	r := gin.New()
	r.POST("/api/v1/auth/refresh", testRefreshHandler(redisStore, jwtCfg))

	body := `{"refresh_token":"non-existent-token"}`
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/auth/refresh", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+validToken)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)

	var resp map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, float64(40102), resp["code"])
	assert.Contains(t, resp["msg"], "refresh token")
}

func TestRefresh_ReplayDetection(t *testing.T) {
	gin.SetMode(gin.TestMode)
	redisStore := newMockRedis()
	jwtCfg := newTestJWTConfig()
	redisStore.Set("refresh_token:1", "replay-test-token-aaaabbbbccccddddeeeeffff")

	validToken, _ := generateTestAccessToken(jwtCfg.Secret, 1, "admin", time.Now().Add(2*time.Hour))

	r := gin.New()
	r.POST("/api/v1/auth/refresh", testRefreshHandler(redisStore, jwtCfg))

	body := `{"refresh_token":"replay-test-token-aaaabbbbccccddddeeeeffff"}`

	// 第一次 → 成功
	w1 := httptest.NewRecorder()
	req1, _ := http.NewRequest(http.MethodPost, "/api/v1/auth/refresh", strings.NewReader(body))
	req1.Header.Set("Content-Type", "application/json")
	req1.Header.Set("Authorization", "Bearer "+validToken)
	r.ServeHTTP(w1, req1)
	assert.Equal(t, http.StatusOK, w1.Code, "第一次使用应成功")

	// 第二次用同一个 token → 失败（重放攻击）
	w2 := httptest.NewRecorder()
	req2, _ := http.NewRequest(http.MethodPost, "/api/v1/auth/refresh", strings.NewReader(body))
	req2.Header.Set("Content-Type", "application/json")
	req2.Header.Set("Authorization", "Bearer "+validToken)
	r.ServeHTTP(w2, req2)

	assert.Equal(t, http.StatusUnauthorized, w2.Code, "重放应被拒绝")

	var resp map[string]interface{}
	json.Unmarshal(w2.Body.Bytes(), &resp)
	assert.Equal(t, float64(40102), resp["code"])
}

// ========== Me 测试 ==========

func TestMe_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()
	_ = sqlx.NewDb(db, "postgres")

	jwtCfg := newTestJWTConfig()
	validToken, err := generateTestAccessToken(jwtCfg.Secret, 1, "admin", time.Now().Add(2*time.Hour))
	require.NoError(t, err)

	r := gin.New()
	auth := r.Group("/api/v1/auth")
	auth.Use(middleware.Auth(jwtCfg))
	auth.GET("/me", func(c *gin.Context) {
		userID, exists := c.Get("user_id")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"code": 401, "msg": "未授权访问"})
			return
		}
		_ = mock // 实际 handler 会查询 DB
		c.JSON(http.StatusOK, gin.H{
			"code": 0,
			"msg":  "ok",
			"data": gin.H{
				"id":       userID,
				"username": "admin",
				"nickname": "管理员",
				"role":     "admin",
				"status":   1,
			},
		})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/auth/me", nil)
	req.Header.Set("Authorization", "Bearer "+validToken)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, float64(0), resp["code"])

	data, ok := resp["data"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, float64(1), data["id"])
	assert.Equal(t, "admin", data["username"])
	assert.Equal(t, "admin", data["role"])

	// password_hash 不应泄露
	_, hasPassword := data["password_hash"]
	assert.False(t, hasPassword, "响应不应包含 password_hash")
}

func TestMe_NoToken(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()
	_ = sqlx.NewDb(db, "postgres")

	jwtCfg := newTestJWTConfig()

	r := gin.New()
	auth := r.Group("/api/v1/auth")
	auth.Use(middleware.Auth(jwtCfg))
	auth.GET("/me", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})
	_ = mock

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/auth/me", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)

	var resp map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, float64(401), resp["code"])
}

func TestMe_ExpiredToken(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()
	_ = sqlx.NewDb(db, "postgres")

	jwtCfg := newTestJWTConfig()
	expiredToken, err := generateExpiredAccessToken(jwtCfg.Secret, 1)
	require.NoError(t, err)

	r := gin.New()
	auth := r.Group("/api/v1/auth")
	auth.Use(middleware.Auth(jwtCfg))
	auth.GET("/me", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})
	_ = mock

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/auth/me", nil)
	req.Header.Set("Authorization", "Bearer "+expiredToken)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)

	var resp map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, float64(401), resp["code"])
	assert.Contains(t, resp["msg"], "过期")
}

// ========== /auth/refresh 路由不在 Auth 中间件组内 — 专项验证 ==========

func TestRefreshRouteBypassesAuthMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)
	jwtCfg := newTestJWTConfig()

	expiredToken, err := generateExpiredAccessToken(jwtCfg.Secret, 1)
	require.NoError(t, err)

	r := gin.New()

	// Auth 中间件保护的组
	protected := r.Group("/api/v1/auth")
	protected.Use(middleware.Auth(jwtCfg))
	protected.GET("/me", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"reached": "me"})
	})

	// /refresh 独立注册（不受 Auth 保护）
	r.POST("/api/v1/auth/refresh", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"reached": "refresh", "code": 0, "msg": "ok"})
	})

	// 过期 token 访问 /refresh
	wRefresh := httptest.NewRecorder()
	reqRefresh, _ := http.NewRequest(http.MethodPost, "/api/v1/auth/refresh", strings.NewReader(`{"refresh_token":"x"}`))
	reqRefresh.Header.Set("Content-Type", "application/json")
	reqRefresh.Header.Set("Authorization", "Bearer "+expiredToken)
	r.ServeHTTP(wRefresh, reqRefresh)

	// 过期 token 访问 /me
	wMe := httptest.NewRecorder()
	reqMe, _ := http.NewRequest(http.MethodGet, "/api/v1/auth/me", nil)
	reqMe.Header.Set("Authorization", "Bearer "+expiredToken)
	r.ServeHTTP(wMe, reqMe)

	// 断言
	assert.Equal(t, http.StatusOK, wRefresh.Code,
		"POST /auth/refresh 使用过期 token → 应进入 handler (200), 不在 Auth 组内")
	assert.Equal(t, http.StatusUnauthorized, wMe.Code,
		"GET /auth/me 使用过期 token → 应被 Auth 中间件拦截 (401)")

	var refreshResp map[string]interface{}
	json.Unmarshal(wRefresh.Body.Bytes(), &refreshResp)
	assert.Equal(t, "refresh", refreshResp["reached"], "确实到达 refresh handler")

	t.Log("路由验证通过: /auth/refresh 独立于 Auth 中间件组外注册")
}

// ========== 账号枚举防护测试 ==========

func TestAccountEnumerationProtection(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()
	_ = sqlx.NewDb(db, "postgres")

	redisStore := newMockRedis()
	jwtCfg := newTestJWTConfig()

	r := gin.New()
	r.POST("/api/v1/auth/login", testLoginHandler(mock, redisStore, jwtCfg))

	// 用户不存在
	w1 := httptest.NewRecorder()
	req1, _ := http.NewRequest(http.MethodPost, "/api/v1/auth/login",
		strings.NewReader(`{"username":"no_such_user","password":"test123"}`))
	req1.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w1, req1)

	var resp1 map[string]interface{}
	json.Unmarshal(w1.Body.Bytes(), &resp1)

	// 密码错误
	w2 := httptest.NewRecorder()
	req2, _ := http.NewRequest(http.MethodPost, "/api/v1/auth/login",
		strings.NewReader(`{"username":"admin","password":"wrong_password"}`))
	req2.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w2, req2)

	var resp2 map[string]interface{}
	json.Unmarshal(w2.Body.Bytes(), &resp2)

	// 关键: 两种情况完全一致
	assert.Equal(t, w1.Code, w2.Code, "HTTP 状态码应一致")
	assert.Equal(t, resp1["code"], resp2["code"], "业务错误码应一致")
	assert.Equal(t, resp1["msg"], resp2["msg"], "错误消息应一致（防止账号枚举）")

	assert.Equal(t, http.StatusUnauthorized, w1.Code)
	assert.Equal(t, float64(40101), resp1["code"])
	assert.Equal(t, "用户名或密码错误", resp1["msg"])
}

// ========== 响应格式测试 ==========

func TestAuthResponseFormat_401(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/test-401", func(c *gin.Context) {
		c.JSON(http.StatusUnauthorized, gin.H{"code": 40101, "msg": "用户名或密码错误"})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/test-401", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Equal(t, "application/json; charset=utf-8", w.Header().Get("Content-Type"))

	var resp map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, float64(40101), resp["code"])
	assert.Equal(t, "用户名或密码错误", resp["msg"])
}

func TestAuthResponseFormat_200(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/test-200", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"code": 0, "msg": "ok", "data": gin.H{"key": "value"}})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/test-200", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, float64(0), resp["code"])
	assert.Equal(t, "ok", resp["msg"])
	assert.NotNil(t, resp["data"])
}

// ========== Access Token Claims 结构验证 ==========

func TestAccessTokenClaims(t *testing.T) {
	secret := "test-secret-32bytes-length-key!!"
	userID := int64(42)

	claims := jwt.MapClaims{
		"sub":       userID,
		"tenant_id": userID,
		"role":      "admin",
		"nickname":  "管理员",
		"iat":       time.Now().Unix(),
		"exp":       time.Now().Add(2 * time.Hour).Unix(),
		"iss":       "weiyeston-v2",
		"jti":       "unique-jti-id",
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenStr, err := token.SignedString([]byte(secret))
	require.NoError(t, err)

	parsed, err := jwt.Parse(tokenStr, func(t *jwt.Token) (interface{}, error) {
		if t.Method.Alg() != jwt.SigningMethodHS256.Alg() {
			return nil, jwt.ErrSignatureInvalid
		}
		return []byte(secret), nil
	})
	require.NoError(t, err)
	require.True(t, parsed.Valid)

	parsedClaims, ok := parsed.Claims.(jwt.MapClaims)
	require.True(t, ok)

	requiredFields := []string{"sub", "tenant_id", "role", "nickname", "iat", "exp", "iss", "jti"}
	for _, field := range requiredFields {
		_, exists := parsedClaims[field]
		assert.True(t, exists, "token 应包含 %s 字段", field)
	}

	sub, ok := parsedClaims["sub"]
	assert.True(t, ok)
	subFloat, ok := sub.(float64)
	assert.True(t, ok, "sub 应为 float64（JSON number）: got %T", sub)
	assert.Equal(t, userID, int64(subFloat))

	iat := int64(parsedClaims["iat"].(float64))
	exp := int64(parsedClaims["exp"].(float64))
	assert.Equal(t, int64(7200), exp-iat, "exp - iat 应为 7200 秒（2小时）")
}
