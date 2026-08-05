// Package wechat 微信第三方平台服务
// 核心功能：授权流程、Token 刷新架构、事件处理
package wechat

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"sync"
	"time"

	"github.com/jmoiron/sqlx"
	"go.uber.org/zap"

	"github.com/weiyeston/weiyeston-v2/internal/cache"
	"github.com/weiyeston/weiyeston-v2/internal/config"
	"github.com/weiyeston/weiyeston-v2/internal/model"
	"github.com/weiyeston/weiyeston-v2/internal/repository/account"
	"github.com/weiyeston/weiyeston-v2/internal/repository/ticket"
)

// WechatService 微信第三方平台核心服务
type WechatService struct {
	config      *config.Config
	cache       *cache.Client
	db          *sqlx.DB
	logger      *zap.Logger
	accountRepo *account.Repo
	ticketRepo  *ticket.Repo

	// componentAccessToken 缓存保护（避免并发刷新）
	mu             sync.RWMutex
	refreshing     bool
	tokenRefresher context.CancelFunc
}

// NewWechatService 初始化微信服务
func NewWechatService(
	cfg *config.Config,
	cacheClient *cache.Client,
	db *sqlx.DB,
	logger *zap.Logger,
	accountRepo *account.Repo,
	ticketRepo *ticket.Repo,
) *WechatService {
	return &WechatService{
		config:      cfg,
		cache:       cacheClient,
		db:          db,
		logger:      logger,
		accountRepo: accountRepo,
		ticketRepo:  ticketRepo,
	}
}

// ========== 预授权 URL 生成 ==========

// GeneratePreAuthURL 生成预授权 URL
// tenantID: 租户 ID，用于授权回调时确定关联哪个租户
func (s *WechatService) GeneratePreAuthURL(ctx context.Context, tenantID int64) (string, error) {
	// 1. 获取 component_access_token（校验可用性）
	token, err := s.GetComponentAccessToken(ctx)
	if err != nil {
		return "", fmt.Errorf("获取 component_access_token 失败: %w", err)
	}
	if token == "" {
		return "", fmt.Errorf("component_access_token 为空，请检查微信第三方平台配置")
	}

	// 2. 生成预授权码（本地生成，实际应由 SDK GetPreAuthCode 获取）
	// 此处 SDK 调用: preAuthCode, err := s.sdk.GetPreAuthCode()
	preAuthCode := fmt.Sprintf("preauth_%d_%d", tenantID, time.Now().UnixNano())

	// 3. 将 (pre_auth_code, tenant_id) 关联存入 Redis, TTL 10 分钟
	authKey := fmt.Sprintf("pre_auth:%s", preAuthCode)
	err = s.cache.Set(ctx, authKey, strconv.FormatInt(tenantID, 10), 10*time.Minute)
	if err != nil {
		return "", fmt.Errorf("缓存预授权关联失败: %w", err)
	}

	// 4. 构造授权 URL
	redirectURI := fmt.Sprintf("%s/wx/component/callback", s.config.Wechat.ServerURL)
	authURL := fmt.Sprintf(
		"https://mp.weixin.qq.com/cgi-bin/componentloginpage?component_appid=%s&pre_auth_code=%s&redirect_uri=%s&auth_type=3",
		s.config.Wechat.ComponentAppID,
		preAuthCode,
		url.QueryEscape(redirectURI),
	)

	s.logger.Info("生成预授权 URL",
		zap.Int64("tenantID", tenantID),
		zap.String("preAuthCode", preAuthCode))

	return authURL, nil
}

// ========== component_access_token 管理 ==========

// GetComponentAccessToken 获取当前有效的 component_access_token
// 优先从 Redis 读取，key 不存在时触发同步刷新
func (s *WechatService) GetComponentAccessToken(ctx context.Context) (string, error) {
	tokenKey := fmt.Sprintf("component_access_token:%s", s.config.Wechat.ComponentAppID)
	token, err := s.cache.Get(ctx, tokenKey)
	if err != nil {
		return "", err
	}
	if token != "" {
		return token, nil
	}

	// 缓存未命中，同步刷新
	s.refreshComponentToken(ctx)
	token, err = s.cache.Get(ctx, tokenKey)
	return token, err
}

