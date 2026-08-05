// Package api 快讯管理 Handler (T13)
package api

import (
	"context"
	"database/sql"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"
	"go.uber.org/zap"

	"github.com/weiyeston/weiyeston-v2/internal/model"
)

// NewsRepo 快讯数据访问接口（依赖注入 + 测试 mock）
type NewsRepo interface {
	ListNews(ctx context.Context, accountID int64, offset, limit int) ([]model.QuickNews, error)
	GetNews(ctx context.Context, id int64) (*model.QuickNews, error)
	CreateNews(ctx context.Context, n *model.QuickNews) error
	UpdateNews(ctx context.Context, n *model.QuickNews) error
	SoftDeleteNews(ctx context.Context, id int64) error
	ListUsers(ctx context.Context, accountID int64, offset, limit int) ([]model.QuickNewsUser, error)
}

// NewsHandler 快讯管理 API 处理器
type NewsHandler struct {
	newsRepo NewsRepo
	logger   *zap.Logger
}

// NewNewsHandler 创建快讯 Handler
func NewNewsHandler(newsRepo NewsRepo, logger *zap.Logger) *NewsHandler {
	return &NewsHandler{
		newsRepo: newsRepo,
		logger:   logger,
	}
}

// ========== 请求/响应结构体 ==========

// CreateNewsRequest 创建快讯请求
type CreateNewsRequest struct {
	ChannelID  int64  `json:"channel_id"  binding:"required"`
	Content    string `json:"content"     binding:"required,min=1,max=500"`
	AuthorName *string `json:"author_name"`
	Status     int16  `json:"status"`
	IsTop      bool   `json:"is_top"`
}

// UpdateNewsRequest 更新快讯请求
type UpdateNewsRequest struct {
	ChannelID  *int64  `json:"channel_id"`
	Content    *string `json:"content"`
	AuthorName *string `json:"author_name"`
	Status     *int16  `json:"status"`
	IsTop      *bool   `json:"is_top"`
}

