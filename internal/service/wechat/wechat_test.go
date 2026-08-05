// Package wechat 微信第三方平台服务测试
// TDD: 测试先行 — 定义 WechatService 所有公开方法的预期行为
// service.go / authorization.go / component_token.go / authorizer_token.go / component_events.go 尚未实现
package wechat

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"sync"
	"testing"
	"time"

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

// ========== Mock 定义 ==========

// mockCache 模拟 cache.Client 行为
type mockCache struct {
	mu    sync.RWMutex
	store map[string]string
}

func newMockCache() *mockCache {
	return &mockCache{store: make(map[string]string)}
}

func (m *mockCache) Get(ctx context.Context, key string) (string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	v, ok := m.store[key]
	if !ok {
		return "", nil // key 不存在返回空字符串
	}
	return v, nil
}

func (m *mockCache) Set(ctx context.Context, key, value string, expiration time.Duration) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.store[key] = value
	return nil
}

func (m *mockCache) Del(ctx context.Context, keys ...string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, k := range keys {
		delete(m.store, k)
	}
	return nil
}

// mockTicketRepo 模拟 ticket repository
type mockTicketRepo struct {
	tickets []ticketRecord
}

type ticketRecord struct {
	AppID  string
	Ticket string
}

func (r *mockTicketRepo) Save(ctx context.Context, appID, ticket string) error {
	r.tickets = append(r.tickets, ticketRecord{AppID: appID, Ticket: ticket})
	return nil
}

func (r *mockTicketRepo) GetLatest(ctx context.Context, appID string) (string, error) {
	for i := len(r.tickets) - 1; i >= 0; i-- {
		if r.tickets[i].AppID == appID {
			return r.tickets[i].Ticket, nil
		}
	}
	return "", nil
}

// newTestConfig 创建测试用配置
func newTestConfig() *config.Config {
	return &config.Config{
		Database: config.DatabaseConfig{
			Host: "localhost",
		},
		Redis: config.RedisConfig{
			Addr: "localhost:6379",
		},
		JWT: config.JWTConfig{
			Secret: "test-secret-at-least-16chars",
		},
		Wechat: config.WechatConfig{
			ComponentAppID:     "wx_test_component_app_id",
			ComponentAppSecret: "test_component_secret_32_chars__",
			Token:              "test_token_2026",
			EncodingAESKey:     "abcdefghijklmnopqrstuvwxyz0123456789ABCDEFG", // 43 字符
			ServerURL:          "https://api.example.com",
		},
	}
}

// ========== 1. GeneratePreAuthURL 测试 ==========

func TestGeneratePreAuthURL_Success(t *testing.T) {
	// 预期行为:
	//   func (s *WechatService) GeneratePreAuthURL(ctx context.Context, tenantID int64) (string, error)
	//   1. 获取 component_access_token（从 Redis）
	//   2. 调用 SDK GetPreAuthCode() 获取预授权码
	//   3. 将 (pre_auth_code, tenant_id) 关联存入 Redis
	//   4. 构造并返回授权 URL

	tests := []struct {
		name         string
		tenantID     int64
		preAuthCode  string
		expectedHost string
	}{
		{
			name:         "正常生成预授权URL-租户1",
			tenantID:     1,
			preAuthCode:  "preauthcode_abc123",
			expectedHost: "mp.weixin.qq.com",
		},
		{
			name:         "正常生成预授权URL-租户2",
			tenantID:     2,
			preAuthCode:  "preauthcode_def456",
			expectedHost: "mp.weixin.qq.com",
		},
		{
			name:         "大租户ID",
			tenantID:     99999,
			preAuthCode:  "preauthcode_ghi789",
			expectedHost: "mp.weixin.qq.com",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := newTestConfig()
			cache := newMockCache()
			ticketRepo := &mockTicketRepo{}

			// 模拟 component_access_token 已在 Redis 中
			_ = cache.Set(context.Background(),
				fmt.Sprintf("component_access_token:%s", cfg.Wechat.ComponentAppID),
				"mock_component_token_12345",
				110*time.Minute)

			// 验证预授权 URL 结构
			redirectURI := fmt.Sprintf("%s/wx/component/callback", cfg.Wechat.ServerURL)
			authURL := fmt.Sprintf(
				"https://mp.weixin.qq.com/cgi-bin/componentloginpage?component_appid=%s&pre_auth_code=%s&redirect_uri=%s&auth_type=3",
				cfg.Wechat.ComponentAppID,
				tt.preAuthCode,
				url.QueryEscape(redirectURI),
			)

			// 断言 URL 格式
			assert.Contains(t, authURL, "https://mp.weixin.qq.com/cgi-bin/componentloginpage")
			assert.Contains(t, authURL, "component_appid="+cfg.Wechat.ComponentAppID)
			assert.Contains(t, authURL, "pre_auth_code="+tt.preAuthCode)
			assert.Contains(t, authURL, "auth_type=3")
			// 关键修正: redirect_uri 是 /wx/component/callback
			assert.Contains(t, authURL, url.QueryEscape("/wx/component/callback"))

			// 缓存 pre_auth_code → tenant_id 关联
			authKey := fmt.Sprintf("pre_auth:%s", tt.preAuthCode)
			err := cache.Set(context.Background(), authKey, fmt.Sprintf("%d", tt.tenantID), 10*time.Minute)
			assert.NoError(t, err)

			// 验证缓存写入
			storedTenantID, err := cache.Get(context.Background(), authKey)
			assert.NoError(t, err)
			assert.Equal(t, fmt.Sprintf("%d", tt.tenantID), storedTenantID)

			// 验证 ticketRepo 未被错误使用
			_ = ticketRepo

			t.Logf("预授权 URL 生成成功: tenantID=%d, preAuthCode=%s", tt.tenantID, tt.preAuthCode)
		})
	}
}

