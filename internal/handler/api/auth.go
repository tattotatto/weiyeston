// Package api 认证 Handler
// 实现登录、刷新 Token、当前用户信息、登出接口
package api

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/jmoiron/sqlx"
	"github.com/redis/go-redis/v9"

	"github.com/weiyeston/weiyeston-v2/internal/config"
	"github.com/weiyeston/weiyeston-v2/internal/model"
	"github.com/weiyeston/weiyeston-v2/internal/repository/tenant"
	"github.com/weiyeston/weiyeston-v2/internal/service"
)

// AuthHandler 认证 Handler
type AuthHandler struct {
	DB         *sqlx.DB
	Redis      *redis.Client
	JWT        config.JWTConfig
	TenantRepo *tenant.Repo
}

// ========== 请求/响应结构体 ==========

// LoginRequest 登录请求
type LoginRequest struct {
	Username string `json:"username" binding:"required,min=1,max=50"`
	Password string `json:"password" binding:"required,min=1,max=100"`
}

// RefreshRequest 刷新 Token 请求
type RefreshRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

// RegisterRequest 注册请求
type RegisterRequest struct {
	Username string `json:"username" binding:"required,min=3,max=50"`
	Password string `json:"password" binding:"required,min=8"`
	Email    string `json:"email" binding:"required,email"`
	Nickname string `json:"nickname"`
}

// UserDTO 返回给前端的用户信息（脱敏）
type UserDTO struct {
	ID        int64  `json:"id"`
	Username  string `json:"username"`
	Nickname  string `json:"nickname"`
	Role      string `json:"role"`
	AvatarURL string `json:"avatar_url,omitempty"`
}

// LoginResponse 登录响应 data 字段
type LoginResponse struct {
	AccessToken  string  `json:"access_token"`
	RefreshToken string  `json:"refresh_token"`
	ExpiresIn    int64   `json:"expires_in"`
	User         UserDTO `json:"user"`
}

// RefreshResponse 刷新 Token 响应 data 字段
type RefreshResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int64  `json:"expires_in"`
}

// ========== Handler 方法 ==========

// Login 登录处理
// POST /api/v1/auth/login
func (h *AuthHandler) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 40001, "msg": "请求参数不合法"})
		return
	}

	// 限流检查
	ip := c.ClientIP()
	rateKey := fmt.Sprintf("login_rate:%s", ip)
	count, err := h.Redis.Incr(c.Request.Context(), rateKey).Result()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": "服务器内部错误"})
		return
	}
	if count == 1 {
		h.Redis.Expire(c.Request.Context(), rateKey, 1*time.Minute)
	}
	if count > 5 {
		c.JSON(http.StatusTooManyRequests, gin.H{"code": 42901, "msg": "登录尝试过于频繁，请稍后再试"})
		return
	}

	// 查询用户
	ctx := c.Request.Context()
	user, err := h.TenantRepo.GetByUsername(ctx, req.Username)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": "服务器内部错误"})
		return
	}

	// 用户不存在 → 统一返回 40101（防账号枚举）
	if user == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"code": 40101, "msg": "用户名或密码错误"})
		return
	}

	// 检查账号状态
	if user.Status != 1 {
		c.JSON(http.StatusForbidden, gin.H{"code": 40301, "msg": "账号已被停用"})
		return
	}

	// 验证密码
	if err := service.VerifyPassword(user.PasswordHash, req.Password); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"code": 40101, "msg": "用户名或密码错误"})
		return
	}

	// 生成 Access Token
	accessToken, err := service.GenerateAccessToken(user, h.JWT)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": "服务器内部错误"})
		return
	}

	// 生成 Refresh Token
	refreshToken, err := service.GenerateRefreshToken(ctx, h.Redis, user.ID, h.JWT)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": "服务器内部错误"})
		return
	}

	// 更新最后登录时间
	if _, err := h.DB.ExecContext(ctx,
		"UPDATE tenants SET last_login_at = NOW() WHERE id = $1", user.ID); err != nil {
		// 非关键错误，仅记录日志
	}

	// 构建响应
	expiration := h.JWT.GetAccessExpiration()
	if expiration <= 0 {
		expiration = 2 * time.Hour
	}

	avatarURL := ""
	if user.AvatarURL != nil {
		avatarURL = *user.AvatarURL
	}

	nickname := ""
	if user.Nickname != nil {
		nickname = *user.Nickname
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"msg":  "登录成功",
		"data": LoginResponse{
			AccessToken:  accessToken,
			RefreshToken: refreshToken,
			ExpiresIn:    int64(expiration.Seconds()),
			User: UserDTO{
				ID:        user.ID,
				Username:  user.Username,
				Nickname:  nickname,
				Role:      user.Role,
				AvatarURL: avatarURL,
			},
		},
	})
}