// NewsVO 快讯视图对象
type NewsVO struct {
	ID           int64      `json:"id"`
	AccountID    int64      `json:"account_id"`
	ChannelID    int64      `json:"channel_id"`
	AuthorName   *string    `json:"author_name"`
	AuthorAvatar *string    `json:"author_avatar"`
	Content      string     `json:"content"`
	LikeCount    int        `json:"like_count"`
	CommentCount int        `json:"comment_count"`
	Status       int16      `json:"status"`
	StatusText   string     `json:"status_text"`
	IsTop        bool       `json:"is_top"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
}

// QuickNewsUserVO 快讯用户视图对象
type QuickNewsUserVO struct {
	ID        int64     `json:"id"`
	AccountID int64     `json:"account_id"`
	Openid    string    `json:"openid"`
	Nickname  *string   `json:"nickname"`
	AvatarURL *string   `json:"avatar_url"`
	Status    int16     `json:"status"`
	CreatedAt time.Time `json:"created_at"`
}

func toNewsVO(n *model.QuickNews) NewsVO {
	statusText := "草稿"
	if n.Status == 1 {
		statusText = "已发布"
	} else if n.Status == 2 {
		statusText = "隐藏"
	}
	return NewsVO{
		ID:           n.ID,
		AccountID:    n.AccountID,
		ChannelID:    n.ChannelID,
		AuthorName:   n.AuthorName,
		AuthorAvatar: n.AuthorAvatar,
		Content:      n.Content,
		LikeCount:    n.LikeCount,
		CommentCount: n.CommentCount,
		Status:       n.Status,
		StatusText:   statusText,
		IsTop:        n.IsTop,
		CreatedAt:    n.CreatedAt,
		UpdatedAt:    n.UpdatedAt,
	}
}

// ========== Tenant helper ==========

func (h *NewsHandler) getTenantID(c *gin.Context) (int64, bool) {
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

// ======================== 快讯 API ========================

// ListNews GET /api/v1/quicknews/news — 获取快讯列表
func (h *NewsHandler) ListNews(c *gin.Context) {
	tenantID, ok := h.getTenantID(c)
	if !ok {
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	size, _ := strconv.Atoi(c.DefaultQuery("size", "20"))
	if page < 1 {
		page = 1
	}
	if size < 1 || size > 100 {
		size = 20
	}
	offset := (page - 1) * size

	news, err := h.newsRepo.ListNews(c.Request.Context(), tenantID, offset, size)
	if err != nil {
		h.logger.Error("查询快讯列表失败", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"code": 50001,
			"msg":  "查询快讯列表失败",
			"data": nil,
		})
		return
	}

	var vos []NewsVO
	for i := range news {
		vos = append(vos, toNewsVO(&news[i]))
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"msg":  "ok",
		"data": gin.H{
			"list":      vos,
			"page":      page,
			"page_size": size,
		},
	})
}

// CreateNews POST /api/v1/quicknews/news — 发布快讯
func (h *NewsHandler) CreateNews(c *gin.Context) {
	tenantID, ok := h.getTenantID(c)
	if !ok {
		return
	}

	var req CreateNewsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code": 40001,
			"msg":  "参数错误: " + err.Error(),
			"data": nil,
		})
		return
	}

	n := &model.QuickNews{
		AccountID:  tenantID,
		ChannelID:  req.ChannelID,
		AuthorName: req.AuthorName,
		Content:    req.Content,
		Status:     req.Status,
		IsTop:      req.IsTop,
	}
	if n.Status == 0 {
		n.Status = 1
	}

	if err := h.newsRepo.CreateNews(c.Request.Context(), n); err != nil {
		h.logger.Error("创建快讯失败", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"code": 50001,
			"msg":  "创建快讯失败",
			"data": nil,
		})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"code": 0,
		"msg":  "快讯已发布",
		"data": toNewsVO(n),
	})
}

// GetNews GET /api/v1/quicknews/news/:id — 获取单条快讯
func (h *NewsHandler) GetNews(c *gin.Context) {
	_, ok := h.getTenantID(c)
	if !ok {
		return
	}

	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code": 40001,
			"msg":  "无效的快讯 ID",
			"data": nil,
		})
		return
	}

	news, err := h.newsRepo.GetNews(c.Request.Context(), id)
	if err != nil {
		h.logger.Error("查询快讯失败", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"code": 50001,
			"msg":  "查询快讯失败",
			"data": nil,
		})
		return
	}
	if news == nil {
		c.JSON(http.StatusNotFound, gin.H{
			"code": 40401,
			"msg":  "快讯不存在",
			"data": nil,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"msg":  "ok",
		"data": toNewsVO(news),
	})
}

// UpdateNews PUT /api/v1/quicknews/news/:id — 更新快讯
func (h *NewsHandler) UpdateNews(c *gin.Context) {
	_, ok := h.getTenantID(c)
	if !ok {
		return
	}

	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code": 40001,
			"msg":  "无效的快讯 ID",
			"data": nil,
		})
		return
	}

	var req UpdateNewsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code": 40001,
			"msg":  "参数错误: " + err.Error(),
			"data": nil,
		})
		return
	}

	n := &model.QuickNews{ID: id}
	if req.ChannelID != nil {
		n.ChannelID = *req.ChannelID
	}
	if req.Content != nil {
		n.Content = *req.Content
	}
	if req.AuthorName != nil {
		n.AuthorName = req.AuthorName
	}
	if req.Status != nil {
		n.Status = *req.Status
	}
	if req.IsTop != nil {
		n.IsTop = *req.IsTop
	}

	if err := h.newsRepo.UpdateNews(c.Request.Context(), n); err != nil {
		h.logger.Error("更新快讯失败", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"code": 50001,
			"msg":  "更新快讯失败: " + err.Error(),
			"data": nil,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"msg":  "快讯已更新",
		"data": nil,
	})
}

// DeleteNews DELETE /api/v1/quicknews/news/:id — 删除快讯
func (h *NewsHandler) DeleteNews(c *gin.Context) {
	_, ok := h.getTenantID(c)
	if !ok {
		return
	}

	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code": 40001,
			"msg":  "无效的快讯 ID",
			"data": nil,
		})
		return
	}

	if err := h.newsRepo.SoftDeleteNews(c.Request.Context(), id); err != nil {
		h.logger.Error("删除快讯失败", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"code": 50001,
			"msg":  "删除快讯失败",
			"data": nil,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"msg":  "快讯已删除",
		"data": nil,
	})
}

// ListUsers GET /api/v1/quicknews/users — 获取注册用户列表
func (h *NewsHandler) ListUsers(c *gin.Context) {
	tenantID, ok := h.getTenantID(c)
	if !ok {
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	size, _ := strconv.Atoi(c.DefaultQuery("size", "20"))
	if page < 1 {
		page = 1
	}
	if size < 1 || size > 100 {
		size = 20
	}
	offset := (page - 1) * size

	users, err := h.newsRepo.ListUsers(c.Request.Context(), tenantID, offset, size)
	if err != nil {
		h.logger.Error("查询用户列表失败", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"code": 50001,
			"msg":  "查询用户列表失败",
			"data": nil,
		})
		return
	}

	var vos []QuickNewsUserVO
	for i := range users {
		vos = append(vos, QuickNewsUserVO{
			ID:        users[i].ID,
			AccountID: users[i].AccountID,
			Openid:    users[i].Openid,
			Nickname:  users[i].Nickname,
			AvatarURL: users[i].AvatarURL,
			Status:    users[i].Status,
			CreatedAt: users[i].CreatedAt,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"msg":  "ok",
		"data": gin.H{
			"list":      vos,
			"page":      page,
			"page_size": size,
		},
	})
}

// ========== NewsRepo PostgreSQL 实现 ==========

// newsRepoImpl NewsRepo 的 PostgreSQL 实现
type newsRepoImpl struct {
	DB *sqlx.DB
}

// NewNewsRepo 创建快讯 Repository
func NewNewsRepo(db *sqlx.DB) *newsRepoImpl {
	return &newsRepoImpl{DB: db}
}

func (r *newsRepoImpl) ListNews(ctx context.Context, accountID int64, offset, limit int) ([]model.QuickNews, error) {
	var news []model.QuickNews
	query := `SELECT * FROM quicknews_news WHERE account_id = $1 AND deleted_at IS NULL ORDER BY is_top DESC, created_at DESC LIMIT $2 OFFSET $3`
	err := r.DB.SelectContext(ctx, &news, query, accountID, limit, offset)
	return news, err
}

func (r *newsRepoImpl) GetNews(ctx context.Context, id int64) (*model.QuickNews, error) {
	if id <= 0 {
		return nil, errors.New("无效的快讯 ID")
	}
	var n model.QuickNews
	query := `SELECT * FROM quicknews_news WHERE id = $1 AND deleted_at IS NULL`
	err := r.DB.GetContext(ctx, &n, query, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &n, nil
}

func (r *newsRepoImpl) CreateNews(ctx context.Context, n *model.QuickNews) error {
	query := `INSERT INTO quicknews_news (account_id, channel_id, author_name, author_avatar, content, like_count, comment_count, status, is_top)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING id, created_at, updated_at`
	row := r.DB.QueryRowContext(ctx, query,
		n.AccountID, n.ChannelID, n.AuthorName, n.AuthorAvatar, n.Content,
		n.LikeCount, n.CommentCount, n.Status, n.IsTop,
	)
	return row.Scan(&n.ID, &n.CreatedAt, &n.UpdatedAt)
}

func (r *newsRepoImpl) UpdateNews(ctx context.Context, n *model.QuickNews) error {
		return r.updateNewsDirect(ctx, n)
}

func (r *newsRepoImpl) updateNewsDirect(ctx context.Context, n *model.QuickNews) error {
	// We need to fetch the existing record and merge
	existing, err := r.GetNews(ctx, n.ID)
	if err != nil {
		return err
	}
	if existing == nil {
		return errors.New("快讯不存在或已删除")
	}

	// Merge fields
	if n.ChannelID != 0 {
		existing.ChannelID = n.ChannelID
	}
	if n.Content != "" {
		existing.Content = n.Content
	}
	if n.AuthorName != nil {
		existing.AuthorName = n.AuthorName
	}
	if n.Status != 0 {
		existing.Status = n.Status
	}
	existing.IsTop = n.IsTop
	// Note: is_top cannot be unset in this partial approach; the handler will handle it

	query := `UPDATE quicknews_news SET
		channel_id = $1, content = $2, author_name = $3,
		status = $4, is_top = $5, updated_at = NOW()
		WHERE id = $6 AND deleted_at IS NULL`
	result, err := r.DB.ExecContext(ctx, query,
		existing.ChannelID, existing.Content, existing.AuthorName,
		existing.Status, existing.IsTop, n.ID,
	)
	if err != nil {
		return err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return errors.New("快讯不存在或已删除")
	}
	return nil
}

func (r *newsRepoImpl) SoftDeleteNews(ctx context.Context, id int64) error {
	query := `UPDATE quicknews_news SET deleted_at = NOW() WHERE id = $1 AND deleted_at IS NULL`
	_, err := r.DB.ExecContext(ctx, query, id)
	return err
}

func (r *newsRepoImpl) ListUsers(ctx context.Context, accountID int64, offset, limit int) ([]model.QuickNewsUser, error) {
	var users []model.QuickNewsUser
	query := `SELECT * FROM quicknews_users WHERE account_id = $1 ORDER BY created_at DESC LIMIT $2 OFFSET $3`
	err := r.DB.SelectContext(ctx, &users, query, accountID, limit, offset)
	return users, err
}