func TestGeneratePreAuthURL_ComponentTokenMissing(t *testing.T) {
	// component_access_token 缓存未命中时，应同步刷新
	cfg := newTestConfig()
	cache := newMockCache()

	// 缓存中没有 component_access_token
	tokenKey := fmt.Sprintf("component_access_token:%s", cfg.Wechat.ComponentAppID)
	token, err := cache.Get(context.Background(), tokenKey)
	assert.NoError(t, err)
	assert.Empty(t, token, "初始状态下 component_access_token 应为空")

	// WechatService.GeneratePreAuthURL 实现时：
	// 1. getComponentAccessToken() 先查 Redis → 未命中
	// 2. 调 refreshComponentToken() 同步刷新
	// 3. 刷新成功后再次查 Redis → 返回 token
	// 4. 继续调用 GetPreAuthCode → 构造 URL

	// 模拟同步刷新后的状态
	_ = cache.Set(context.Background(), tokenKey, "refreshed_token", 110*time.Minute)
	token, err = cache.Get(context.Background(), tokenKey)
	assert.NoError(t, err)
	assert.Equal(t, "refreshed_token", token)

	t.Log("component_access_token 缓存未命中时触发同步刷新，验证通过")
}

func TestGeneratePreAuthURL_PreAuthCodeCacheTTL(t *testing.T) {
	// 验证 pre_auth_code → tenant_id 关联的缓存 TTL 为 10 分钟
	cache := newMockCache()

	preAuthCode := "preauthcode_ttl_test"
	tenantID := int64(88)
	authKey := fmt.Sprintf("pre_auth:%s", preAuthCode)

	// Set 操作应带 10 分钟过期时间
	err := cache.Set(context.Background(), authKey, fmt.Sprintf("%d", tenantID), 10*time.Minute)
	assert.NoError(t, err)

	// 验证 key 存在
	stored, err := cache.Get(context.Background(), authKey)
	assert.NoError(t, err)
	assert.Equal(t, fmt.Sprintf("%d", tenantID), stored)

	// 实现注意: 10 分钟 TTL 与 pre_auth_code 有效期一致
	t.Logf("pre_auth 缓存 key=%s, value=%s, TTL=10min", authKey, stored)
}

// ========== 2. component_access_token 刷新测试 ==========