// GetComponentVerifyTicket 获取 component_verify_ticket（用于自检）
func (s *WechatService) GetComponentVerifyTicket(ctx context.Context) (string, error) {
	ticketKey := fmt.Sprintf("component_verify_ticket:%s", s.config.Wechat.ComponentAppID)
	ticket, err := s.cache.Get(ctx, ticketKey)
	if err != nil {
		return "", err
	}
	return ticket, nil
}

// StartComponentTokenRefresher 启动 component_access_token 定时刷新 goroutine
func (s *WechatService) StartComponentTokenRefresher(ctx context.Context) {
	refreshInterval := 100 * time.Minute

	// 启动时立即刷新一次
	s.refreshComponentToken(ctx)

	refreshCtx, cancel := context.WithCancel(ctx)
	s.tokenRefresher = cancel

	go func() {
		ticker := time.NewTicker(refreshInterval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				s.refreshComponentToken(refreshCtx)
			case <-refreshCtx.Done():
				s.logger.Info("component token refresher stopped")
				return
			}
		}
	}()

	s.logger.Info("component token refresher started",
		zap.Duration("interval", refreshInterval))
}

// StopComponentTokenRefresher 停止后台 refresher（优雅关闭）
func (s *WechatService) StopComponentTokenRefresher() {
	if s.tokenRefresher != nil {
		s.tokenRefresher()
	}
}

// refreshComponentToken 刷新 component_access_token
// Redis 主存储 + DB（ticket）兜底
func (s *WechatService) refreshComponentToken(ctx context.Context) {
	ticketKey := fmt.Sprintf("component_verify_ticket:%s", s.config.Wechat.ComponentAppID)
	tokenKey := fmt.Sprintf("component_access_token:%s", s.config.Wechat.ComponentAppID)

	// 1. 从 Redis 读取 component_verify_ticket
	ticket, err := s.cache.Get(ctx, ticketKey)
	if err != nil || ticket == "" {
		// 从数据库兜底读取最近一条 ticket
		dbTicket, dbErr := s.ticketRepo.GetLatest(ctx, s.config.Wechat.ComponentAppID)
		if dbErr != nil || dbTicket == "" {
			s.logger.Error("无法获取 component_verify_ticket，component_access_token 将无法刷新",
				zap.Error(err), zap.Error(dbErr))
			return
		}
		ticket = dbTicket
		s.logger.Info("从 DB 兜底恢复 component_verify_ticket")
	}

	// 2. 调用 SDK 刷新 token
	// SDK 调用: s.sdk.GetComponentAccessToken()
	// 此处模拟生成新 token（实际由 SDK 处理 ticket 后调用微信 API）
	newToken := fmt.Sprintf("component_token_%d", time.Now().Unix())

	// 3. 缓存到 Redis, TTL 110 分钟
	err = s.cache.Set(ctx, tokenKey, newToken, 110*time.Minute)
	if err != nil {
		s.logger.Error("缓存 component_access_token 失败", zap.Error(err))
		return
	}

	s.logger.Info("component_access_token 刷新成功",
		zap.Int("token_len", len(newToken)))
}

// ========== authorizer_access_token 懒加载 ==========

// GetAuthorizerAccessToken 获取授权方的 access_token（懒加载 + 自动刷新）
func (s *WechatService) GetAuthorizerAccessToken(ctx context.Context, accountID int64) (string, error) {
	// 1. 从 Redis 尝试读取缓存
	cacheKey := fmt.Sprintf("authorizer_token:%d", accountID)
	token, _ := s.cache.Get(ctx, cacheKey)
	if token != "" {
		return token, nil
	}

	// 2. Redis 未命中 → 从 DB 读取
	account, err := s.accountRepo.GetByID(ctx, accountID)
	if err != nil {
		return "", fmt.Errorf("查询公众号失败: %w", err)
	}
	if account == nil {
		return "", fmt.Errorf("公众号不存在: id=%d", accountID)
	}

	// 3. 判断 token 是否过期
	if account.AccessToken != nil && account.TokenExpireAt != nil && time.Now().Before(*account.TokenExpireAt) {
		// Token 未过期，直接使用并回填缓存
		ttl := time.Until(*account.TokenExpireAt)
		_ = s.cache.Set(ctx, cacheKey, *account.AccessToken, ttl)
		return *account.AccessToken, nil
	}

	// 4. Token 过期或不存在 → 使用 refresh_token 刷新
	return s.refreshAuthorizerToken(ctx, account)
}

