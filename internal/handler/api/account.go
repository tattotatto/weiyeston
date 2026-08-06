// Package api 管理后台公众号管理 API Handler
package api

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/weiyeston/weiyeston-v2/internal/model"
	"github.com/weiyeston/weiyeston-v2/internal/repository/account"
)

// AccountCache 缓存接口（用于 AccountHandler 依赖注入和测试 mock）
type AccountCache interface {
	Del(ctx context.Context, keys ...string) error
}

// WechatService 微信服务接口（用于 AccountHandler 依赖注入和测试 mock）
type WechatService interface {
	GeneratePreAuthURL(ctx context.Context, tenantID int64) (string, error)
	FetchManualAccessToken(ctx context.Context, appID, appSecret string) (string, int, error)
}

// AccountHandler 公众号管理 API 处理器
type AccountHandler struct {
	accountRepo *account.Repo
	wechatSvc   WechatService
	cache       AccountCache
	logger      *zap.Logger
}

// NewAccountHandler 创建公众号管理 Handler
func NewAccountHandler(accountRepo *account.Repo, wechatSvc WechatService, cache AccountCache, logger *zap.Logger) *AccountHandler {
	return &AccountHandler{
		accountRepo: accountRepo,
		wechatSvc:   wechatSvc,
		cache:       cache,
		logger:      logger,
	}
}

// ========== 请求/响应结构体 ==========

// CreateAccountRequest 手动创建公众号请求体
type CreateAccountRequest struct {
	Name        string `json:"name"          binding:"required,min=1,max=100"`
	WxAppID     string `json:"wx_app_id"     binding:"required,min=1,max=50"`
	WxAppSecret string `json:"wx_app_secret" binding:"required,min=1,max=200"`
	WxOriginalID string `json:"wx_original_id" binding:"omitempty,max=50"`
	Description string `json:"description"   binding:"omitempty,max=500"`
	AvatarURL   string `json:"avatar_url"    binding:"omitempty,max=500"`
	QRCodeURL   string `json:"qr_code_url"   binding:"omitempty,max=500"`
}

// UpdateAccountRequest 编辑公众号请求体（所有字段可选）
type UpdateAccountRequest struct {
	Name         *string `json:"name"          binding:"omitempty,min=1,max=100"`
	WxAppID      *string `json:"wx_app_id"     binding:"omitempty,min=1,max=50"`
	WxAppSecret  *string `json:"wx_app_secret" binding:"omitempty,min=1,max=200"`
	WxOriginalID *string `json:"wx_original_id" binding:"omitempty,max=50"`
	Description  *string `json:"description"   binding:"omitempty,max=500"`
	AvatarURL    *string `json:"avatar_url"    binding:"omitempty,max=500"`
	QRCodeURL    *string `json:"qr_code_url"   binding:"omitempty,max=500"`
}

// ListAccountsRequest 分页列表查询参数
type ListAccountsRequest struct {
	Page       int    `form:"page,default=1"        binding:"omitempty,min=1"`
	PageSize   int    `form:"page_size,default=20"  binding:"omitempty,min=1,max=100"`
	Keyword    string `form:"keyword"`
	AuthType   int    `form:"auth_type"`
	AuthStatus int    `form:"auth_status,default=-1"`
}