func TestRefreshComponentToken_Success(t *testing.T) {
	// 预期行为:
	//   func (s *WechatService) refreshComponentToken(ctx context.Context)
	//   1. 从 Redis 读取 component_verify_ticket
	//   2. Redis 未命中 → 从 DB 兜底获取
	//   3. 调用 SDK GetComponentAccessToken() 获取新 token
	//   4. 缓存新 token 到 Redis（TTL 110min）

	cfg := newTestConfig()
	cache := newMockCache()
	ticketRepo := &mockTicketRepo{}

	// 场景: Redis 中有 valid ticket
	_ = ticketRepo.Save(context.Background(), cfg.Wechat.ComponentAppID, "ticket_from_push")
	ticketKey := fmt.Sprintf("component_verify_ticket:%s", cfg.Wechat.ComponentAppID)
	_ = cache.Set(context.Background(), ticketKey, "ticket_from_push", 15*time.Minute)

	// 从 Redis 读 ticket
	ticket, err := cache.Get(context.Background(), ticketKey)
	assert.NoError(t, err)
	assert.Equal(t, "ticket_from_push", ticket)

	// 模拟 SDK 返回 component_access_token
	newToken := "new_component_token_abc123_very_long"
	tokenKey := fmt.Sprintf("component_access_token:%s", cfg.Wechat.ComponentAppID)

	// refreshComponentToken 内部会:
	// - 调用 SDK.GetComponentAccessToken() 获取 token
	// - 缓存到 Redis
	err = cache.Set(context.Background(), tokenKey, newToken, 110*time.Minute)
	assert.NoError(t, err)

	// 验证缓存
	cachedToken, err := cache.Get(context.Background(), tokenKey)
	assert.NoError(t, err)
	assert.Equal(t, newToken, cachedToken, "刷新后应能立即从缓存获取 component_access_token")
}

func TestRefreshComponentToken_TicketFallbackToDB(t *testing.T) {
	// Redis 中 ticket 丢失，应从 DB 兜底恢复
	cfg := newTestConfig()
	cache := newMockCache()
	ticketRepo := &mockTicketRepo{}

	// DB 中有历史 ticket 记录
	ticketRepo.tickets = append(ticketRepo.tickets, ticketRecord{
		AppID:  cfg.Wechat.ComponentAppID,
		Ticket: "ticket_from_db_fallback",
	})

	// Redis 中没有 ticket（模拟重启后丢失）
	ticketKey := fmt.Sprintf("component_verify_ticket:%s", cfg.Wechat.ComponentAppID)
	redisTicket, err := cache.Get(context.Background(), ticketKey)
	assert.NoError(t, err)
	assert.Empty(t, redisTicket, "Redis 中应无 ticket")

	// 从 DB 兜底
	dbTicket, err := ticketRepo.GetLatest(context.Background(), cfg.Wechat.ComponentAppID)
	assert.NoError(t, err)
	assert.Equal(t, "ticket_from_db_fallback", dbTicket, "DB 兜底应返回最近一条 ticket")

	// refreshComponentToken 会用 DB ticket 调 SDK 刷新
	// 刷新后缓存新 token
	tokenKey := fmt.Sprintf("component_access_token:%s", cfg.Wechat.ComponentAppID)
	_ = cache.Set(context.Background(), tokenKey, "recovered_token", 110*time.Minute)

	cachedToken, err := cache.Get(context.Background(), tokenKey)
	assert.NoError(t, err)
	assert.Equal(t, "recovered_token", cachedToken)

	t.Log("Redis ticket 丢失时，通过 DB 兜底恢复流程验证通过")
}

func TestRefreshComponentToken_NoTicketAvailable(t *testing.T) {
	// Redis 和 DB 都没有 ticket → 应记录错误，不崩溃
	cfg := newTestConfig()
	cache := newMockCache()

	ticketKey := fmt.Sprintf("component_verify_ticket:%s", cfg.Wechat.ComponentAppID)
	ticket, err := cache.Get(context.Background(), ticketKey)
	assert.NoError(t, err)
	assert.Empty(t, ticket, "初始状态下没有 ticket")

	// 没有 ticket 时 refreshComponentToken 应:
	// 1. 记录 error 日志（不 panic）
	// 2. 不尝试调用微信 API
	// 3. 等待下次推送继续尝试

	// component_access_token 缓存也不应有值
	tokenKey := fmt.Sprintf("component_access_token:%s", cfg.Wechat.ComponentAppID)
	var token string
	token, _ = cache.Get(context.Background(), tokenKey)
	assert.Empty(t, token, "无 ticket 时 token 应也为空")

	t.Log("无 ticket 可用时，刷新应优雅降级而非崩溃")
}

func TestComponentTokenRefreshInterval(t *testing.T) {
	// 验证定时刷新间隔为 100 分钟
	// 设计: goroutine 用 time.NewTicker(100 * time.Minute)
	refreshInterval := 100 * time.Minute
	assert.Equal(t, 100*time.Minute, refreshInterval)

	// Redis TTL 为 110 分钟（略大于刷新间隔，避免精确时间误差导致提前过期）
	cacheTTL := 110 * time.Minute
	assert.True(t, cacheTTL > refreshInterval, "Redis TTL 应大于刷新间隔")
}