// refreshAuthorizerToken 刷新授权方 access_token
func (s *WechatService) refreshAuthorizerToken(ctx context.Context, account *model.WechatAccount) (string, error) {
	if account.RefreshToken == nil || *account.RefreshToken == "" {
		return "", fmt.Errorf("公众号 %d 没有 refresh_token，请重新授权", account.ID)
	}

	// 1. 获取 component_access_token
	componentToken, err := s.GetComponentAccessToken(ctx)
	if err != nil {
		return "", fmt.Errorf("获取 component_access_token 失败: %w", err)
	}

	// 2. SDK 调用: s.sdk.RefreshAuthorizerToken(authorizerAppid, authorizerRefreshToken)
	// 模拟结果
	_ = componentToken
	newAccessToken := fmt.Sprintf("authorizer_token_%d_%d", account.ID, time.Now().Unix())
	newRefreshToken := fmt.Sprintf("refresh_token_%d_%d", account.ID, time.Now().Unix())
	expiresIn := 7200

	// 3. 更新 DB
	expireAt := time.Now().Add(time.Duration(expiresIn) * time.Second)
	err = s.accountRepo.UpdateToken(ctx, account.ID,
		newAccessToken,
		newRefreshToken,
		expireAt,
	)
	if err != nil {
		// 刷新失败时更新 auth_status 为过期（2 = 令牌过期）
		_ = s.accountRepo.UpdateAuthStatus(ctx, account.ID, 2)
		return "", fmt.Errorf("更新 token 失败: %w", err)
	}

	// 4. 更新 Redis 缓存
	cacheKey := fmt.Sprintf("authorizer_token:%d", account.ID)
	_ = s.cache.Set(ctx, cacheKey, newAccessToken,
		time.Duration(expiresIn)*time.Second)

	return newAccessToken, nil
}

// ========== 事件处理 ==========

// HandleComponentVerifyTicket 处理 component_verify_ticket 推送
func (s *WechatService) HandleComponentVerifyTicket(ctx context.Context, ticket string) error {
	// 1. 缓存到 Redis（主存储，供 goroutine 读取）
	ticketKey := fmt.Sprintf("component_verify_ticket:%s", s.config.Wechat.ComponentAppID)
	err := s.cache.Set(ctx, ticketKey, ticket, 15*time.Minute)
	if err != nil {
		s.logger.Error("缓存 component_verify_ticket 到 Redis 失败", zap.Error(err))
	}

	// 2. 持久化到数据库（兜底恢复）
	_ = s.ticketRepo.Save(ctx, s.config.Wechat.ComponentAppID, ticket)

	// 3. 立即触发 component_access_token 刷新
	s.refreshComponentToken(ctx)

	s.logger.Info("component_verify_ticket 已接收并处理")
	return nil
}

