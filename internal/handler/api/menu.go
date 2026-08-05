// Package api 微信自定义菜单管理 Handler
package api

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"
	"go.uber.org/zap"

	"github.com/weiyeston/weiyeston-v2/internal/model"
)

// MenuRepo 菜单数据访问接口（依赖注入 + 测试 mock）
type MenuRepo interface {
	GetByAccountID(ctx context.Context, accountID int64) (*model.WechatMenu, error)
	Create(ctx context.Context, m *model.WechatMenu) error
	Update(ctx context.Context, m *model.WechatMenu) error
	SoftDelete(ctx context.Context, id int64) (bool, error)
}

// menuRepoImpl MenuRepo 的 PostgreSQL 实现
type menuRepoImpl struct {
	DB *sqlx.DB
}

// NewMenuRepo 创建菜单 Repository
func NewMenuRepo(db *sqlx.DB) *menuRepoImpl {
	return &menuRepoImpl{DB: db}
}

// GetByAccountID 获取某公众号的最新菜单（草稿或已发布，排除软删除）
func (r *menuRepoImpl) GetByAccountID(ctx context.Context, accountID int64) (*model.WechatMenu, error) {
	if accountID <= 0 {
		return nil, errors.New("无效的公众号 ID")
	}
	var m model.WechatMenu
	query := `SELECT * FROM wechat_menus WHERE account_id = $1 AND deleted_at IS NULL ORDER BY status DESC, updated_at DESC LIMIT 1`
	err := r.DB.GetContext(ctx, &m, query, accountID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &m, nil
}

// Create 创建菜单记录
func (r *menuRepoImpl) Create(ctx context.Context, m *model.WechatMenu) error {
	query := `INSERT INTO wechat_menus (account_id, menu_json, status, published_at)
		VALUES ($1, $2, $3, $4)
		RETURNING id, created_at, updated_at`
	row := r.DB.QueryRowContext(ctx, query, m.AccountID, m.MenuJSON, m.Status, m.PublishedAt)
	return row.Scan(&m.ID, &m.CreatedAt, &m.UpdatedAt)
}

// Update 更新菜单记录
func (r *menuRepoImpl) Update(ctx context.Context, m *model.WechatMenu) error {
	query := `UPDATE wechat_menus SET
		menu_json = $1, status = $2, published_at = $3, updated_at = NOW()
		WHERE id = $4 AND deleted_at IS NULL`
	result, err := r.DB.ExecContext(ctx, query, m.MenuJSON, m.Status, m.PublishedAt, m.ID)
	if err != nil {
		return err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return errors.New("菜单不存在或已删除")
	}
	return nil
}

// SoftDelete 软删除菜单
func (r *menuRepoImpl) SoftDelete(ctx context.Context, id int64) (bool, error) {
	query := `UPDATE wechat_menus SET deleted_at = NOW() WHERE id = $1 AND deleted_at IS NULL`
	result, err := r.DB.ExecContext(ctx, query, id)
	if err != nil {
		return false, err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return false, err
	}
	return rows > 0, nil
}

// MenuHandler 微信自定义菜单管理 API 处理器
type MenuHandler struct {
	menuRepo MenuRepo
	logger   *zap.Logger
}

// NewMenuHandler 创建菜单 Handler
func NewMenuHandler(menuRepo MenuRepo, logger *zap.Logger) *MenuHandler {
	return &MenuHandler{
		menuRepo: menuRepo,
		logger:   logger,
	}
}

// ========== 请求/响应结构体 ==========

// SaveMenuRequest 保存菜单草稿请求体
type SaveMenuRequest struct {
	MenuJSON json.RawMessage `json:"menu_json" binding:"required"`
}

// MenuVO 返回给前端的菜单视图对象
type MenuVO struct {
	ID          int64            `json:"id"`
	AccountID   int64            `json:"account_id"`
	MenuJSON    *json.RawMessage `json:"menu_json"`
	Status      int16            `json:"status"`
	StatusText  string           `json:"status_text"`
	PublishedAt *time.Time       `json:"published_at"`
	UpdatedAt   time.Time        `json:"updated_at"`
}

func toMenuVO(m *model.WechatMenu) MenuVO {
	statusText := "草稿"
	if m.Status == model.MenuStatusPublished {
		statusText = "已发布"
	}
	return MenuVO{
		ID:          m.ID,
		AccountID:   m.AccountID,
		MenuJSON:    m.MenuJSON,
		Status:      m.Status,
		StatusText:  statusText,
		PublishedAt: m.PublishedAt,
		UpdatedAt:   m.UpdatedAt,
	}
}

// ========== Handler 方法 ==========

// getTenantIDMenu extract tenant_id from context
func (h *MenuHandler) getTenantIDMenu(c *gin.Context) (int64, bool) {
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

// GetMenu GET /api/v1/accounts/:id/menu — 获取当前菜单
func (h *MenuHandler) GetMenu(c *gin.Context) {
	accountID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code": 40001,
			"msg":  "无效的公众号 ID",
			"data": nil,
		})
		return
	}

	_, ok := h.getTenantIDMenu(c)
	if !ok {
		return
	}

	menu, err := h.menuRepo.GetByAccountID(c.Request.Context(), accountID)
	if err != nil {
		h.logger.Error("查询菜单失败", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"code": 50001,
			"msg":  "查询菜单失败",
			"data": nil,
		})
		return
	}
	if menu == nil {
		c.JSON(http.StatusOK, gin.H{
			"code": 0,
			"msg":  "ok",
			"data": nil, // 无菜单数据
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"msg":  "ok",
		"data": toMenuVO(menu),
	})
}