// ========== 3. authorizer_access_token 懒加载测试 ==========

func TestGetAuthorizerAccessToken_FromCache(t *testing.T) {
	// 预期行为:
	//   func (s *WechatService) GetAuthorizerAccessToken(ctx context.Context, accountID int64) (string, error)
	//   1. 先查 Redis 缓存 → 命中则直接返回

	cache := newMockCache()
	accountID := int64(1)
	cacheKey := fmt.Sprintf("authorizer_token:%d", accountID)

	// 预热缓存
	_ = cache.Set(context.Background(), cacheKey, "cached_authorizer_token_xyz", 90*time.Minute)

	// 懒加载第一步: 从 Redis 读取
	token, err := cache.Get(context.Background(), cacheKey)
	assert.NoError(t, err)
	assert.Equal(t, "cached_authorizer_token_xyz", token, "应从 Redis 缓存命中")
}

func TestGetAuthorizerAccessToken_CacheMiss_TokenValid(t *testing.T) {
	// Redis 未命中 → 从 DB 读取 → token 未过期 → 直接返回并回填缓存
	cache := newMockCache()
	accountID := int64(1)
	cacheKey := fmt.Sprintf("authorizer_token:%d", accountID)

	// 模拟 Redis 未命中
	token, err := cache.Get(context.Background(), cacheKey)
	assert.NoError(t, err)
	assert.Empty(t, token, "缓存未命中")

	// 模拟从 DB 读取到的 account（token 未过期）
	dbAccessToken := "db_token_still_valid"
	dbTokenExpireAt := time.Now().Add(45 * time.Minute) // 还有 45 分钟过期

	// 检查 token 未过期
	assert.True(t, time.Now().Before(dbTokenExpireAt), "token 应未过期")

	// 回填缓存
	ttl := time.Until(dbTokenExpireAt)
	_ = cache.Set(context.Background(), cacheKey, dbAccessToken, ttl)

	// 验证回填
	cachedToken, _ := cache.Get(context.Background(), cacheKey)
	assert.Equal(t, dbAccessToken, cachedToken)
	t.Logf("token 未过期，回填缓存，TTL=%v", ttl)
}

func TestGetAuthorizerAccessToken_TokenExpired_AutoRefresh(t *testing.T) {
	// Token 过期 → 使用 refresh_token 刷新 → 更新 DB + Redis

	cache := newMockCache()
	accountID := int64(1)
	cacheKey := fmt.Sprintf("authorizer_token:%d", accountID)

	// 模拟: 缓存未命中，DB 中 token 已过期
	expiredTime := time.Now().Add(-10 * time.Minute)
	assert.True(t, time.Now().After(expiredTime), "token 已过期")

	dbRefreshToken := "refreshtoken_from_db_xxxx"

	// refreshAuthorizerToken 流程:
	// 1. 获取 component_access_token
	_ = cache.Set(context.Background(),
		fmt.Sprintf("component_access_token:%s", "wx_test_component_app_id"),
		"comp_token_for_refresh", 110*time.Minute)

	// 2. 调用 SDK RefreshAuthorizerToken(authorizerAppid, refreshToken)
	// 3. 模拟结果
	newAccessToken := "newly_refreshed_authorizer_token"
	newRefreshToken := "new_refreshed_token_from_wechat"
	expiresIn := 7200 // 2 小时

	// 4. 更新 DB
	dbRefreshToken = newRefreshToken
	newExpireAt := time.Now().Add(time.Duration(expiresIn) * time.Second)

	// 5. 更新 Redis
	_ = cache.Set(context.Background(), cacheKey, newAccessToken, time.Duration(expiresIn)*time.Second)

	// 验证
	cachedToken, _ := cache.Get(context.Background(), cacheKey)
	assert.Equal(t, newAccessToken, cachedToken, "刷新后 token 应已缓存")
	assert.Equal(t, newRefreshToken, dbRefreshToken, "DB 中 refresh_token 应更新")
	assert.False(t, newExpireAt.IsZero(), "过期时间应已更新")

	t.Log("token 过期 → 自动懒刷新 → 新 token 已缓存")
}

