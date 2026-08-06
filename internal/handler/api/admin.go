// Package api 管理后台用户管理 Handler
// 提供管理员级别的用户列表查询和用户状态更新
package api

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"

	"github.com/weiyeston/weiyeston-v2/internal/repository/tenant"
)

// AdminHandler 管理员操作处理器
type AdminHandler struct {
	DB         *sqlx.DB
	TenantRepo *tenant.Repo
}

// NewAdminHandler 创建管理员 Handler 实例
func NewAdminHandler(db *sqlx.DB) *AdminHandler {
	return &AdminHandler{
		DB:         db,
		TenantRepo: &tenant.Repo{DB: db},
	}
}

// ListUsersRequest 用户列表查询参数
type ListUsersRequest struct {
	Page     int    `form:"page,default=1"       binding:"omitempty,min=1"`
	PageSize int    `form:"page_size,default=20" binding:"omitempty,min=1,max=100"`
	Keyword  string `form:"keyword"`
	Status   *int16 `form:"status"`
}

// ListUsers GET /api/v1/admin/users — 查询用户列表（管理员）
func (h *AdminHandler) ListUsers(c *gin.Context) {
	var req ListUsersRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code": 40002,
			"msg":  "分页参数错误",
			"data": nil,
		})
		return
	}

	offset := (req.Page - 1) * req.PageSize
	users, total, err := h.TenantRepo.List(c.Request.Context(), req.Keyword, req.Status, req.PageSize, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code": 50001,
			"msg":  "查询用户列表失败",
			"data": nil,
		})
		return
	}

	type userItem struct {
		ID          int64  `json:"id"`
		Username    string `json:"username"`
		Nickname    string `json:"nickname"`
		Email       string `json:"email,omitempty"`
		Role        string `json:"role"`
		Status      int16  `json:"status"`
		VipLevel    string `json:"vip_level"`
		CreatedAt   string `json:"created_at"`
		LastLoginAt string `json:"last_login_at,omitempty"`
	}

	list := make([]userItem, 0, len(users))
	for i := range users {
		u := users[i]
		item := userItem{
			ID:        u.ID,
			Username:  u.Username,
			Role:      u.Role,
			Status:    u.Status,
			VipLevel:  u.VipLevel,
			CreatedAt: u.CreatedAt.Format("2006-01-02T15:04:05Z"),
		}
		if u.Nickname != nil {
			item.Nickname = *u.Nickname
		}
		if u.Email != nil {
			item.Email = *u.Email
		}
		if u.LastLoginAt != nil {
			item.LastLoginAt = u.LastLoginAt.Format("2006-01-02T15:04:05Z")
		}
		list = append(list, item)
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"msg":  "ok",
		"data": gin.H{
			"list":      list,
			"total":     total,
			"page":      req.Page,
			"page_size": req.PageSize,
		},
	})
}

// UpdateUserRequest 更新用户请求体
type UpdateUserRequest struct {
	Status     *int16  `json:"status"      binding:"omitempty,oneof=0 1 2"`
	Role       *string `json:"role"        binding:"omitempty,oneof=admin user"`
	VipLevel   *string `json:"vip_level"   binding:"omitempty,oneof=trial basic pro enterprise"`
	VipEndTime *string `json:"vip_end_time"` // ISO 日期字符串，如 "2026-12-31T00:00:00Z"
}

// UpdateUser PUT /api/v1/admin/users/:id — 更新用户状态/角色（管理员）
func (h *AdminHandler) UpdateUser(c *gin.Context) {
	userID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code": 40001,
			"msg":  "无效的用户 ID",
			"data": nil,
		})
		return
	}

	var req UpdateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code": 40001,
			"msg":  "参数错误: " + err.Error(),
			"data": nil,
		})
		return
	}

	// 查询用户
	user, err := h.TenantRepo.GetByID(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code": 50001,
			"msg":  "服务器内部错误",
			"data": nil,
		})
		return
	}
	if user == nil {
		c.JSON(http.StatusNotFound, gin.H{
			"code": 40401,
			"msg":  "用户不存在",
			"data": nil,
		})
		return
	}

	// 更新允许的字段
	if req.Status != nil {
		user.Status = *req.Status
	}
	if req.Role != nil {
		user.Role = *req.Role
	}
	if req.VipLevel != nil {
		user.VipLevel = *req.VipLevel
	}
	if req.VipEndTime != nil && *req.VipEndTime != "" {
		t, err := time.Parse(time.RFC3339, *req.VipEndTime)
		if err != nil {
			t, err = time.Parse("2006-01-02", *req.VipEndTime)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{
					"code": 40001,
					"msg":  "无效的时间格式",
					"data": nil,
				})
				return
			}
		}
		user.VipEndTime = &t
	}

	if err := h.TenantRepo.Update(c.Request.Context(), user); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code": 50001,
			"msg":  "更新用户失败",
			"data": nil,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"msg":  "ok",
		"data": nil,
	})
}
