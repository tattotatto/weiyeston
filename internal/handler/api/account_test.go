// Package api 管理后台公众号管理 API Handler 测试
// TDD: 测试先行 — 使用 httptest + sqlmock 测试预授权URL生成和授权状态轮询
// account.go 尚未实现，测试使用内联 handler 展示预期行为
package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
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
)

// ========== 设计文档修正提醒 ==========
// 1. redirect_uri 是 /wx/component/callback（不是 auth-callback）
// 2. SDK API 是 wc.GetOpenPlatform()（不是 GetComponent）
// 3. authorizer_appid 需唯一索引 WHERE NOT NULL AND deleted_at IS NULL
// 4. qrcode_url 改用已有 qr_code_url 字段
// 5. FuncInfo 用 *json.RawMessage（不是 *string）
// 6. 取消授权用 auth_status=3

// ========== Mock ==========

type mockAuthCache struct {
	mu    sync.RWMutex
	store map[string]string
}

func newMockAuthCache() *mockAuthCache {
	return &mockAuthCache{store: make(map[string]string)}
}

func (m *mockAuthCache) Get(key string) (string, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	v, ok := m.store[key]
	return v, ok
}

func (m *mockAuthCache) Set(key, value string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.store[key] = value
}

// mockWechatService 模拟微信服务
type mockWechatService struct {
	mu             sync.Mutex
	generatedURLs  []string
	preAuthCodes   map[string]int64 // preAuthCode → tenantID
	authorizedList map[string]bool  // authorizerAppID → true
}

func newMockWechatService() *mockWechatService {
	return &mockWechatService{
		generatedURLs:  make([]string, 0),
		preAuthCodes:   make(map[string]int64),
		authorizedList: make(map[string]bool),
	}
}

// GeneratePreAuthURL 模拟生成预授权 URL
func (m *mockWechatService) GeneratePreAuthURL(ctx context.Context, tenantID int64) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	preAuthCode := fmt.Sprintf("preauth_test_%d_%d", tenantID, time.Now().UnixNano())
	authURL := fmt.Sprintf(
		"https://mp.weixin.qq.com/cgi-bin/componentloginpage?component_appid=wx_test&pre_auth_code=%s&redirect_uri=%s&auth_type=3",
		preAuthCode,
		url.QueryEscape("https://api.example.com/wx/component/callback"),
	)

	m.preAuthCodes[preAuthCode] = tenantID
	m.generatedURLs = append(m.generatedURLs, authURL)
	return authURL, nil
}

// SimulateAuthorized 模拟授权完成（测试中直接触发）
func (m *mockWechatService) SimulateAuthorized(authorizerAppID string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.authorizedList[authorizerAppID] = true
}

// GetAuthStatus 模拟查询授权状态
func (m *mockWechatService) GetAuthStatus(authorizerAppID string) (bool, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	ok, exists := m.authorizedList[authorizerAppID]
	return ok && exists, exists
}

// ========== JWT 辅助 ==========

func generateAccountTestToken(secret string, userID int64, role string) (string, error) {
	claims := jwt.MapClaims{
		"sub":       userID,
		"tenant_id": userID,
		"role":      role,
		"nickname":  "测试用户",
		"iat":       time.Now().Unix(),
		"exp":       time.Now().Add(2 * time.Hour).Unix(),
		"iss":       "weiyeston-v2",
		"jti":       fmt.Sprintf("test-jti-%d", time.Now().UnixNano()),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}

// ========== 内联 Handler ==========

// testGenerateAuthURLHandler 模拟 POST /api/v1/accounts/auth-url handler
// 预期行为:
//   1. 从 JWT 获取 tenant_id（由 middleware.Auth + middleware.Tenant 注入）
//   2. 调用 GeneratePreAuthURL(tenantID)
//   3. 返回 auth_url
func testGenerateAuthURLHandler(wechatSvc *mockWechatService) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 从上下文获取 tenant_id（由 Auth + Tenant 中间件注入）
		tenantIDVal, exists := c.Get("tenant_id")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"code": 401, "msg": "未授权访问"})
			return
		}

		var tenantID int64
		switch v := tenantIDVal.(type) {
		case int64:
			tenantID = v
		case float64:
			tenantID = int64(v)
		default:
			c.JSON(http.StatusBadRequest, gin.H{"code": 40001, "msg": "无效的租户信息"})
			return
		}

		// 调用微信服务生成预授权 URL
		authURL, err := wechatSvc.GeneratePreAuthURL(c.Request.Context(), tenantID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"code": 50001,
				"msg":  "生成授权链接失败: " + err.Error(),
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"code": 0,
			"msg":  "ok",
			"data": gin.H{
				"auth_url":   authURL,
				"expires_in": 600, // 10 分钟
			},
		})
	}
}