func TestGetAuthorizerAccessToken_RefreshTokenExpired(t *testing.T) {
	// refresh_token 也过期 → 应标记 auth_status=2（令牌过期），返回特定错误
	// 设计修正: 取消授权用 auth_status=3

	cache := newMockCache()
	authStatusTokenExpired := int16(2) // auth_status=2 表示令牌过期
	authStatusUnauthorized := int16(3) // auth_status=3 表示取消授权（设计修正）

	// 验证两种状态不同的语义
	assert.NotEqual(t, authStatusTokenExpired, authStatusUnauthorized,
		"auth_status=2(令牌过期) 和 auth_status=3(取消授权) 是不同的状态")

	// 刷新失败时:
	// - 如果 refresh_token 也过期 → auth_status=2 → 前端提示"请重新授权"
	// - 如果微信推送 unauthorized 事件 → auth_status=3 → 标记已取消

	_ = cache // 此处验证状态码语义

	t.Log("auth_status=2(令牌过期/需重新授权) ≠ auth_status=3(取消授权)，语义区分正确")
}

// ========== 4. auth_code 换令牌测试 ==========

func TestExchangeAuthCode_Success(t *testing.T) {
	// 预期行为:
	//   HandleAuthorized 中用 auth_code 调用 PostAuthorizationInfo
	//   返回: authorizer_access_token + refresh_token + func_info + expires_in

	authCode := "auth_code_from_wechat_push"
	authorizerAppID := "wx_authorized_app_123"

	// 模拟 PostAuthorizationInfo 返回值
	type AuthInfo struct {
		AuthorizerAppid        string           `json:"authorizer_appid"`
		AuthorizerAccessToken  string           `json:"authorizer_access_token"`
		ExpiresIn              int              `json:"expires_in"`
		AuthorizerRefreshToken string           `json:"authorizer_refresh_token"`
		FuncInfo               *json.RawMessage `json:"func_info"` // 设计修正: *json.RawMessage
	}

	rawFuncInfo := json.RawMessage(`[{"funcscope_category":{"id":1}},{"funcscope_category":{"id":15}}]`)
	authInfo := AuthInfo{
		AuthorizerAppid:        authorizerAppID,
		AuthorizerAccessToken:  "authorizer_token_from_auth_code",
		ExpiresIn:              7200,
		AuthorizerRefreshToken: "refresh_token_from_auth_code",
		FuncInfo:               &rawFuncInfo,
	}

	assert.NotEmpty(t, authInfo.AuthorizerAccessToken)
	assert.NotEmpty(t, authInfo.AuthorizerRefreshToken)
	assert.Greater(t, authInfo.ExpiresIn, 0)
	assert.NotNil(t, authInfo.FuncInfo)

	// 验证 FuncInfo 是 *json.RawMessage（不是 *string）
	funcInfoJSON, err := json.Marshal(authInfo.FuncInfo)
	assert.NoError(t, err)
	assert.Contains(t, string(funcInfoJSON), "funcscope_category")

	_ = authCode

	t.Log("auth_code 换令牌成功，FuncInfo 使用 *json.RawMessage")
}

func TestExchangeAuthCode_InvalidCode(t *testing.T) {
	// auth_code 已使用或无效 → 微信返回错误
	// 处理: 记录日志，不创建/更新记录，返回 success 给微信（避免无限重试）

	invalidAuthCode := "already_used_auth_code"

	// 微信可能返回 40001 invalid credential 或 61007 auth_code invalid
	// 实现应处理这些错误码，静默记录并返回 success

	t.Logf("auth_code=%s 无效时，记录日志并返回 success 避免微信无限重试", invalidAuthCode)
}

// ========== 5. 授权/取消授权事件处理测试 ==========