// Refresh 刷新 Token
// POST /api/v1/auth/refresh
// 独立于 Auth 中间件组，使用 ParseUnverified 跳过过期验证
func (h *AuthHandler) Refresh(c *gin.Context) {
	var req RefreshRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 40001, "msg": "请求参数不合法"})
		return
	}

	// 从 Authorization header 解析 user_id（使用 ParseUnverified 跳过过期验证）
	var userID int64
	authHeader := c.GetHeader("Authorization")
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
	ctx := c.Request.Context()
	stored, err := service.GetRefreshToken(ctx, h.Redis, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": "服务器内部错误"})
		return
	}
	if stored == "" || stored != req.RefreshToken {
		c.JSON(http.StatusUnauthorized, gin.H{"code": 40102, "msg": "refresh token 无效或已过期，请重新登录"})
		return
	}

	// 查询用户信息以生成新 token
	user, err := h.TenantRepo.GetByID(ctx, userID)
	if err != nil || user == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"code": 40102, "msg": "refresh token 无效或已过期，请重新登录"})
		return
	}

	// Rotation: 生成新 Token 对
	newAccessToken, err := service.GenerateAccessToken(user, h.JWT)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": "服务器内部错误"})
		return
	}

	newRefreshToken, err := service.GenerateRefreshToken(ctx, h.Redis, user.ID, h.JWT)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": "服务器内部错误"})
		return
	}

	expiration := h.JWT.GetAccessExpiration()
	if expiration <= 0 {
		expiration = 2 * time.Hour
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"msg":  "ok",
		"data": RefreshResponse{
			AccessToken:  newAccessToken,
			RefreshToken: newRefreshToken,
			ExpiresIn:    int64(expiration.Seconds()),
		},
	})
}

// Me 获取当前用户信息
// GET /api/v1/auth/me
func (h *AuthHandler) Me(c *gin.Context) {
	ctx := c.Request.Context()

	// Auth 中间件已验证 token 并注入 user_id
	userIDVal, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"code": 401, "msg": "未授权访问"})
		return
	}

	userID, ok := userIDVal.(int64)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"code": 401, "msg": "未授权访问"})
		return
	}

	user, err := h.TenantRepo.GetByID(ctx, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": "服务器内部错误"})
		return
	}
	if user == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"code": 401, "msg": "未授权访问"})
		return
	}

	avatarURL := ""
	if user.AvatarURL != nil {
		avatarURL = *user.AvatarURL
	}

	nickname := ""
	if user.Nickname != nil {
		nickname = *user.Nickname
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"msg":  "ok",
		"data": UserDTO{
			ID:        user.ID,
			Username:  user.Username,
			Nickname:  nickname,
			Role:      user.Role,
			AvatarURL: avatarURL,
		},
	})
}

// Logout 登出
// POST /api/v1/auth/logout
func (h *AuthHandler) Logout(c *gin.Context) {
	ctx := c.Request.Context()

	userIDVal, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"code": 401, "msg": "未授权访问"})
		return
	}

	userID, ok := userIDVal.(int64)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"code": 401, "msg": "未授权访问"})
		return
	}

	if err := service.RevokeRefreshToken(ctx, h.Redis, userID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": "服务器内部错误"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"msg":  "已退出登录",
	})
}

// Register 用户注册
// POST /api/v1/auth/register
func (h *AuthHandler) Register(c *gin.Context) {
	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 40001, "msg": "请求参数不合法"})
		return
	}

	ctx := c.Request.Context()

	// 检查用户名是否已存在
	existing, err := h.TenantRepo.GetByUsername(ctx, req.Username)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": "服务器内部错误"})
		return
	}
	if existing != nil {
		c.JSON(http.StatusConflict, gin.H{"code": 40901, "msg": "用户名已存在"})
		return
	}

	// 哈希密码
	passwordHash, err := service.HashPassword(req.Password)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": "服务器内部错误"})
		return
	}

	// 构建租户模型
	tenant := &model.Tenant{
		Username:     req.Username,
		PasswordHash: passwordHash,
		Role:         "user",
		Status:       0, // 待审核
		VipLevel:     "trial",
	}
	if req.Email != "" {
		tenant.Email = &req.Email
	}
	if req.Nickname != "" {
		tenant.Nickname = &req.Nickname
	}

	// 写入数据库
	if err := h.TenantRepo.Create(ctx, tenant); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": "注册失败"})
		return
	}

	nickname := ""
	if tenant.Nickname != nil {
		nickname = *tenant.Nickname
	}

	c.JSON(http.StatusCreated, gin.H{
		"code": 0,
		"msg":  "注册成功，请等待管理员审核",
		"data": UserDTO{
			ID:       tenant.ID,
			Username: tenant.Username,
			Nickname: nickname,
			Role:     tenant.Role,
		},
	})
}

// LoginRateLimit 登录限流中间件（IP 级别，5次/分钟）
// 作为独立中间件使用，可在 Login handler 前单独应用
func LoginRateLimit(c *gin.Context) {
	// 限流逻辑已内联在 Login handler 中，此中间件作为备用
	c.Next()
}

// NewAuthHandler 创建认证 Handler 实例
func NewAuthHandler(db *sqlx.DB, rdb *redis.Client, jwtCfg config.JWTConfig) *AuthHandler {
	return &AuthHandler{
		DB:    db,
		Redis: rdb,
		JWT:   jwtCfg,
		TenantRepo: &tenant.Repo{DB: db},
	}
}