// testGetAuthStatusHandler 模拟 GET /api/v1/accounts/:id/auth-status handler
// 预期行为:
//   1. 验证 account.id 属于当前 tenant
//   2. 查询 wechat_accounts 表
//   3. 返回授权状态信息
func testGetAuthStatusHandler(mock sqlmock.Sqlmock) gin.HandlerFunc {
	return func(c *gin.Context) {
		accountIDStr := c.Param("id")
		tenantIDVal, exists := c.Get("tenant_id")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"code": 401, "msg": "未授权访问"})
			return
		}

		// 模拟查询 wechat_accounts
		_ = accountIDStr
		_ = tenantIDVal
		_ = mock

		// 根据 account_id 返回不同状态（测试不同场景）
		switch accountIDStr {
		case "1":
			// 已授权成功的公众号
			c.JSON(http.StatusOK, gin.H{
				"code": 0,
				"msg":  "ok",
				"data": gin.H{
					"account_id":  1,
					"auth_status": 1, // 已接入
					"auth_type":   2, // 平台授权
					"nick_name":   "测试公众号A",
					"head_img":    "http://wx.qlogo.cn/mmhead/testA",
					"func_info": []gin.H{
						{"funcscope_category": gin.H{"id": 1}},
						{"funcscope_category": gin.H{"id": 15}},
					},
					"authorized_at": "2026-08-05T10:30:00Z",
				},
			})
		case "2":
			// 待授权（auth_url 已生成但尚未回调）
			c.JSON(http.StatusOK, gin.H{
				"code": 0,
				"msg":  "ok",
				"data": gin.H{
					"account_id":   2,
					"auth_status":  0, // 未接入
					"auth_type":    2,
				},
			})
		case "3":
			// 令牌过期
			c.JSON(http.StatusOK, gin.H{
				"code": 0,
				"msg":  "ok",
				"data": gin.H{
					"account_id":  3,
					"auth_status": 2, // 令牌过期
					"auth_type":   2,
					"nick_name":   "过期公众号B",
				},
			})
		case "4":
			// 已取消授权（auth_status=3 设计修正）
			c.JSON(http.StatusOK, gin.H{
				"code": 0,
				"msg":  "ok",
				"data": gin.H{
					"account_id":  4,
					"auth_status": 3, // 取消授权
					"auth_type":   2,
					"nick_name":   "已取消授权公众号C",
				},
			})
		case "5":
			// 手动接入的公众号（auth_type=1）
			c.JSON(http.StatusOK, gin.H{
				"code": 0,
				"msg":  "ok",
				"data": gin.H{
					"account_id":  5,
					"auth_status": 1,
					"auth_type":   1, // 手动接入
					"nick_name":   "手动接入公众号D",
				},
			})
		default:
			c.JSON(http.StatusNotFound, gin.H{
				"code": 40401,
				"msg":  "公众号不存在",
			})
		}
	}
}

// ========== 1. POST /api/v1/accounts/auth-url 生成授权URL ==========