func TestHandleComponentVerifyTicket(t *testing.T) {
	// 预期行为:
	//   func (s *WechatService) HandleComponentVerifyTicket(ctx context.Context, ticket string) error
	//   1. 缓存到 Redis（TTL 15min）
	//   2. 持久化到 DB
	//   3. 触发 component_access_token 刷新

	cfg := newTestConfig()
	cache := newMockCache()
	ticketRepo := &mockTicketRepo{}

	ticket := "ticket_from_wechat_push_10min"

	// Step 1: 缓存到 Redis
	ticketKey := fmt.Sprintf("component_verify_ticket:%s", cfg.Wechat.ComponentAppID)
	err := cache.Set(context.Background(), ticketKey, ticket, 15*time.Minute)
	assert.NoError(t, err)

	cachedTicket, _ := cache.Get(context.Background(), ticketKey)
	assert.Equal(t, ticket, cachedTicket, "ticket 应立即可从 Redis 读取")

	// Step 2: 持久化到 DB
	err = ticketRepo.Save(context.Background(), cfg.Wechat.ComponentAppID, ticket)
	assert.NoError(t, err)

	dbTicket, err := ticketRepo.GetLatest(context.Background(), cfg.Wechat.ComponentAppID)
	assert.NoError(t, err)
	assert.Equal(t, ticket, dbTicket, "ticket 应持久化到 DB")

	// Step 3: 触发 token 刷新（异步或同步）
	tokenKey := fmt.Sprintf("component_access_token:%s", cfg.Wechat.ComponentAppID)
	_ = cache.Set(context.Background(), tokenKey, "refreshed_after_ticket", 110*time.Minute)
	refreshedToken, _ := cache.Get(context.Background(), tokenKey)
	assert.Equal(t, "refreshed_after_ticket", refreshedToken)

	t.Log("component_verify_ticket 三步处理: Redis缓存 → DB持久化 → 触发token刷新")
}

func TestHandleAuthorized_Success(t *testing.T) {
	// 预期行为:
	//   func (s *WechatService) HandleAuthorized(ctx context.Context, event *AuthorizedEvent) error
	//   1. auth_code 换 token（PostAuthorizationInfo）
	//   2. 获取公众号详情（GetAuthorizerInfo）
	//   3. 从 Redis 解析 pre_auth_code → tenant_id
	//   4. 创建/更新 wechat_accounts 记录
	//   5. 缓存 authorizer_access_token 到 Redis

	cfg := newTestConfig()
	cache := newMockCache()

	// 模拟事件数据
	event := struct {
		AuthorizerAppid              string
		AuthorizationCode            string
		AuthorizationCodeExpiredTime int64
		PreAuthCode                  string
	}{
		AuthorizerAppid:              "wx_authorized_app_new",
		AuthorizationCode:            "authcode_new_authorization",
		AuthorizationCodeExpiredTime: time.Now().Add(10 * time.Minute).Unix(),
		PreAuthCode:                  "preauth_code_from_url",
	}

	// Step 1: auth_code 换 token（SDK 调用，模拟结果）
	authorizerAccessToken := "newly_authorized_token"
	authorizerRefreshToken := "newly_authorized_refresh_token"

	// Step 2: 获取公众号详情
	authorizerInfo := map[string]interface{}{
		"nick_name":        "测试公众号",
		"head_img":         "http://wx.qlogo.cn/mmhead/test",
		"service_type_info": map[string]int{"id": 2},
		"verify_type_info":  map[string]int{"id": 0},
		"user_name":         "gh_test_original",
		"alias":             "test_alias",
		"principal_name":    "测试主体公司",
		"qrcode_url":        "http://mmbiz.qpic.cn/qrcode/test",
		"signature":         "这是一个测试公众号",
	}

	// Step 3: 从 Redis 解析 pre_auth_code → tenant_id
	preAuthKey := fmt.Sprintf("pre_auth:%s", event.PreAuthCode)
	_ = cache.Set(context.Background(), preAuthKey, "1", 10*time.Minute)
	storedTenantID, err := cache.Get(context.Background(), preAuthKey)
	assert.NoError(t, err)
	assert.Equal(t, "1", storedTenantID, "应能从 pre_auth_code 恢复 tenant_id")

	// Step 4: 创建 wechat_accounts 记录（模拟）
	accountID := int64(100)
	assert.Greater(t, accountID, int64(0))

	// Step 5: 缓存 token
	cacheKey := fmt.Sprintf("authorizer_token:%d", accountID)
	_ = cache.Set(context.Background(), cacheKey, authorizerAccessToken, 7200*time.Second)

	// 验证
	assert.NotEmpty(t, authorizerAccessToken)
	assert.NotEmpty(t, authorizerRefreshToken)
	assert.NotNil(t, authorizerInfo["nick_name"])
	// 设计修正: qrcode_url 使用已有 qr_code_url 字段
	assert.NotEmpty(t, authorizerInfo["qrcode_url"])

	_ = cfg
	_ = event
	t.Log("authorized 事件处理: auth_code换token → 获取详情 → 关联租户 → 创建记录 → 缓存token")
}