// HandleAuthorized 处理授权成功事件
func (s *WechatService) HandleAuthorized(ctx context.Context, authorizerAppID, authCode, preAuthCode string) error {
	// 1. 通过 auth_code 换取 authorizer 令牌（SDK: PostAuthorizationInfo）
	// SDK 调用: authInfo, err := s.sdk.PostAuthorizationInfo(authCode)
	type AuthInfo struct {
		AuthorizerAccessToken  string           `json:"authorizer_access_token"`
		ExpiresIn              int              `json:"expires_in"`
		AuthorizerRefreshToken string           `json:"authorizer_refresh_token"`
		FuncInfo               *json.RawMessage `json:"func_info"` // 设计修正: *json.RawMessage
	}

	// 模拟 SDK 返回
	rawFuncInfo := json.RawMessage(`[{"funcscope_category":{"id":1}},{"funcscope_category":{"id":15}}]`)
	authInfo := AuthInfo{
		AuthorizerAccessToken:  fmt.Sprintf("auth_token_%s_%d", authorizerAppID, time.Now().Unix()),
		ExpiresIn:              7200,
		AuthorizerRefreshToken: fmt.Sprintf("refresh_%s_%d", authorizerAppID, time.Now().Unix()),
		FuncInfo:               &rawFuncInfo,
	}

	// 2. 获取授权公众号的详细信息（SDK: GetAuthorizerInfo）
	// SDK 调用: authorizerInfo, err := s.sdk.GetAuthorizerInfo(authorizerAppID)

	// 3. 从 Redis 查找 pre_auth_code 关联的 tenant_id
	tenantID, err := s.resolveTenantFromPreAuth(ctx, preAuthCode)
	if err != nil {
		s.logger.Error("无法解析预授权码关联的租户",
			zap.String("preAuthCode", preAuthCode),
			zap.Error(err))
		return err
	}

	// 4. 构建公众号记录
	expireAt := time.Now().Add(time.Duration(authInfo.ExpiresIn) * time.Second)
	nickName := "授权公众号_" + authorizerAppID[:12]
	headImg := "http://wx.qlogo.cn/mmhead/default"
	userName := "gh_default"
	alias := "default_alias"
	principalName := "默认主体"
	qrcodeUrl := "http://mmbiz.qpic.cn/qrcode/default"
	signature := "功能介绍"
	svcTypeInfo := int16(2)
	verifyTypeInfo := int16(0)

	// 检查是否已有记录（通过 authorizer_appid 查重）
	existing, _ := s.accountRepo.GetByAuthorizerAppid(ctx, authorizerAppID)
	if existing != nil {
		// 更新已有记录
		existing.RefreshToken = &authInfo.AuthorizerRefreshToken
		existing.AccessToken = &authInfo.AuthorizerAccessToken
		existing.TokenExpireAt = &expireAt
		existing.FuncInfo = authInfo.FuncInfo
		existing.AuthStatus = 1
		err = s.accountRepo.Update(ctx, existing)
		if err != nil {
			return fmt.Errorf("更新授权记录失败: %w", err)
		}
	} else {
		// 创建新记录
		account := &model.WechatAccount{
			TenantID:        tenantID,
			Name:            &nickName,
			WxAppID:         &authorizerAppID,
			AuthType:        2, // 平台授权
			AuthStatus:      1, // 已接入
			AuthorizerAppid: &authorizerAppID,
			FuncInfo:        authInfo.FuncInfo,
			RefreshToken:    &authInfo.AuthorizerRefreshToken,
			AccessToken:     &authInfo.AuthorizerAccessToken,
			TokenExpireAt:   &expireAt,
			NickName:        &nickName,
			HeadImg:         &headImg,
			UserName:        &userName,
			Alias:           &alias,
			PrincipalName:   &principalName,
			QrcodeUrl:       &qrcodeUrl,
			Signature:       &signature,
			ServiceTypeInfo: &svcTypeInfo,
			VerifyTypeInfo:  &verifyTypeInfo,
		}
		err = s.accountRepo.Create(ctx, account)
		if err != nil {
			return fmt.Errorf("创建授权记录失败: %w", err)
		}
	}

	// 5. 缓存 authorizer_access_token 到 Redis
	// account.ID 在 Create/Get 后已设置
	if existing != nil {
		cacheKey := fmt.Sprintf("authorizer_token:%d", existing.ID)
		_ = s.cache.Set(ctx, cacheKey, authInfo.AuthorizerAccessToken,
			time.Duration(authInfo.ExpiresIn)*time.Second)
	}

	s.logger.Info("公众号授权成功",
		zap.String("authorizerAppid", authorizerAppID))

	return nil
}

// HandleUpdateAuthorized 处理授权更新事件
func (s *WechatService) HandleUpdateAuthorized(ctx context.Context, authorizerAppID, authCode, preAuthCode string) error {
	// SDK 调用: authInfo, err := s.sdk.PostAuthorizationInfo(authCode)
	type AuthInfo struct {
		AuthorizerAccessToken  string           `json:"authorizer_access_token"`
		ExpiresIn              int              `json:"expires_in"`
		AuthorizerRefreshToken string           `json:"authorizer_refresh_token"`
		FuncInfo               *json.RawMessage `json:"func_info"`
	}

	// 模拟 SDK 返回
	rawFuncInfo := json.RawMessage(`[{"funcscope_category":{"id":1}}]`)
	authInfo := AuthInfo{
		AuthorizerAccessToken:  fmt.Sprintf("updated_token_%s_%d", authorizerAppID, time.Now().Unix()),
		ExpiresIn:              7200,
		AuthorizerRefreshToken: fmt.Sprintf("updated_refresh_%s_%d", authorizerAppID, time.Now().Unix()),
		FuncInfo:               &rawFuncInfo,
	}

	// 查找已有记录
	account, err := s.accountRepo.GetByAuthorizerAppid(ctx, authorizerAppID)
	if err != nil || account == nil {
		return fmt.Errorf("找不到被更新的授权记录: %w", err)
	}

	// 更新令牌和权限集
	expireAt := time.Now().Add(time.Duration(authInfo.ExpiresIn) * time.Second)
	err = s.accountRepo.UpdateToken(ctx, account.ID,
		authInfo.AuthorizerAccessToken,
		authInfo.AuthorizerRefreshToken,
		expireAt,
	)
	if err != nil {
		return fmt.Errorf("更新授权令牌失败: %w", err)
	}

	// 更新 func_info
	if authInfo.FuncInfo != nil {
		account.FuncInfo = authInfo.FuncInfo
		_ = s.accountRepo.Update(ctx, account)
	}

	s.logger.Info("授权更新成功",
		zap.Int64("accountID", account.ID),
		zap.String("authorizerAppid", authorizerAppID))

	_ = authCode
	_ = preAuthCode

	return nil
}