// SaveDraft POST /api/v1/accounts/:id/menu — 保存草稿
func (h *MenuHandler) SaveDraft(c *gin.Context) {
	accountID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code": 40001,
			"msg":  "无效的公众号 ID",
			"data": nil,
		})
		return
	}

	_, ok := h.getTenantIDMenu(c)
	if !ok {
		return
	}

	var req SaveMenuRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code": 40001,
			"msg":  "参数错误: " + err.Error(),
			"data": nil,
		})
		return
	}

	// 检查是否已有草稿
	existingMenu, err := h.menuRepo.GetByAccountID(c.Request.Context(), accountID)
	if err != nil {
		h.logger.Error("查询已有菜单失败", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"code": 50001,
			"msg":  "服务器内部错误",
			"data": nil,
		})
		return
	}

	if existingMenu != nil {
		// 更新已有草稿
		existingMenu.MenuJSON = &req.MenuJSON
		if err := h.menuRepo.Update(c.Request.Context(), existingMenu); err != nil {
			h.logger.Error("更新菜单草稿失败", zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{
				"code": 50001,
				"msg":  "更新菜单草稿失败",
				"data": nil,
			})
			return
		}
		c.JSON(http.StatusOK, gin.H{
			"code": 0,
			"msg":  "草稿已更新",
			"data": toMenuVO(existingMenu),
		})
		return
	}

	// 创建新草稿
	menu := &model.WechatMenu{
		AccountID: accountID,
		MenuJSON:  &req.MenuJSON,
		Status:    model.MenuStatusDraft,
	}

	if err := h.menuRepo.Create(c.Request.Context(), menu); err != nil {
		h.logger.Error("创建菜单草稿失败", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"code": 50001,
			"msg":  "创建菜单草稿失败",
			"data": nil,
		})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"code": 0,
		"msg":  "草稿已保存",
		"data": toMenuVO(menu),
	})
}

// Publish PUT /api/v1/accounts/:id/menu/publish — 发布菜单到微信
func (h *MenuHandler) Publish(c *gin.Context) {
	accountID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code": 40001,
			"msg":  "无效的公众号 ID",
			"data": nil,
		})
		return
	}

	_, ok := h.getTenantIDMenu(c)
	if !ok {
		return
	}

	// 查询草稿菜单
	menu, err := h.menuRepo.GetByAccountID(c.Request.Context(), accountID)
	if err != nil {
		h.logger.Error("查询菜单失败", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"code": 50001,
			"msg":  "查询菜单失败",
			"data": nil,
		})
		return
	}
	if menu == nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code": 40001,
			"msg":  "请先保存菜单草稿",
			"data": nil,
		})
		return
	}

	// TODO: 调用微信 API 创建/发布菜单
	// 微信接口: POST https://api.weixin.qq.com/cgi-bin/menu/create?access_token=ACCESS_TOKEN
	// 当前阶段标记为已发布，后续集成微信 API

	now := time.Now()
	menu.Status = model.MenuStatusPublished
	menu.PublishedAt = &now

	if err := h.menuRepo.Update(c.Request.Context(), menu); err != nil {
		h.logger.Error("发布菜单失败", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"code": 50001,
			"msg":  "发布菜单失败",
			"data": nil,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"msg":  "菜单已发布",
		"data": toMenuVO(menu),
	})
}

// DeleteDraft DELETE /api/v1/accounts/:id/menu — 删除草稿
func (h *MenuHandler) DeleteDraft(c *gin.Context) {
	accountID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code": 40001,
			"msg":  "无效的公众号 ID",
			"data": nil,
		})
		return
	}

	_, ok := h.getTenantIDMenu(c)
	if !ok {
		return
	}

	menu, err := h.menuRepo.GetByAccountID(c.Request.Context(), accountID)
	if err != nil {
		h.logger.Error("查询菜单失败", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"code": 50001,
			"msg":  "查询菜单失败",
			"data": nil,
		})
		return
	}
	if menu == nil {
		c.JSON(http.StatusNotFound, gin.H{
			"code": 40401,
			"msg":  "菜单不存在",
			"data": nil,
		})
		return
	}

	// 仅允许删除草稿状态的菜单
	if menu.Status == model.MenuStatusPublished {
		c.JSON(http.StatusBadRequest, gin.H{
			"code": 40001,
			"msg":  "已发布的菜单不能直接删除，请先取消发布",
			"data": nil,
		})
		return
	}

	deleted, err := h.menuRepo.SoftDelete(c.Request.Context(), menu.ID)
	if err != nil {
		h.logger.Error("删除菜单失败", zap.Error(err))
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
			"msg":  "菜单不存在或已删除",
			"data": nil,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"msg":  "已删除",
		"data": nil,
	})
}