// AccountVO 返回给前端的公众号视图对象（脱敏）
type AccountVO struct {
	ID            int64      `json:"id"`
	TenantID      int64      `json:"tenant_id"`
	Name          string     `json:"name"`
	WxAppID       string     `json:"wx_app_id"`
	WxOriginalID  string     `json:"wx_original_id,omitempty"`
	AuthType      int16      `json:"auth_type"`
	AuthStatus    int16      `json:"auth_status"`
	Description   string     `json:"description,omitempty"`
	AvatarURL     string     `json:"avatar_url,omitempty"`
	QRCodeURL     string     `json:"qr_code_url,omitempty"`
	FansCount     int        `json:"fans_count"`
	NickName      string     `json:"nick_name,omitempty"`
	HeadImg       string     `json:"head_img,omitempty"`
	PrincipalName string     `json:"principal_name,omitempty"`
	TokenExpireAt *time.Time `json:"token_expire_at,omitempty"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
}

// PaginatedResponse 分页响应
type PaginatedResponse struct {
	List     []AccountVO `json:"list"`
	Total    int64       `json:"total"`
	Page     int         `json:"page"`
	PageSize int         `json:"page_size"`
}

// ========== Handler 方法 ==========

// getTenantID 从上下文提取 tenant_id
func (h *AccountHandler) getTenantID(c *gin.Context) (int64, bool) {
	tenantIDVal, exists := c.Get("tenant_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"code": 401,
			"msg":  "未授权访问",
			"data": nil,
		})
		return 0, false
	}

	switch v := tenantIDVal.(type) {
	case int64:
		return v, true
	case float64:
		return int64(v), true
	default:
		c.JSON(http.StatusBadRequest, gin.H{
			"code": 40001,
			"msg":  "无效的租户信息",
			"data": nil,
		})
		return 0, false
	}
}

// requireOwnership 校验公众号是否属于当前租户
func (h *AccountHandler) requireOwnership(c *gin.Context, tenantID int64, accountID int64) bool {
	if tenantID != accountID {
		// This method signature differs from design doc - we check after fetching the account
		_ = tenantID
		return true
	}
	return true
}

// toAccountVO 将 model.WechatAccount 转为脱敏的 AccountVO
func toAccountVO(acc *model.WechatAccount) AccountVO {
	vo := AccountVO{
		ID:           acc.ID,
		TenantID:     acc.TenantID,
		WxAppID:      safeString(acc.WxAppID),
		AuthType:     acc.AuthType,
		AuthStatus:   acc.AuthStatus,
		FansCount:    acc.FansCount,
		CreatedAt:    acc.CreatedAt,
		UpdatedAt:    acc.UpdatedAt,
	}

	if acc.Name != nil {
		vo.Name = *acc.Name
	}
	if acc.WxOriginalID != nil {
		vo.WxOriginalID = *acc.WxOriginalID
	}
	if acc.Description != nil {
		vo.Description = *acc.Description
	}
	if acc.AvatarURL != nil {
		vo.AvatarURL = *acc.AvatarURL
	}
	if acc.QRCodeURL != nil {
		vo.QRCodeURL = *acc.QRCodeURL
	}
	if acc.NickName != nil {
		vo.NickName = *acc.NickName
	}
	if acc.HeadImg != nil {
		vo.HeadImg = *acc.HeadImg
	}
	if acc.PrincipalName != nil {
		vo.PrincipalName = *acc.PrincipalName
	}
	if acc.TokenExpireAt != nil {
		vo.TokenExpireAt = acc.TokenExpireAt
	}

	return vo
}

func safeString(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

// getPublicIPHint 获取服务器公网 IP 用于微信公众号 IP 白名单配置提示
// 使用 2 秒超时的 HTTP 请求，避免阻塞
func getPublicIPHint() string {
	client := &http.Client{Timeout: 2 * time.Second}
	resp, err := client.Get("https://api.ipify.org")
	if err != nil {
		return "请将服务器公网IP加入微信公众号IP白名单"
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 64))
	ip := strings.TrimSpace(string(body))
	if ip == "" {
		return "请将服务器公网IP加入微信公众号IP白名单"
	}
	return "请将以下IP加入微信公众号IP白名单: " + ip
}

// ========== CRUD Handlers ==========

// Create POST /api/v1/accounts — 手动创建公众号
func (h *AccountHandler) Create(c *gin.Context) {
	// 1. 参数校验
	var req CreateAccountRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code": 40001,
			"msg":  "参数错误: " + err.Error(),
			"data": nil,
		})
		return
	}

	// 验证 wx_app_id 格式（以 wx 开头）
	if !strings.HasPrefix(req.WxAppID, "wx") {
		c.JSON(http.StatusBadRequest, gin.H{
			"code": 40001,
			"msg":  "AppId 必须以 wx 开头",
			"data": nil,
		})
		return
	}

	// 2. 获取 tenant_id
	tenantID, ok := h.getTenantID(c)
	if !ok {
		return
	}

	// 3. AppId 唯一性检查
	existingAccount, err := h.accountRepo.GetByAppIDAndTenant(c.Request.Context(), req.WxAppID, tenantID)
	if err != nil {
		h.logger.Error("唯一性检查失败", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"code": 50001,
			"msg":  "服务器内部错误",
			"data": nil,
		})
		return
	}
	if existingAccount != nil {
		c.JSON(http.StatusConflict, gin.H{
			"code": 40901,
			"msg":  "该 AppId 已被绑定",
			"data": nil,
		})
		return
	}

	// 4. 验证 AppId + AppSecret 有效性
	accessToken, expiresIn, err := h.wechatSvc.FetchManualAccessToken(c.Request.Context(), req.WxAppID, req.WxAppSecret)
	if err != nil {
		h.logger.Error("微信验证失败", zap.Error(err))
		c.JSON(http.StatusBadGateway, gin.H{
			"code": 50201,
			"msg":  "微信验证失败: " + err.Error(),
			"data": nil,
		})
		return
	}

	// 5. 构建 WechatAccount 记录
	tokenExpireAt := time.Now().Add(time.Duration(expiresIn) * time.Second)
	name := req.Name
	wxAppID := req.WxAppID
	wxAppSecret := req.WxAppSecret
	authType := int16(1)   // 手动接入
	authStatus := int16(1) // 正常

	account := &model.WechatAccount{
		TenantID:     tenantID,
		Name:         &name,
		WxAppID:      &wxAppID,
		WxAppSecret:  &wxAppSecret,
		AuthType:     authType,
		AuthStatus:   authStatus,
		AccessToken:  &accessToken,
		TokenExpireAt: &tokenExpireAt,
	}

	if req.WxOriginalID != "" {
		oid := req.WxOriginalID
		account.WxOriginalID = &oid
	}
	if req.Description != "" {
		desc := req.Description
		account.Description = &desc
	}
	if req.AvatarURL != "" {
		avatar := req.AvatarURL
		account.AvatarURL = &avatar
	}
	if req.QRCodeURL != "" {
		qr := req.QRCodeURL
		account.QRCodeURL = &qr
	}

	// 6. 写入数据库
	if err := h.accountRepo.Create(c.Request.Context(), account); err != nil {
		h.logger.Error("创建公众号失败", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"code": 50001,
			"msg":  "创建公众号失败",
			"data": nil,
		})
		return
	}

	// 7. 返回创建结果（脱敏），包含 IP 白名单提示
	c.JSON(http.StatusCreated, gin.H{
		"code": 0,
		"msg":  "ok",
		"data": gin.H{
			"account":      toAccountVO(account),
			"ip_whitelist": getPublicIPHint(),
		},
	})
}

// List GET /api/v1/accounts — 公众号分页列表
func (h *AccountHandler) List(c *gin.Context) {
	// 1. 获取 tenant_id
	tenantID, ok := h.getTenantID(c)
	if !ok {
		return
	}

	// 2. 解析查询参数
	var req ListAccountsRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code": 40002,
			"msg":  "分页参数错误",
			"data": nil,
		})
		return
	}

	// 3. 调用 repository 分页查询
	params := account.ListParams{
		TenantID:   tenantID,
		Page:       req.Page,
		PageSize:   req.PageSize,
		Keyword:    req.Keyword,
		AuthType:   req.AuthType,
		AuthStatus: req.AuthStatus,
	}

	accounts, total, err := h.accountRepo.ListPaginated(c.Request.Context(), params)
	if err != nil {
		h.logger.Error("查询公众号列表失败", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"code": 50001,
			"msg":  "查询公众号列表失败",
			"data": nil,
		})
		return
	}

	// 4. 转换为 VO 列表
	list := make([]AccountVO, 0, len(accounts))
	for i := range accounts {
		list = append(list, toAccountVO(&accounts[i]))
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"msg":  "ok",
		"data": PaginatedResponse{
			List:     list,
			Total:    total,
			Page:     req.Page,
			PageSize: req.PageSize,
		},
	})
}

// GetByID GET /api/v1/accounts/:id — 公众号详情
func (h *AccountHandler) GetByID(c *gin.Context) {
	// 1. 解析 :id
	accountID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code": 40001,
			"msg":  "无效的公众号 ID",
			"data": nil,
		})
		return
	}

	// 2. 获取 tenant_id
	tenantID, ok := h.getTenantID(c)
	if !ok {
		return
	}

	// 3. 查询记录
	acc, err := h.accountRepo.GetByID(c.Request.Context(), accountID)
	if err != nil {
		h.logger.Error("查询公众号失败", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"code": 50001,
			"msg":  "查询公众号失败",
			"data": nil,
		})
		return
	}
	if acc == nil {
		c.JSON(http.StatusNotFound, gin.H{
			"code": 40401,
			"msg":  "公众号不存在",
			"data": nil,
		})
		return
	}

	// 4. 租户校验
	if acc.TenantID != tenantID {
		c.JSON(http.StatusForbidden, gin.H{
			"code": 40301,
			"msg":  "无权限操作该公众号",
			"data": nil,
		})
		return
	}

	// 5. 返回结果
	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"msg":  "ok",
		"data": toAccountVO(acc),
	})
}

// Update PUT /api/v1/accounts/:id — 编辑公众号
func (h *AccountHandler) Update(c *gin.Context) {
	// 1. 解析 :id
	accountID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code": 40001,
			"msg":  "无效的公众号 ID",
			"data": nil,
		})
		return
	}

	// 2. 获取 tenant_id
	tenantID, ok := h.getTenantID(c)
	if !ok {
		return
	}

	// 3. 查询现有记录
	acc, err := h.accountRepo.GetByID(c.Request.Context(), accountID)
	if err != nil {
		h.logger.Error("查询公众号失败", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"code": 50001,
			"msg":  "查询公众号失败",
			"data": nil,
		})
		return
	}
	if acc == nil {
		c.JSON(http.StatusNotFound, gin.H{
			"code": 40401,
			"msg":  "公众号不存在",
			"data": nil,
		})
		return
	}

	// 4. 租户校验
	if acc.TenantID != tenantID {
		c.JSON(http.StatusForbidden, gin.H{
			"code": 40301,
			"msg":  "无权限操作该公众号",
			"data": nil,
		})
		return
	}

	// 5. 解析请求体
	var req UpdateAccountRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code": 40001,
			"msg":  "参数错误: " + err.Error(),
			"data": nil,
		})
		return
	}

	// 6. 权限检查：平台授权账号（auth_type=2）不可编辑 secret 相关字段
	if acc.AuthType == 2 {
		if req.WxAppID != nil || req.WxAppSecret != nil {
			c.JSON(http.StatusForbidden, gin.H{
				"code": 40301,
				"msg":  "平台授权账号不允许编辑 AppId 和 AppSecret",
				"data": nil,
			})
			return
		}
	}

	// 7. 若传入 wx_app_id 且与现有值不同，则唯一性校验
	needTokenRefresh := false
	newAppID := safeString(acc.WxAppID)
	newSecret := safeString(acc.WxAppSecret)

	if req.WxAppID != nil && *req.WxAppID != newAppID {
		if !strings.HasPrefix(*req.WxAppID, "wx") {
			c.JSON(http.StatusBadRequest, gin.H{
				"code": 40001,
				"msg":  "AppId 必须以 wx 开头",
				"data": nil,
			})
			return
		}
		existing, err := h.accountRepo.GetByAppIDAndTenant(c.Request.Context(), *req.WxAppID, tenantID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"code": 50001,
				"msg":  "服务器内部错误",
				"data": nil,
			})
			return
		}
		if existing != nil && existing.ID != accountID {
			c.JSON(http.StatusConflict, gin.H{
				"code": 40901,
				"msg":  "该 AppId 已被其他公众号绑定",
				"data": nil,
			})
			return
		}
		needTokenRefresh = true
	}

	// 8. 若传入 wx_app_secret 且与现有值不同，或 AppId 已变更，需验证
	if req.WxAppSecret != nil && *req.WxAppSecret != newSecret {
		needTokenRefresh = true
	}

	if needTokenRefresh {
		verifyAppID := newAppID
		verifySecret := newSecret
		if req.WxAppID != nil {
			verifyAppID = *req.WxAppID
		}
		if req.WxAppSecret != nil {
			verifySecret = *req.WxAppSecret
		}

		accessToken, expiresIn, err := h.wechatSvc.FetchManualAccessToken(c.Request.Context(), verifyAppID, verifySecret)
		if err != nil {
			h.logger.Error("微信验证失败", zap.Error(err))
			c.JSON(http.StatusBadGateway, gin.H{
				"code": 50201,
				"msg":  "微信验证失败: " + err.Error(),
				"data": nil,
			})
			return
		}

		tokenExpireAt := time.Now().Add(time.Duration(expiresIn) * time.Second)
		acc.AccessToken = &accessToken
		acc.TokenExpireAt = &tokenExpireAt
	}

	// 9. 更新字段
	if req.Name != nil {
		acc.Name = req.Name
	}
	if req.WxAppID != nil {
		acc.WxAppID = req.WxAppID
	}
	if req.WxAppSecret != nil {
		acc.WxAppSecret = req.WxAppSecret
	}
	if req.WxOriginalID != nil {
		acc.WxOriginalID = req.WxOriginalID
	}
	if req.Description != nil {
		acc.Description = req.Description
	}
	if req.AvatarURL != nil {
		acc.AvatarURL = req.AvatarURL
	}
	if req.QRCodeURL != nil {
		acc.QRCodeURL = req.QRCodeURL
	}

	// 10. 执行更新
	if err := h.accountRepo.Update(c.Request.Context(), acc); err != nil {
		h.logger.Error("更新公众号失败", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"code": 50001,
			"msg":  "更新公众号失败",
			"data": nil,
		})
		return
	}

	// 11. 返回更新后的记录
	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"msg":  "ok",
		"data": toAccountVO(acc),
	})
}

// Delete DELETE /api/v1/accounts/:id — 软删除公众号
func (h *AccountHandler) Delete(c *gin.Context) {
	// 1. 解析 :id
	accountID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code": 40001,
			"msg":  "无效的公众号 ID",
			"data": nil,
		})
		return
	}

	// 2. 获取 tenant_id
	tenantID, ok := h.getTenantID(c)
	if !ok {
		return
	}

	// 3. 查询记录
	acc, err := h.accountRepo.GetByID(c.Request.Context(), accountID)
	if err != nil {
		h.logger.Error("查询公众号失败", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"code": 50001,
			"msg":  "查询公众号失败",
			"data": nil,
		})
		return
	}
	if acc == nil {
		c.JSON(http.StatusNotFound, gin.H{
			"code": 40401,
			"msg":  "公众号不存在",
			"data": nil,
		})
		return
	}

	// 4. 租户校验
	if acc.TenantID != tenantID {
		c.JSON(http.StatusForbidden, gin.H{
			"code": 40301,
			"msg":  "无权限操作该公众号",
			"data": nil,
		})
		return
	}

	// 5. 软删除
	deleted, err := h.accountRepo.SoftDeleteWithReturn(c.Request.Context(), accountID)
	if err != nil {
		h.logger.Error("软删除公众号失败", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"code": 50001,
			"msg":  "删除失败",
			"data": nil,
		})
		return
	}
	if !deleted {
		c.JSON(http.StatusNotFound, gin.H{
			"code": 40401,
			"msg":  "公众号不存在或已删除",
			"data": nil,
		})
		return
	}

	// 6. 清理 Redis 缓存
	if h.cache != nil {
		cacheKey := fmt.Sprintf("authorizer_token:%d", accountID)
		_ = h.cache.Del(c.Request.Context(), cacheKey)
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"msg":  "已删除",
		"data": nil,
	})
}

// ========== T3 已有方法 ==========

// GenerateAuthURL POST /api/v1/accounts/auth-url
// 生成微信第三方平台预授权 URL
// 需要 JWT 认证（由中间件保证 tenant_id 在上下文中）
func (h *AccountHandler) GenerateAuthURL(c *gin.Context) {
	tenantID, ok := h.getTenantID(c)
	if !ok {
		return
	}

	// 调用微信服务生成预授权 URL
	authURL, err := h.wechatSvc.GeneratePreAuthURL(c.Request.Context(), tenantID)
	if err != nil {
		h.logger.Error("生成预授权 URL 失败", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"code": 50001,
			"msg":  "生成授权链接失败: " + err.Error(),
			"data": nil,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"msg":  "ok",
		"data": gin.H{
			"auth_url":   authURL,
			"expires_in": 600,
		},
	})
}

// GetAuthStatus GET /api/v1/accounts/:id/auth-status
// 查询公众号授权状态（前端轮询用）
// 需要 JWT 认证
func (h *AccountHandler) GetAuthStatus(c *gin.Context) {
	accountIDStr := c.Param("id")
	accountID, err := strconv.ParseInt(accountIDStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code": 40001,
			"msg":  "无效的公众号 ID",
			"data": nil,
		})
		return
	}

	// 获取 tenant_id
	tenantID, ok := h.getTenantID(c)
	if !ok {
		return
	}

	// 查询公众号信息
	acc, err := h.accountRepo.GetByID(c.Request.Context(), accountID)
	if err != nil {
		h.logger.Error("查询公众号失败", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"code": 50001,
			"msg":  "查询公众号失败",
			"data": nil,
		})
		return
	}
	if acc == nil {
		c.JSON(http.StatusNotFound, gin.H{
			"code": 40401,
			"msg":  "公众号不存在",
			"data": nil,
		})
		return
	}

	// 验证公众号属于当前租户
	if acc.TenantID != tenantID {
		c.JSON(http.StatusForbidden, gin.H{
			"code": 40301,
			"msg":  "无权限访问该公众号",
			"data": nil,
		})
		return
	}

	// 构造响应
	respData := gin.H{
		"account_id":  acc.ID,
		"auth_status": acc.AuthStatus,
		"auth_type":   acc.AuthType,
	}

	if acc.NickName != nil {
		respData["nick_name"] = *acc.NickName
	}
	if acc.HeadImg != nil {
		respData["head_img"] = *acc.HeadImg
	}
	if acc.FuncInfo != nil {
		// 解析 JSON 为数组返回
		var funcInfoArr []json.RawMessage
		if err := json.Unmarshal(*acc.FuncInfo, &funcInfoArr); err == nil {
			respData["func_info"] = funcInfoArr
		} else {
			// 尝试解析为单对象再包裹
			respData["func_info"] = acc.FuncInfo
		}
	}
	if acc.CreatedAt.IsZero() == false {
		respData["authorized_at"] = acc.CreatedAt.Format("2006-01-02T15:04:05Z")
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"msg":  "ok",
		"data": respData,
	})
}
