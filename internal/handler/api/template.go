// Package api 模板系统 Handler (T10)
package api

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"

	"github.com/weiyeston/weiyeston-v2/internal/model"
)

// TemplateHandler 模板管理 API 处理器
// 使用函数变量模式方便测试时 Mock
type TemplateHandler struct {
	selectFunc func(ctx context.Context, dest interface{}, query string, args ...interface{}) error
	saveFunc   func(ctx context.Context, query string, args ...interface{}) (int64, time.Time, time.Time, error)
}

// TemplateHandlerOption 可选的构造函数配置
type TemplateHandlerOption func(*TemplateHandler)

// WithSelectFunc 注入自定义 SelectFunc（测试用）
func WithSelectFunc(fn func(ctx context.Context, dest interface{}, query string, args ...interface{}) error) TemplateHandlerOption {
	return func(h *TemplateHandler) {
		h.selectFunc = fn
	}
}

// WithSaveFunc 注入自定义 SaveFunc（测试用）
func WithSaveFunc(fn func(ctx context.Context, query string, args ...interface{}) (int64, time.Time, time.Time, error)) TemplateHandlerOption {
	return func(h *TemplateHandler) {
		h.saveFunc = fn
	}
}

// NewTemplateHandler 创建模板 Handler
func NewTemplateHandler(db *sqlx.DB, opts ...TemplateHandlerOption) *TemplateHandler {
	h := &TemplateHandler{
		selectFunc: db.SelectContext,
		saveFunc: func(ctx context.Context, query string, args ...interface{}) (int64, time.Time, time.Time, error) {
			row := db.QueryRowContext(ctx, query, args...)
			var id int64
			var createdAt, updatedAt time.Time
			if err := row.Scan(&id, &createdAt, &updatedAt); err != nil {
				return 0, time.Time{}, time.Time{}, err
			}
			return id, createdAt, updatedAt, nil
		},
	}
	for _, opt := range opts {
		opt(h)
	}
	return h
}

// ========== 请求/响应结构体 ==========

// TemplateVO 返回给前端的模板视图对象
type TemplateVO struct {
	ID          int64           `json:"id"`
	Title       string          `json:"title"`
	CoverURL    string          `json:"cover_url"`
	Summary     string          `json:"summary"`
	TemplateCat string          `json:"template_cat"`
	Content     json.RawMessage `json:"content"`
	Author      string          `json:"author"`
}

// SaveTemplateRequest 保存模板请求
type SaveTemplateRequest struct {
	Title       string          `json:"title" binding:"required"`
	CoverURL    string          `json:"cover_url"`
	Summary     string          `json:"summary"`
	Content     json.RawMessage `json:"content" binding:"required"`
	TemplateCat string          `json:"template_cat"`
	Author      string          `json:"author"`
}

func toTemplateVO(a *model.Article) TemplateVO {
	title := ""
	if a.Title != nil {
		title = *a.Title
	}
	coverURL := ""
	if a.CoverURL != nil {
		coverURL = *a.CoverURL
	}
	summary := ""
	if a.Summary != nil {
		summary = *a.Summary
	}
	templateCat := ""
	if a.TemplateCat != nil {
		templateCat = *a.TemplateCat
	}
	author := ""
	if a.Author != nil {
		author = *a.Author
	}

	return TemplateVO{
		ID:          a.ID,
		Title:       title,
		CoverURL:    coverURL,
		Summary:     summary,
		TemplateCat: templateCat,
		Content:     a.Content,
		Author:      author,
	}
}

func toTemplateVOs(articles []model.Article) []TemplateVO {
	result := make([]TemplateVO, 0, len(articles))
	for i := range articles {
		result = append(result, toTemplateVO(&articles[i]))
	}
	return result
}

// ========== Handler 方法 ==========

// ListSystemTemplates GET /api/v1/templates?category=xxx
// 返回系统模板列表（is_template=true 的文章）
func (h *TemplateHandler) ListSystemTemplates(c *gin.Context) {
	category := c.Query("category")

	query := `SELECT id, account_id, channel_id, title, cover_url, summary, author,
		content, html_cache, status, is_template, template_cat, sort_order,
		view_count, published_at, deleted_at, created_at, updated_at
		FROM cms_articles
		WHERE is_template = true AND deleted_at IS NULL`
	args := []interface{}{}

	if category != "" {
		query += ` AND template_cat = $1`
		args = append(args, category)
	}

	query += ` ORDER BY created_at DESC`

	var articles []model.Article
	err := h.selectFunc(c.Request.Context(), &articles, query, args...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code": 50001,
			"msg":  "查询模板列表失败",
			"data": nil,
		})
		return
	}

	if articles == nil {
		articles = make([]model.Article, 0)
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"msg":  "ok",
		"data": toTemplateVOs(articles),
	})
}

// SaveTemplate POST /api/v1/templates
// 保存用户自定义模板
func (h *TemplateHandler) SaveTemplate(c *gin.Context) {
	var req SaveTemplateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code": 40001,
			"msg":  "请求参数不合法: " + err.Error(),
			"data": nil,
		})
		return
	}

	// Get account_id from tenant context
	accountID, exists := c.Get("tenant_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"code": 401,
			"msg":  "未授权访问",
			"data": nil,
		})
		return
	}

	var aid int64
	switch v := accountID.(type) {
	case int64:
		aid = v
	case float64:
		aid = int64(v)
	default:
		c.JSON(http.StatusBadRequest, gin.H{
			"code": 40001,
			"msg":  "无效的租户信息",
			"data": nil,
		})
		return
	}

	query := `INSERT INTO cms_articles (account_id, title, cover_url, summary, author, content,
		status, is_template, template_cat)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING id, created_at, updated_at`

	var titlePtr, coverPtr, summaryPtr, authorPtr, catPtr *string
	title := req.Title
	titlePtr = &title
	if req.CoverURL != "" {
		s := req.CoverURL
		coverPtr = &s
	}
	if req.Summary != "" {
		s := req.Summary
		summaryPtr = &s
	}
	if req.Author != "" {
		s := req.Author
		authorPtr = &s
	}
	if req.TemplateCat != "" {
		s := req.TemplateCat
		catPtr = &s
	}

	args := []interface{}{aid, titlePtr, coverPtr, summaryPtr, authorPtr, req.Content, int16(1), true, catPtr}

	id, createdAt, updatedAt, err := h.saveFunc(c.Request.Context(), query, args...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code": 50001,
			"msg":  "保存模板失败: " + err.Error(),
			"data": nil,
		})
		return
	}

	article := &model.Article{
		ID:          id,
		AccountID:   aid,
		Title:       titlePtr,
		CoverURL:    coverPtr,
		Summary:     summaryPtr,
		Author:      authorPtr,
		TemplateCat: catPtr,
		Content:     req.Content,
		Status:      1,
		IsTemplate:  true,
		CreatedAt:   createdAt,
		UpdatedAt:   updatedAt,
	}

	c.JSON(http.StatusCreated, gin.H{
		"code": 0,
		"msg":  "模板保存成功",
		"data": toTemplateVO(article),
	})
}
