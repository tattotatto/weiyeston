// Package api 微官网 CMS Handler (T12)
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

// CMSRepo CMS 数据访问接口（依赖注入 + 测试 mock）
type CMSRepo interface {
	// 栏目
	CreateChannel(ctx context.Context, ch *model.Channel) error
	GetChannelTree(ctx context.Context, accountID int64) ([]model.Channel, error)
	UpdateChannel(ctx context.Context, ch *model.Channel) error
	SoftDeleteChannel(ctx context.Context, id int64) error
	// 文章
	CreateArticle(ctx context.Context, a *model.Article) error
	GetArticle(ctx context.Context, id int64) (*model.Article, error)
	ListArticles(ctx context.Context, accountID int64, channelID *int64, status *int16, offset, limit int) ([]model.Article, error)
	UpdateArticle(ctx context.Context, a *model.Article) error
	SoftDeleteArticle(ctx context.Context, id int64) error
}

// CMSHandler 微官网 CMS API 处理器
type CMSHandler struct {
	cmsRepo CMSRepo
	logger  *zap.Logger
}

// NewCMSHandler 创建 CMS Handler
func NewCMSHandler(cmsRepo CMSRepo, logger *zap.Logger) *CMSHandler {
	return &CMSHandler{
		cmsRepo: cmsRepo,
		logger:  logger,
	}
}

// ========== 请求/响应结构体 ==========

// CreateChannelRequest 创建栏目请求
type CreateChannelRequest struct {
	Name        string  `json:"name"        binding:"required,min=1,max=100"`
	ParentID    *int64  `json:"parent_id"`
	Slug        *string `json:"slug"`
	Level       int16   `json:"level"`
	SortOrder   int     `json:"sort_order"`
	CoverURL    *string `json:"cover_url"`
	Description *string `json:"description"`
	Status      int16   `json:"status"`
}

// UpdateChannelRequest 更新栏目请求
type UpdateChannelRequest struct {
	Name        *string `json:"name"`
	ParentID    *int64  `json:"parent_id"`
	Slug        *string `json:"slug"`
	Level       *int16  `json:"level"`
	SortOrder   *int    `json:"sort_order"`
	CoverURL    *string `json:"cover_url"`
	Description *string `json:"description"`
	Status      *int16  `json:"status"`
}

// CreateArticleRequest 创建文章请求
type CreateArticleRequest struct {
	ChannelID   *int64           `json:"channel_id"`
	Title       *string          `json:"title"`
	CoverURL    *string          `json:"cover_url"`
	Summary     *string          `json:"summary"`
	Author      *string          `json:"author"`
	Content     json.RawMessage  `json:"content"      binding:"required"`
	Status      int16            `json:"status"`
	SortOrder   int              `json:"sort_order"`
	IsTemplate  bool             `json:"is_template"`
	TemplateCat *string          `json:"template_cat"`
}

// UpdateArticleRequest 更新文章请求
type UpdateArticleRequest struct {
	ChannelID   *int64           `json:"channel_id"`
	Title       *string          `json:"title"`
	CoverURL    *string          `json:"cover_url"`
	Summary     *string          `json:"summary"`
	Author      *string          `json:"author"`
	Content     *json.RawMessage `json:"content"`
	Status      *int16           `json:"status"`
	SortOrder   *int             `json:"sort_order"`
	IsTemplate  *bool            `json:"is_template"`
	TemplateCat *string          `json:"template_cat"`
}