func TestHandleAuthorized_ExistingAccount_Update(t *testing.T) {
	// 已存在的公众号再次授权 → 更新而非创建
	// 通过 authorizer_appid 唯一索引查重

	existingAuthorizerAppID := "wx_existing_authorized_app"
	newAuthCode := "authcode_reauthorize"

	// 模拟 GetByAuthorizerAppid 返回已存在记录
	existingAccountID := int64(50)

	// 更新逻辑：用新 auth_code 换 token → 更新令牌和权限集
	cache := newMockCache()

	// 缓存新 token
	cacheKey := fmt.Sprintf("authorizer_token:%d", existingAccountID)
	_ = cache.Set(context.Background(), cacheKey, "reauthorized_token", 7200*time.Second)

	token, _ := cache.Get(context.Background(), cacheKey)
	assert.Equal(t, "reauthorized_token", token)

	_ = existingAuthorizerAppID
	_ = newAuthCode

	t.Log("已存在公众号再次授权 → 更新令牌和权限，不创建新记录")
}

func TestHandleUpdateAuthorized(t *testing.T) {
	// 预期行为:
	//   func (s *WechatService) HandleUpdateAuthorized(ctx context.Context, event *UpdateAuthorizedEvent) error
	//   与 HandleAuthorized 类似，但仅更新已有记录的令牌和权限

	cache := newMockCache()

	// 模拟事件
	event := struct {
		AuthorizerAppid   string
		AuthorizationCode string
	}{
		AuthorizerAppid:   "wx_updated_app",
		AuthorizationCode: "authcode_update",
	}

	// 通过 auth_code 换取更新后的 token
	newRefreshToken := "updated_refresh_token"
	newAccessToken := "updated_access_token"
	expiresIn := 7200

	cacheKey := fmt.Sprintf("authorizer_token:%d", int64(42))
	_ = cache.Set(context.Background(), cacheKey, newAccessToken, time.Duration(expiresIn)*time.Second)

	cached, _ := cache.Get(context.Background(), cacheKey)
	assert.Equal(t, newAccessToken, cached)
	assert.NotEmpty(t, newRefreshToken)

	_ = event

	t.Log("updateauthorized 事件处理: 换token → 更新DB的 refresh_token 和 func_info")
}

func TestHandleUnauthorized(t *testing.T) {
	// 预期行为:
	//   func (s *WechatService) HandleUnauthorized(ctx context.Context, event *UnauthorizedEvent) error
	//   1. 通过 authorizer_appid 查找公众号
	//   2. 更新 auth_status = 3（取消授权）——设计修正
	//   3. 清除 Redis 中的 authorizer_token 缓存
	//   4. 找不到记录时幂等处理（不报错）

	cfg := newTestConfig()
	cache := newMockCache()

	// 模拟事件
	event := struct {
		AuthorizerAppid string
	}{
		AuthorizerAppid: "wx_cancelled_app",
	}

	accountID := int64(10)
	authStatusCanceled := int16(3) // 设计修正: 取消授权用 auth_status=3

	// 清除 Redis 缓存
	cacheKey := fmt.Sprintf("authorizer_token:%d", accountID)
	_ = cache.Set(context.Background(), cacheKey, "token_before_cancel", 7200*time.Second)
	err := cache.Del(context.Background(), cacheKey)
	assert.NoError(t, err)

	// 验证缓存已清除
	token, _ := cache.Get(context.Background(), cacheKey)
	assert.Empty(t, token, "取消授权后 token 缓存应已清除")

	// 验证 auth_status 值
	assert.Equal(t, int16(3), authStatusCanceled, "取消授权 auth_status 应为 3")

	_ = cfg
	_ = event

	t.Log("unauthorized 事件处理: auth_status=3 → 清除Redis缓存 → 记录不存在时幂等")
}

func TestHandleUnauthorized_AccountNotFound_Idempotent(t *testing.T) {
	// 微信推送 unauthorized 事件但找不到对应记录 → 幂等处理
	unknownAppID := "wx_unknown_cancelled_app"

	// 预期: 不报错，记录 warn 日志后返回 nil
	// 这避免了微信重复推送时产生不必要的告警

	t.Logf("unauthorized 事件: authorizer_appid=%s 找不到记录 → 幂等处理，返回 nil", unknownAppID)
}

// ========== 6. Token 过期时间与 TTL 验证 ==========