// HandleUnauthorized 处理取消授权事件
func (s *WechatService) HandleUnauthorized(ctx context.Context, authorizerAppID string) error {
	// 设计修正: 取消授权用 auth_status = 3
	account, err := s.accountRepo.GetByAuthorizerAppid(ctx, authorizerAppID)
	if err != nil || account == nil {
		s.logger.Warn("取消授权但找不到对应记录",
			zap.String("authorizerAppid", authorizerAppID))
		return nil // 幂等处理
	}

	// 更新状态为 auth_status = 3（取消授权）
	_ = s.accountRepo.UpdateAuthStatus(ctx, account.ID, 3)

	// 清除 Redis 中的 authorizer_token 缓存
	cacheKey := fmt.Sprintf("authorizer_token:%d", account.ID)
	_ = s.cache.Del(ctx, cacheKey)

	s.logger.Info("公众号已取消授权",
		zap.Int64("accountID", account.ID),
		zap.String("authorizerAppid", authorizerAppID))

	return nil
}

// ========== 辅助方法 ==========

// resolveTenantFromPreAuth 从 Redis 查找 pre_auth_code 关联的 tenant_id
func (s *WechatService) resolveTenantFromPreAuth(ctx context.Context, preAuthCode string) (int64, error) {
	authKey := fmt.Sprintf("pre_auth:%s", preAuthCode)
	stored, err := s.cache.Get(ctx, authKey)
	if err != nil {
		return 0, fmt.Errorf("查询预授权关联失败: %w", err)
	}
	if stored == "" {
		return 0, fmt.Errorf("预授权码已过期或无效: %s", preAuthCode)
	}
	tenantID, err := strconv.ParseInt(stored, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("预授权关联数据异常: %w", err)
	}
	return tenantID, nil
}


	// ========== 手动接入 access_token 管理 ==========

// WechatTokenResponse 微信 access_token 接口响应
type WechatTokenResponse struct {
	AccessToken string `json:"access_token"`
	ExpiresIn   int    `json:"expires_in"`
	ErrCode     int    `json:"errcode"`
	ErrMsg      string `json:"errmsg"`
}

// FetchManualAccessToken 手动接入：验证 AppId+AppSecret 获取 access_token
// 直接调用微信基础 API（不需要 component_access_token）
func (s *WechatService) FetchManualAccessToken(ctx context.Context, appID, appSecret string) (string, int, error) {
	url := fmt.Sprintf(
		"https://api.weixin.qq.com/cgi-bin/token?grant_type=client_credential&appid=%s&secret=%s",
		appID, appSecret,
	)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", 0, fmt.Errorf("构建请求失败: %w", err)
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", 0, fmt.Errorf("微信 API 请求失败: %w", err)
	}
	defer resp.Body.Close()

	var result WechatTokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", 0, fmt.Errorf("解析微信响应失败: %w", err)
	}

	if result.ErrCode != 0 {
		return "", 0, fmt.Errorf("微信返回错误: %s (errcode=%d)", result.ErrMsg, result.ErrCode)
	}

	return result.AccessToken, result.ExpiresIn, nil
}

// ========== Handler 辅助 ==========

// ParseComponentMessage 解析微信开放平台推送的加密消息
// 返回 InfoType 和原始解密后的 XML body
func (s *WechatService) ParseComponentMessage(body []byte, timestamp, nonce, msgSignature string) (infoType string, plainXML []byte, err error) {
	// 实际应由 SDK 中间件处理加解密
	// SDK 使用: s.sdk.ServeHTTP(c.Writer, c.Request)
	// 此处简化为由 handler 层直接处理
	_ = timestamp
	_ = nonce
	_ = msgSignature
	_ = body
	_ = io.Discard

	return "", nil, fmt.Errorf("消息解析需由 handler 层通过 SDK 中间件处理")
}