// ChannelVO 栏目视图对象
type ChannelVO struct {
	ID          int64      `json:"id"`
	AccountID   int64      `json:"account_id"`
	ParentID    *int64     `json:"parent_id"`
	Name        string     `json:"name"`
	Slug        *string    `json:"slug"`
	Level       int16      `json:"level"`
	SortOrder   int        `json:"sort_order"`
	CoverURL    *string    `json:"cover_url"`
	Description *string    `json:"description"`
	Status      int16      `json:"status"`
	Children    []*ChannelVO `json:"children,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

// ArticleVO 文章视图对象
type ArticleVO struct {
	ID          int64            `json:"id"`
	AccountID   int64            `json:"account_id"`
	ChannelID   *int64           `json:"channel_id"`
	Title       *string          `json:"title"`
	CoverURL    *string          `json:"cover_url"`
	Summary     *string          `json:"summary"`
	Author      *string          `json:"author"`
	Content     json.RawMessage  `json:"content"`
	Status      int16            `json:"status"`
	StatusText  string           `json:"status_text"`
	IsTemplate  bool             `json:"is_template"`
	TemplateCat *string          `json:"template_cat"`
	SortOrder   int              `json:"sort_order"`
	ViewCount   int              `json:"view_count"`
	PublishedAt *time.Time       `json:"published_at"`
	CreatedAt   time.Time        `json:"created_at"`
	UpdatedAt   time.Time        `json:"updated_at"`
}

func toArticleVO(a *model.Article) ArticleVO {
	statusText := "草稿"
	if a.Status == 1 {
		statusText = "已发布"
	}
	return ArticleVO{
		ID:          a.ID,
		AccountID:   a.AccountID,
		ChannelID:   a.ChannelID,
		Title:       a.Title,
		CoverURL:    a.CoverURL,
		Summary:     a.Summary,
		Author:      a.Author,
		Content:     a.Content,
		Status:      a.Status,
		StatusText:  statusText,
		IsTemplate:  a.IsTemplate,
		TemplateCat: a.TemplateCat,
		SortOrder:   a.SortOrder,
		ViewCount:   a.ViewCount,
		PublishedAt: a.PublishedAt,
		CreatedAt:   a.CreatedAt,
		UpdatedAt:   a.UpdatedAt,
	}
}

// ========== Tenant helper ==========

func (h *CMSHandler) getTenantID(c *gin.Context) (int64, bool) {
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

// ======================== 栏目 API ========================

// buildChannelTree 将平铺的栏目列表构建为树形结构
func buildChannelTree(channels []model.Channel) []*ChannelVO {
	var roots []*ChannelVO
	nodeMap := make(map[int64]*ChannelVO)

	for i := range channels {
		vo := &ChannelVO{
			ID:          channels[i].ID,
			AccountID:   channels[i].AccountID,
			ParentID:    channels[i].ParentID,
			Name:        channels[i].Name,
			Slug:        channels[i].Slug,
			Level:       channels[i].Level,
			SortOrder:   channels[i].SortOrder,
			CoverURL:    channels[i].CoverURL,
			Description: channels[i].Description,
			Status:      channels[i].Status,
			Children:    []*ChannelVO{},
			CreatedAt:   channels[i].CreatedAt,
			UpdatedAt:   channels[i].UpdatedAt,
		}
		nodeMap[channels[i].ID] = vo
	}

	for i := range channels {
		vo := nodeMap[channels[i].ID]
		if channels[i].ParentID != nil {
			if parent, ok := nodeMap[*channels[i].ParentID]; ok {
				parent.Children = append(parent.Children, vo)
			} else {
				roots = append(roots, vo)
			}
		} else {
			roots = append(roots, vo)
		}
	}

	return roots
}

// ListChannels GET /api/v1/cms/channels — 获取栏目树
func (h *CMSHandler) ListChannels(c *gin.Context) {
	tenantID, ok := h.getTenantID(c)
	if !ok {
		return
	}

	channels, err := h.cmsRepo.GetChannelTree(c.Request.Context(), tenantID)
	if err != nil {
		h.logger.Error("查询栏目树失败", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"code": 50001,
			"msg":  "查询栏目树失败",
			"data": nil,
		})
		return
	}

	tree := buildChannelTree(channels)
	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"msg":  "ok",
		"data": tree,
	})
}

// CreateChannel POST /api/v1/cms/channels — 创建栏目
func (h *CMSHandler) CreateChannel(c *gin.Context) {
	tenantID, ok := h.getTenantID(c)
	if !ok {
		return
	}

	var req CreateChannelRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code": 40001,
			"msg":  "参数错误: " + err.Error(),
			"data": nil,
		})
		return
	}

	ch := &model.Channel{
		AccountID:   tenantID,
		ParentID:    req.ParentID,
		Name:        req.Name,
		Slug:        req.Slug,
		Level:       req.Level,
		SortOrder:   req.SortOrder,
		CoverURL:    req.CoverURL,
		Description: req.Description,
		Status:      req.Status,
	}
	if req.Status == 0 {
		ch.Status = 1
	}

	if err := h.cmsRepo.CreateChannel(c.Request.Context(), ch); err != nil {
		h.logger.Error("创建栏目失败", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"code": 50001,
			"msg":  "创建栏目失败",
			"data": nil,
		})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"code": 0,
		"msg":  "栏目已创建",
		"data": ChannelVO{
			ID:          ch.ID,
			AccountID:   ch.AccountID,
			ParentID:    ch.ParentID,
			Name:        ch.Name,
			Slug:        ch.Slug,
			Level:       ch.Level,
			SortOrder:   ch.SortOrder,
			CoverURL:    ch.CoverURL,
			Description: ch.Description,
			Status:      ch.Status,
			CreatedAt:   ch.CreatedAt,
			UpdatedAt:   ch.UpdatedAt,
		},
	})
}

// UpdateChannel PUT /api/v1/cms/channels/:id — 更新栏目
func (h *CMSHandler) UpdateChannel(c *gin.Context) {
	_, ok := h.getTenantID(c)
	if !ok {
		return
	}

	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code": 40001,
			"msg":  "无效的栏目 ID",
			"data": nil,
		})
		return
	}

	var req UpdateChannelRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code": 40001,
			"msg":  "参数错误: " + err.Error(),
			"data": nil,
		})
		return
	}

	ch := &model.Channel{ID: id}
	if req.Name != nil {
		ch.Name = *req.Name
	}
	if req.ParentID != nil {
		ch.ParentID = req.ParentID
	}
	if req.Slug != nil {
		ch.Slug = req.Slug
	}
	if req.Level != nil {
		ch.Level = *req.Level
	}
	if req.SortOrder != nil {
		ch.SortOrder = *req.SortOrder
	}
	if req.CoverURL != nil {
		ch.CoverURL = req.CoverURL
	}
	if req.Description != nil {
		ch.Description = req.Description
	}
	if req.Status != nil {
		ch.Status = *req.Status
	}

	if err := h.cmsRepo.UpdateChannel(c.Request.Context(), ch); err != nil {
		h.logger.Error("更新栏目失败", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"code": 50001,
			"msg":  "更新栏目失败: " + err.Error(),
			"data": nil,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"msg":  "栏目已更新",
		"data": nil,
	})
}

// DeleteChannel DELETE /api/v1/cms/channels/:id — 删除栏目
func (h *CMSHandler) DeleteChannel(c *gin.Context) {
	_, ok := h.getTenantID(c)
	if !ok {
		return
	}

	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code": 40001,
			"msg":  "无效的栏目 ID",
			"data": nil,
		})
		return
	}

	if err := h.cmsRepo.SoftDeleteChannel(c.Request.Context(), id); err != nil {
		h.logger.Error("删除栏目失败", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"code": 50001,
			"msg":  "删除栏目失败",
			"data": nil,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"msg":  "栏目已删除",
		"data": nil,
	})
}

// ======================== 文章 API ========================

// ListArticles GET /api/v1/cms/articles — 文章列表
func (h *CMSHandler) ListArticles(c *gin.Context) {
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

	var channelID *int64
	if cidStr := c.Query("channel_id"); cidStr != "" {
		if cid, err := strconv.ParseInt(cidStr, 10, 64); err == nil {
			channelID = &cid
		}
	}

	var status *int16
	if statusStr := c.Query("status"); statusStr != "" {
		if s, err := strconv.ParseInt(statusStr, 10, 16); err == nil {
			st := int16(s)
			status = &st
		}
	}

	articles, err := h.cmsRepo.ListArticles(c.Request.Context(), tenantID, channelID, status, offset, size)
	if err != nil {
		h.logger.Error("查询文章列表失败", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"code": 50001,
			"msg":  "查询文章列表失败",
			"data": nil,
		})
		return
	}

	var vos []ArticleVO
	for i := range articles {
		vos = append(vos, toArticleVO(&articles[i]))
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

// GetArticle GET /api/v1/cms/articles/:id — 获取单篇文章
func (h *CMSHandler) GetArticle(c *gin.Context) {
	_, ok := h.getTenantID(c)
	if !ok {
		return
	}

	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code": 40001,
			"msg":  "无效的文章 ID",
			"data": nil,
		})
		return
	}

	article, err := h.cmsRepo.GetArticle(c.Request.Context(), id)
	if err != nil {
		h.logger.Error("查询文章失败", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"code": 50001,
			"msg":  "查询文章失败",
			"data": nil,
		})
		return
	}
	if article == nil {
		c.JSON(http.StatusNotFound, gin.H{
			"code": 40401,
			"msg":  "文章不存在",
			"data": nil,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"msg":  "ok",
		"data": toArticleVO(article),
	})
}

// CreateArticle POST /api/v1/cms/articles — 创建文章
func (h *CMSHandler) CreateArticle(c *gin.Context) {
	tenantID, ok := h.getTenantID(c)
	if !ok {
		return
	}

	var req CreateArticleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code": 40001,
			"msg":  "参数错误: " + err.Error(),
			"data": nil,
		})
		return
	}

	var publishedAt *time.Time
	if req.Status == 1 {
		now := time.Now()
		publishedAt = &now
	}

	a := &model.Article{
		AccountID:   tenantID,
		ChannelID:   req.ChannelID,
		Title:       req.Title,
		CoverURL:    req.CoverURL,
		Summary:     req.Summary,
		Author:      req.Author,
		Content:     req.Content,
		Status:      req.Status,
		IsTemplate:  req.IsTemplate,
		TemplateCat: req.TemplateCat,
		SortOrder:   req.SortOrder,
		PublishedAt: publishedAt,
	}

	if err := h.cmsRepo.CreateArticle(c.Request.Context(), a); err != nil {
		h.logger.Error("创建文章失败", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"code": 50001,
			"msg":  "创建文章失败",
			"data": nil,
		})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"code": 0,
		"msg":  "文章已创建",
		"data": toArticleVO(a),
	})
}

// UpdateArticle PUT /api/v1/cms/articles/:id — 更新文章
func (h *CMSHandler) UpdateArticle(c *gin.Context) {
	_, ok := h.getTenantID(c)
	if !ok {
		return
	}

	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code": 40001,
			"msg":  "无效的文章 ID",
			"data": nil,
		})
		return
	}

	var req UpdateArticleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code": 40001,
			"msg":  "参数错误: " + err.Error(),
			"data": nil,
		})
		return
	}

	a := &model.Article{ID: id}
	if req.ChannelID != nil {
		a.ChannelID = req.ChannelID
	}
	if req.Title != nil {
		a.Title = req.Title
	}
	if req.CoverURL != nil {
		a.CoverURL = req.CoverURL
	}
	if req.Summary != nil {
		a.Summary = req.Summary
	}
	if req.Author != nil {
		a.Author = req.Author
	}
	if req.Content != nil {
		a.Content = *req.Content
	}
	if req.Status != nil {
		a.Status = *req.Status
		if *req.Status == 1 {
			now := time.Now()
			a.PublishedAt = &now
		}
	}
	if req.SortOrder != nil {
		a.SortOrder = *req.SortOrder
	}
	if req.IsTemplate != nil {
		a.IsTemplate = *req.IsTemplate
	}
	if req.TemplateCat != nil {
		a.TemplateCat = req.TemplateCat
	}

	if err := h.cmsRepo.UpdateArticle(c.Request.Context(), a); err != nil {
		h.logger.Error("更新文章失败", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"code": 50001,
			"msg":  "更新文章失败: " + err.Error(),
			"data": nil,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"msg":  "文章已更新",
		"data": nil,
	})
}

// DeleteArticle DELETE /api/v1/cms/articles/:id — 删除文章
func (h *CMSHandler) DeleteArticle(c *gin.Context) {
	_, ok := h.getTenantID(c)
	if !ok {
		return
	}

	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code": 40001,
			"msg":  "无效的文章 ID",
			"data": nil,
		})
		return
	}

	if err := h.cmsRepo.SoftDeleteArticle(c.Request.Context(), id); err != nil {
		h.logger.Error("删除文章失败", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"code": 50001,
			"msg":  "删除文章失败",
			"data": nil,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"msg":  "文章已删除",
		"data": nil,
	})
}

// PreviewArticle GET /api/v1/cms/articles/:id/preview — H5 预览
func (h *CMSHandler) PreviewArticle(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code": 40001,
			"msg":  "无效的文章 ID",
			"data": nil,
		})
		return
	}

	article, err := h.cmsRepo.GetArticle(c.Request.Context(), id)
	if err != nil {
		h.logger.Error("查询文章失败", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"code": 50001,
			"msg":  "查询文章失败",
			"data": nil,
		})
		return
	}
	if article == nil {
		c.JSON(http.StatusNotFound, gin.H{
			"code": 40401,
			"msg":  "文章不存在",
			"data": nil,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"msg":  "ok",
		"data": toArticleVO(article),
	})
}

// ========== CMSRepoDB PostgreSQL 实现 ==========

// CMSRepoDB CMSRepo 的 PostgreSQL 实现，用于路由依赖注入
type CMSRepoDB struct {
	DB *sqlx.DB
}

// NewCMSRepoDB 创建 CMSRepo 数据库实现
func NewCMSRepoDB(db *sqlx.DB) *CMSRepoDB {
	return &CMSRepoDB{DB: db}
}

func (r *CMSRepoDB) CreateChannel(ctx context.Context, ch *model.Channel) error {
	query := `INSERT INTO cms_channels (account_id, parent_id, name, slug, level, sort_order, cover_url, description, status)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING id, created_at, updated_at`
	row := r.DB.QueryRowContext(ctx, query,
		ch.AccountID, ch.ParentID, ch.Name, ch.Slug, ch.Level, ch.SortOrder, ch.CoverURL, ch.Description, ch.Status,
	)
	return row.Scan(&ch.ID, &ch.CreatedAt, &ch.UpdatedAt)
}

func (r *CMSRepoDB) GetChannelTree(ctx context.Context, accountID int64) ([]model.Channel, error) {
	if accountID <= 0 {
		return nil, errors.New("无效的公众号 ID")
	}
	var channels []model.Channel
	query := `WITH RECURSIVE channel_tree AS (
		SELECT * FROM cms_channels WHERE account_id = $1 AND parent_id IS NULL AND deleted_at IS NULL
		UNION ALL
		SELECT c.* FROM cms_channels c INNER JOIN channel_tree ct ON c.parent_id = ct.id
		WHERE c.deleted_at IS NULL
	) SELECT * FROM channel_tree ORDER BY level, sort_order`
	err := r.DB.SelectContext(ctx, &channels, query, accountID)
	if err != nil {
		return nil, err
	}
	return channels, nil
}

func (r *CMSRepoDB) UpdateChannel(ctx context.Context, ch *model.Channel) error {
	query := `UPDATE cms_channels SET
		name = COALESCE(NULLIF($1, ''), name),
		slug = COALESCE($2, slug),
		level = CASE WHEN $3 >= 0 THEN $3 ELSE level END,
		cover_url = COALESCE($4, cover_url),
		description = COALESCE($5, description),
		status = CASE WHEN $6 >= 0 THEN $6 ELSE status END,
		updated_at = NOW()
		WHERE id = $7 AND deleted_at IS NULL`
	result, err := r.DB.ExecContext(ctx, query,
		ch.Name, ch.Slug, ch.Level, ch.CoverURL, ch.Description, ch.Status, ch.ID,
	)
	if err != nil {
		return err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return errors.New("栏目不存在或已删除")
	}
	return nil
}

func (r *CMSRepoDB) SoftDeleteChannel(ctx context.Context, id int64) error {
	query := `UPDATE cms_channels SET deleted_at = NOW() WHERE id = $1 AND deleted_at IS NULL`
	_, err := r.DB.ExecContext(ctx, query, id)
	return err
}

func (r *CMSRepoDB) CreateArticle(ctx context.Context, a *model.Article) error {
	query := `INSERT INTO cms_articles (account_id, channel_id, title, cover_url, summary, author,
		content, html_cache, status, is_template, template_cat, sort_order, view_count, published_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
		RETURNING id, created_at, updated_at`
	row := r.DB.QueryRowContext(ctx, query,
		a.AccountID, a.ChannelID, a.Title, a.CoverURL, a.Summary, a.Author,
		a.Content, a.HTMLCache, a.Status, a.IsTemplate, a.TemplateCat, a.SortOrder, a.ViewCount, a.PublishedAt,
	)
	return row.Scan(&a.ID, &a.CreatedAt, &a.UpdatedAt)
}

func (r *CMSRepoDB) GetArticle(ctx context.Context, id int64) (*model.Article, error) {
	if id <= 0 {
		return nil, errors.New("无效的文章 ID")
	}
	var a model.Article
	query := `SELECT * FROM cms_articles WHERE id = $1 AND deleted_at IS NULL`
	err := r.DB.GetContext(ctx, &a, query, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &a, nil
}

func (r *CMSRepoDB) ListArticles(ctx context.Context, accountID int64, channelID *int64, status *int16, offset, limit int) ([]model.Article, error) {
	if limit <= 0 {
		limit = 20
	}
	if offset < 0 {
		offset = 0
	}
	var articles []model.Article
	var err error
	if channelID != nil && status != nil {
		query := `SELECT * FROM cms_articles WHERE channel_id = $1 AND status = $2 AND deleted_at IS NULL ORDER BY sort_order LIMIT $3 OFFSET $4`
		err = r.DB.SelectContext(ctx, &articles, query, *channelID, *status, limit, offset)
	} else if status != nil {
		query := `SELECT * FROM cms_articles WHERE account_id = $1 AND status = $2 AND deleted_at IS NULL ORDER BY created_at DESC LIMIT $3 OFFSET $4`
		err = r.DB.SelectContext(ctx, &articles, query, accountID, *status, limit, offset)
	} else {
		query := `SELECT * FROM cms_articles WHERE account_id = $1 AND deleted_at IS NULL ORDER BY created_at DESC LIMIT $2 OFFSET $3`
		err = r.DB.SelectContext(ctx, &articles, query, accountID, limit, offset)
	}
	if err != nil {
		return nil, err
	}
	return articles, nil
}

func (r *CMSRepoDB) UpdateArticle(ctx context.Context, a *model.Article) error {
	query := `UPDATE cms_articles SET
		title = COALESCE($1, title),
		cover_url = COALESCE($2, cover_url),
		summary = COALESCE($3, summary),
		author = COALESCE($4, author),
		content = COALESCE($5, content),
		status = CASE WHEN $6 >= 0 THEN $6 ELSE status END,
		is_template = CASE WHEN $7 IS NOT NULL THEN $7 ELSE is_template END,
		published_at = COALESCE($8, published_at),
		updated_at = NOW()
		WHERE id = $9 AND deleted_at IS NULL`
	result, err := r.DB.ExecContext(ctx, query,
		a.Title, a.CoverURL, a.Summary, a.Author, a.Content,
		a.Status, a.IsTemplate, a.PublishedAt, a.ID,
	)
	if err != nil {
		return err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return errors.New("文章不存在或已删除")
	}
	return nil
}

func (r *CMSRepoDB) SoftDeleteArticle(ctx context.Context, id int64) error {
	query := `UPDATE cms_articles SET deleted_at = NOW() WHERE id = $1 AND deleted_at IS NULL`
	_, err := r.DB.ExecContext(ctx, query, id)
	return err
}