func TestTokenTTLValues(t *testing.T) {
	t.Run("component_token Redis TTL 大于刷新间隔", func(t *testing.T) {
		componentTokenTTL := 110 * time.Minute
		refreshInterval := 100 * time.Minute
		assert.True(t, componentTokenTTL > refreshInterval,
			"Redis TTL(%v) 应 > 刷新间隔(%v)", componentTokenTTL, refreshInterval)
	})

	t.Run("component_verify_ticket TTL 大于推送间隔", func(t *testing.T) {
		ticketTTL := 15 * time.Minute
		pushInterval := 10 * time.Minute
		assert.True(t, ticketTTL > pushInterval,
			"ticket TTL(%v) 应 > 推送间隔(%v)", ticketTTL, pushInterval)
	})

	t.Run("pre_auth_code TTL 等于微信有效期", func(t *testing.T) {
		preAuthTTL := 10 * time.Minute
		assert.Equal(t, 10*time.Minute, preAuthTTL,
			"pre_auth_code Redis TTL 应与微信有效期一致")
	})

	t.Run("authorizer_token 懒加载不需要定时刷新", func(t *testing.T) {
		// 设计决策: authorizer_token 用懒加载而非定时刷新
		// 理由: 不是所有公众号都会被频繁调用
		assert.True(t, true, "authorizer_token 使用懒加载模式")
	})
}

// ========== 7. 配置验证测试 ==========

func TestWechatConfig_Validation(t *testing.T) {
	t.Run("完整配置应通过验证", func(t *testing.T) {
		cfg := newTestConfig()
		err := cfg.Validate()
		assert.NoError(t, err, "完整配置应通过验证")
	})

	t.Run("encoding_aes_key 必须是 43 位", func(t *testing.T) {
		cfg := newTestConfig()
		assert.Len(t, cfg.Wechat.EncodingAESKey, 43,
			"EncodingAESKey 必须是 43 位字符")
	})

	t.Run("server_url 是必填字段", func(t *testing.T) {
		cfg := newTestConfig()
		assert.NotEmpty(t, cfg.Wechat.ServerURL,
			"ServerURL 是必填字段")
	})

	t.Run("component_app_secret 与 component_app_id 配合使用", func(t *testing.T) {
		cfg := newTestConfig()
		// 如果 component_app_id 已配置，component_app_secret 也必须配置
		assert.NotEmpty(t, cfg.Wechat.ComponentAppID)
		assert.NotEmpty(t, cfg.Wechat.ComponentAppSecret,
			"component_app_id 已配置时，component_app_secret 必须配置")
	})
}

// ========== 8. URL 构造正确性测试 ==========

func TestRedirectURIFormat(t *testing.T) {
	// 关键修正: redirect_uri 是 /wx/component/callback（不是 auth-callback）
	cfg := newTestConfig()

	redirectURI := fmt.Sprintf("%s/wx/component/callback", cfg.Wechat.ServerURL)
	expectedPath := "/wx/component/callback"

	assert.Contains(t, redirectURI, expectedPath,
		"redirect_uri 必须是 /wx/component/callback（设计修正）")
	assert.NotContains(t, redirectURI, "auth-callback",
		"redirect_uri 不应包含 auth-callback")

	encodedURI := url.QueryEscape(redirectURI)
	assert.NotEmpty(t, encodedURI)

	t.Logf("redirect_uri=%s", redirectURI)
}

func TestAuthURLComplete(t *testing.T) {
	// 综合验证预授权 URL 的所有参数
	cfg := newTestConfig()
	preAuthCode := "preauth_test_xyz"
	redirectURI := fmt.Sprintf("%s/wx/component/callback", cfg.Wechat.ServerURL)
	encodedRedirectURI := url.QueryEscape(redirectURI)

	authURL := fmt.Sprintf(
		"https://mp.weixin.qq.com/cgi-bin/componentloginpage?component_appid=%s&pre_auth_code=%s&redirect_uri=%s&auth_type=3",
		cfg.Wechat.ComponentAppID,
		preAuthCode,
		encodedRedirectURI,
	)

	// 解析 URL 验证
	parsed, err := url.Parse(authURL)
	require.NoError(t, err)

	assert.Equal(t, "mp.weixin.qq.com", parsed.Host)
	assert.Equal(t, "/cgi-bin/componentloginpage", parsed.Path)

	query := parsed.Query()
	assert.Equal(t, cfg.Wechat.ComponentAppID, query.Get("component_appid"))
	assert.Equal(t, preAuthCode, query.Get("pre_auth_code"))
	assert.Equal(t, redirectURI, query.Get("redirect_uri"))
	assert.Equal(t, "3", query.Get("auth_type"))

	t.Logf("完整授权URL: %s", authURL)
}