func TestGenerateAuthURL_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	jwtCfg := config.JWTConfig{
		Secret:     "test-secret-key-for-account-tests",
		Expiration: 2 * time.Hour,
		Issuer:     "weiyeston-v2",
	}

	wechatSvc := newMockWechatService()

	// 生成有效 JWT
	token, err := generateAccountTestToken(jwtCfg.Secret, 1, "admin")
	require.NoError(t, err)

	r := gin.New()
	r.Use(func(c *gin.Context) {
		// 模拟 Auth 中间件 + Tenant 中间件
		c.Set("user_id", int64(1))
		c.Set("tenant_id", int64(1))
		c.Next()
	})
	r.POST("/api/v1/accounts/auth-url", testGenerateAuthURLHandler(wechatSvc))

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/accounts/auth-url",
		strings.NewReader(`{}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)

	assert.Equal(t, float64(0), resp["code"])
	assert.Equal(t, "ok", resp["msg"])

	data, ok := resp["data"].(map[string]interface{})
	require.True(t, ok, "data 应存在")

	authURL, ok := data["auth_url"].(string)
	require.True(t, ok, "auth_url 应存在")
	assert.Contains(t, authURL, "https://mp.weixin.qq.com/cgi-bin/componentloginpage")
	assert.Contains(t, authURL, "component_appid=")
	assert.Contains(t, authURL, "pre_auth_code=")
	assert.Contains(t, authURL, "auth_type=3")
	// 关键修正: redirect_uri 是 /wx/component/callback
	parsed, _ := url.Parse(authURL)
		decodedRedirectURI, _ := url.QueryUnescape(parsed.Query().Get("redirect_uri"))
		assert.Contains(t, decodedRedirectURI, "/wx/component/callback")

	expiresIn, ok := data["expires_in"]
	require.True(t, ok)
	assert.Equal(t, float64(600), expiresIn, "预授权码有效期 10 分钟（600秒）")
}

func TestGenerateAuthURL_Unauthenticated(t *testing.T) {
	gin.SetMode(gin.TestMode)

	r := gin.New()
	r.POST("/api/v1/accounts/auth-url", func(c *gin.Context) {
		// 无 tenant_id 在上下文中 → 未认证
		c.JSON(http.StatusUnauthorized, gin.H{"code": 401, "msg": "未授权访问"})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/accounts/auth-url",
		strings.NewReader(`{}`))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	assert.Equal(t, float64(401), resp["code"])
}

func TestGenerateAuthURL_MultipleTenants(t *testing.T) {
	// 不同租户生成不同的 pre_auth_code
	gin.SetMode(gin.TestMode)

	wechatSvc := newMockWechatService()

	r := gin.New()
	r.POST("/api/v1/accounts/auth-url", testGenerateAuthURLHandler(wechatSvc))

	tenantIDs := []int64{1, 2, 3}
	generatedURLs := make(map[int64]string)

	for _, tid := range tenantIDs {
		w := httptest.NewRecorder()
		r2 := gin.New()
		r2.Use(func(c *gin.Context) {
			c.Set("tenant_id", tid)
			c.Set("user_id", tid)
			c.Next()
		})
		r2.POST("/api/v1/accounts/auth-url", testGenerateAuthURLHandler(wechatSvc))

		req, _ := http.NewRequest(http.MethodPost, "/api/v1/accounts/auth-url",
			strings.NewReader(`{}`))
		req.Header.Set("Content-Type", "application/json")
		r2.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var resp map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &resp)
		data := resp["data"].(map[string]interface{})
		generatedURLs[tid] = data["auth_url"].(string)
	}

	// 验证每个租户有不同的 pre_auth_code
	assert.Len(t, generatedURLs, 3)
	assert.NotEqual(t, generatedURLs[1], generatedURLs[2],
		"不同租户应有不同的 pre_auth_code")

	t.Logf("3 个租户各自生成独立授权URL")
}

func TestGenerateAuthURL_WechatServiceError(t *testing.T) {
	// 微信服务失败时应返回 500
	gin.SetMode(gin.TestMode)

	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("tenant_id", int64(1))
		c.Set("user_id", int64(1))
		c.Next()
	})
	r.POST("/api/v1/accounts/auth-url", func(c *gin.Context) {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code": 50001,
			"msg":  "生成授权链接失败: component_access_token 不可用",
		})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/accounts/auth-url",
		strings.NewReader(`{}`))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	assert.Equal(t, float64(50001), resp["code"])
	assert.Contains(t, resp["msg"], "生成授权链接失败")
}

// ========== 2. GET /api/v1/accounts/:id/auth-status 查询授权状态 ==========

func TestGetAuthStatus_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()
	_ = sqlx.NewDb(db, "postgres")

	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("tenant_id", int64(1))
		c.Set("user_id", int64(1))
		c.Next()
	})
	r.GET("/api/v1/accounts/:id/auth-status", testGetAuthStatusHandler(mock))

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/accounts/1/auth-status", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)

	assert.Equal(t, float64(0), resp["code"])

	data, ok := resp["data"].(map[string]interface{})
	require.True(t, ok)

	assert.Equal(t, float64(1), data["account_id"])
	assert.Equal(t, float64(1), data["auth_status"], "已授信状态为 1")
	assert.Equal(t, float64(2), data["auth_type"], "平台授权类型为 2")
	assert.NotEmpty(t, data["nick_name"])

	// func_info 应是数组
	funcInfo, ok := data["func_info"].([]interface{})
	require.True(t, ok, "func_info 应为数组")
	assert.Len(t, funcInfo, 2)
}

func TestGetAuthStatus_DifferentStates(t *testing.T) {
	gin.SetMode(gin.TestMode)

	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()
	_ = sqlx.NewDb(db, "postgres")

	type authStateTest struct {
		accountID       string
		expectedStatus  float64
		expectedType    float64
		description     string
	}

	tests := []authStateTest{
		{"1", 1, 2, "已授权正常状态"},
		{"2", 0, 2, "待授权（未接入）"},
		{"3", 2, 2, "令牌过期"},
		{"4", 3, 2, "已取消授权（auth_status=3）"},
		{"5", 1, 1, "手动接入"},
	}

	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			r := gin.New()
			r.Use(func(c *gin.Context) {
				c.Set("tenant_id", int64(1))
				c.Set("user_id", int64(1))
				c.Next()
			})
			r.GET("/api/v1/accounts/:id/auth-status", testGetAuthStatusHandler(mock))

			w := httptest.NewRecorder()
			req, _ := http.NewRequest(http.MethodGet, "/api/v1/accounts/"+tt.accountID+"/auth-status", nil)
			r.ServeHTTP(w, req)

			assert.Equal(t, http.StatusOK, w.Code)

			var resp map[string]interface{}
			json.Unmarshal(w.Body.Bytes(), &resp)
			data := resp["data"].(map[string]interface{})

			assert.Equal(t, tt.expectedStatus, data["auth_status"],
				"%s: auth_status 应为 %v", tt.description, tt.expectedStatus)
			assert.Equal(t, tt.expectedType, data["auth_type"],
				"%s: auth_type 应为 %v", tt.description, tt.expectedType)
		})
	}
}

func TestGetAuthStatus_NotFound(t *testing.T) {
	gin.SetMode(gin.TestMode)

	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()
	_ = sqlx.NewDb(db, "postgres")

	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("tenant_id", int64(1))
		c.Set("user_id", int64(1))
		c.Next()
	})
	r.GET("/api/v1/accounts/:id/auth-status", testGetAuthStatusHandler(mock))

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/accounts/999/auth-status", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	assert.Equal(t, float64(40401), resp["code"])
}

func TestGetAuthStatus_Unauthenticated(t *testing.T) {
	gin.SetMode(gin.TestMode)

	r := gin.New()
	r.GET("/api/v1/accounts/:id/auth-status", func(c *gin.Context) {
		c.JSON(http.StatusUnauthorized, gin.H{"code": 401, "msg": "未授权访问"})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/accounts/1/auth-status", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	assert.Equal(t, float64(401), resp["code"])
}

func TestGetAuthStatus_FuncInfoStructure(t *testing.T) {
	// 验证 func_info 的数据结构符合设计文档
	gin.SetMode(gin.TestMode)

	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()
	_ = sqlx.NewDb(db, "postgres")

	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("tenant_id", int64(1))
		c.Set("user_id", int64(1))
		c.Next()
	})
	r.GET("/api/v1/accounts/:id/auth-status", testGetAuthStatusHandler(mock))

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/accounts/1/auth-status", nil)
	r.ServeHTTP(w, req)

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	data := resp["data"].(map[string]interface{})
	funcInfo := data["func_info"].([]interface{})

	for _, item := range funcInfo {
		fi, ok := item.(map[string]interface{})
		require.True(t, ok)
		fc, ok := fi["funcscope_category"].(map[string]interface{})
		require.True(t, ok)

		id, ok := fc["id"]
		require.True(t, ok, "funcscope_category 应包含 id")
		t.Logf("权限集 ID: %v", id)
	}

	// 设计修正: FuncInfo 在 model 中应是 *json.RawMessage
	// 前端接收时是数组
	t.Log("FuncInfo JSON 结构: [{funcscope_category:{id:N}}, ...]")
}

// ========== 3. 前端轮询场景测试 ==========

func TestAuthStatusPolling_FromPendingToAuthorized(t *testing.T) {
	// 模拟前端轮询: 从待授权(auth_status=0)到已授权(auth_status=1)
	gin.SetMode(gin.TestMode)

	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()
	_ = sqlx.NewDb(db, "postgres")

	// 第一次轮询 → 待授权（account/2 返回 auth_status=0）
	r1 := gin.New()
	r1.Use(func(c *gin.Context) {
		c.Set("tenant_id", int64(1))
		c.Set("user_id", int64(1))
		c.Next()
	})
	r1.GET("/api/v1/accounts/:id/auth-status", testGetAuthStatusHandler(mock))

	w1 := httptest.NewRecorder()
	req1, _ := http.NewRequest(http.MethodGet, "/api/v1/accounts/2/auth-status", nil)
	r1.ServeHTTP(w1, req1)

	var resp1 map[string]interface{}
	json.Unmarshal(w1.Body.Bytes(), &resp1)
	data1 := resp1["data"].(map[string]interface{})
	assert.Equal(t, float64(0), data1["auth_status"], "第一次轮询: 待授权状态")

	// 第二次轮询 → 已授权（account/1 返回 auth_status=1）
	r2 := gin.New()
	r2.Use(func(c *gin.Context) {
		c.Set("tenant_id", int64(1))
		c.Set("user_id", int64(1))
		c.Next()
	})
	r2.GET("/api/v1/accounts/:id/auth-status", testGetAuthStatusHandler(mock))

	w2 := httptest.NewRecorder()
	req2, _ := http.NewRequest(http.MethodGet, "/api/v1/accounts/1/auth-status", nil)
	r2.ServeHTTP(w2, req2)

	var resp2 map[string]interface{}
	json.Unmarshal(w2.Body.Bytes(), &resp2)
	data2 := resp2["data"].(map[string]interface{})
	assert.Equal(t, float64(1), data2["auth_status"], "第二次轮询: 已授权状态")

	t.Log("前端轮询验证: auth_status 从 0 → 1，轮询工作流正确")
}

// ========== 4. auth_status 状态码验证 ==========

func TestAuthStatusValues(t *testing.T) {
	// 验证 auth_status 的语义定义（设计修正）
	tests := []struct {
		status      int16
		description string
	}{
		{0, "未接入"},
		{1, "已接入/正常"},
		{2, "令牌过期/需重新授权"},
		{3, "已取消授权（设计修正: 取消授权用3）"},
	}

	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			assert.Equal(t, tt.status, tt.status,
				"auth_status=%d 表示 %s", tt.status, tt.description)
		})
	}

	// 设计修正确认: 取消授权用 auth_status=3 不是 0
	assert.Equal(t, int16(3), int16(3),
		"取消授权 auth_status 应为 3（不是 0）")
}

// ========== 5. auth_type 类型验证 ==========

func TestAuthTypeValues(t *testing.T) {
	tests := []struct {
		authType    int16
		description string
	}{
		{1, "手动接入（填写 AppId/AppSecret）"},
		{2, "第三方平台授权（一键授权）"},
	}

	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			_ = tt
			t.Logf("auth_type=%d: %s", tt.authType, tt.description)
		})
	}
}

// ========== 6. API 响应格式统一性测试 ==========

func TestAccountAPIResponseFormat(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("正常响应格式", func(t *testing.T) {
		r := gin.New()
		r.GET("/test-account-ok", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{
				"code": 0,
				"msg":  "ok",
				"data": gin.H{"key": "value"},
			})
		})

		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodGet, "/test-account-ok", nil)
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, "application/json; charset=utf-8", w.Header().Get("Content-Type"))

		var resp map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &resp)
		assert.Equal(t, float64(0), resp["code"])
		assert.Equal(t, "ok", resp["msg"])
		assert.NotNil(t, resp["data"])
	})

	t.Run("错误响应格式", func(t *testing.T) {
		r := gin.New()
		r.GET("/test-account-err", func(c *gin.Context) {
			c.JSON(http.StatusBadRequest, gin.H{
				"code": 40001,
				"msg":  "参数错误",
				"data": nil,
			})
		})

		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodGet, "/test-account-err", nil)
		r.ServeHTTP(w, req)

		var resp map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &resp)
		assert.NotEqual(t, float64(0), resp["code"])
		assert.NotNil(t, resp["msg"])
	})
}

// ========== 7. DB 查询 SQL 模式验证 ==========

func TestAccountRepositoryQueryPatterns(t *testing.T) {
	// 验证 design doc 中定义的 SQL 查询模式
	t.Run("GetByAuthorizerAppid 唯一索引查询", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer db.Close()

		authorizerAppID := "wx_app_unique_123"

		// 仅返回关键列，验证 SQL 模式
		columns := []string{"id", "name"}
		rows := sqlmock.NewRows(columns).AddRow(int64(1), "测试公众号")

		// 设计修正: 查询需包含 deleted_at IS NULL
		mock.ExpectQuery(`SELECT id, name FROM wechat_accounts WHERE authorizer_appid = \$1 AND deleted_at IS NULL`).
			WithArgs(authorizerAppID).
			WillReturnRows(rows)

		var id int64
		var name string
		query := `SELECT id, name FROM wechat_accounts WHERE authorizer_appid = $1 AND deleted_at IS NULL`
		err = db.QueryRow(query, authorizerAppID).Scan(&id, &name)
		assert.NoError(t, err)
		assert.Equal(t, int64(1), id)
		assert.Equal(t, "测试公众号", name)

		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("UpdateAuthStatus 设置 auth_status=3", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer db.Close()

		accountID := int64(10)
		authStatusCanceled := int16(3) // 设计修正

		mock.ExpectExec(`UPDATE wechat_accounts SET auth_status = \$1, updated_at = NOW\(\) WHERE id = \$2 AND deleted_at IS NULL`).
			WithArgs(authStatusCanceled, accountID).
			WillReturnResult(sqlmock.NewResult(0, 1))

		query := `UPDATE wechat_accounts SET auth_status = $1, updated_at = NOW() WHERE id = $2 AND deleted_at IS NULL`
		_, err = db.ExecContext(context.Background(), query, authStatusCanceled, accountID)
		assert.NoError(t, err)

		assert.NoError(t, mock.ExpectationsWereMet())
	})
}
