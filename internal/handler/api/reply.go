// Package api 自动回复规则管理 Handler
package api

import (
	"context"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/weiyeston/weiyeston-v2/internal/model"
)

// ReplyRepo 自动回复数据访问接口（依赖注入 + 测试 mock）
type ReplyRepo interface {
	ListByAccountID(ctx context.Context, accountID int64) ([]model.AutoReplyRule, error)
	GetByID(ctx context.Context, id int64) (*model.AutoReplyRule, error)
	Create(ctx context.Context, rule *model.AutoReplyRule) error
	Update(ctx context.Context, rule *model.AutoReplyRule) error
	SoftDelete(ctx context.Context, id int64) (bool, error)
}

// ReplyHandler 自动回复规则管理 API 处理器
type ReplyHandler struct {
	replyRepo ReplyRepo
	logger    *zap.Logger
}

// NewReplyHandler 创建回复规则 Handler
func NewReplyHandler(replyRepo ReplyRepo, logger *zap.Logger) *ReplyHandler {
	return &ReplyHandler{
		replyRepo: replyRepo,
		logger:    logger,
	}
}

// ========== 请求/响应结构体 ==========

// CreateReplyRequest 创建回复规则请求体
type CreateReplyRequest struct {
	Keyword       *string `json:"keyword"`
	MatchType     int16   `json:"match_type"`
	ReplyType     int16   `json:"reply_type"     binding:"required,min=1,max=2"`
	ReplyContent  string  `json:"reply_content"  binding:"required"`
	ReplyTitle    *string `json:"reply_title"`
	ReplyDesc     *string `json:"reply_desc"`
	ReplyCoverURL *string `json:"reply_cover_url"`
	ReplyURL      *string `json:"reply_url"`
	Status        int16   `json:"status"`
	SortOrder     int     `json:"sort_order"`
}

// UpdateReplyRequest 更新回复规则请求体
type UpdateReplyRequest struct {
	Keyword       *string `json:"keyword"`
	MatchType     *int16  `json:"match_type"`
	ReplyType     *int16  `json:"reply_type"     binding:"omitempty,min=1,max=2"`
	ReplyContent  *string `json:"reply_content"`
	ReplyTitle    *string `json:"reply_title"`
	ReplyDesc     *string `json:"reply_desc"`
	ReplyCoverURL *string `json:"reply_cover_url"`
	ReplyURL      *string `json:"reply_url"`
	Status        *int16  `json:"status"         binding:"omitempty,min=0,max=1"`
	SortOrder     *int    `json:"sort_order"`
}

// ReplyVO 返回给前端的回复规则视图对象
type ReplyVO struct {
	ID            int64   `json:"id"`
	AccountID     int64   `json:"account_id"`
	Keyword       *string `json:"keyword"`
	MatchType     int16   `json:"match_type"`
	ReplyType     int16   `json:"reply_type"`
	ReplyContent  string  `json:"reply_content"`
	ReplyTitle    *string `json:"reply_title"`
	ReplyDesc     *string `json:"reply_desc"`
	ReplyCoverURL *string `json:"reply_cover_url"`
	ReplyURL      *string `json:"reply_url"`
	Status        int16   `json:"status"`
	SortOrder     int     `json:"sort_order"`
}

// toReplyVO converts model.AutoReplyRule to ReplyVO
func toReplyVO(r *model.AutoReplyRule) ReplyVO {
	return ReplyVO{
		ID:            r.ID,
		AccountID:     r.AccountID,
		Keyword:       r.Keyword,
		MatchType:     r.MatchType,
		ReplyType:     r.ReplyType,
		ReplyContent:  r.ReplyContent,
		ReplyTitle:    r.ReplyTitle,
		ReplyDesc:     r.ReplyDesc,
		ReplyCoverURL: r.ReplyCoverURL,
		ReplyURL:      r.ReplyURL,
		Status:        r.Status,
		SortOrder:     r.SortOrder,
	}
}

// ========== Handler 方法 ==========

// List GET /api/v1/accounts/:id/replies — 回复规则列表
func (h *ReplyHandler) List(c *gin.Context) {
	accountID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code": 40001,
			"msg":  "无效的公众号 ID",
			"data": nil,
		})
		return
	}

	// 获取 tenant_id 校验
	tenantID, ok := h.getTenantID(c)
	if !ok {
		return
	}
	_ = tenantID // 后续可按租户校验公众号归属

	rules, err := h.replyRepo.ListByAccountID(c.Request.Context(), accountID)
	if err != nil {
		h.logger.Error("查询回复规则列表失败", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"code": 50001,
			"msg":  "查询回复规则列表失败",
			"data": nil,
		})
		return
	}

	list := make([]ReplyVO, 0, len(rules))
	for i := range rules {
		list = append(list, toReplyVO(&rules[i]))
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"msg":  "ok",
		"data": list,
	})
}

// Create POST /api/v1/accounts/:id/replies — 创建回复规则
func (h *ReplyHandler) Create(c *gin.Context) {
	accountID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code": 40001,
			"msg":  "无效的公众号 ID",
			"data": nil,
		})
		return
	}

	tenantID, ok := h.getTenantID(c)
	if !ok {
		return
	}
	_ = tenantID

	var req CreateReplyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code": 40001,
			"msg":  "参数错误: " + err.Error(),
			"data": nil,
		})
		return
	}

	rule := &model.AutoReplyRule{
		AccountID:     accountID,
		Keyword:       req.Keyword,
		MatchType:     req.MatchType,
		ReplyType:     req.ReplyType,
		ReplyContent:  req.ReplyContent,
		ReplyTitle:    req.ReplyTitle,
		ReplyDesc:     req.ReplyDesc,
		ReplyCoverURL: req.ReplyCoverURL,
		ReplyURL:      req.ReplyURL,
		Status:        req.Status,
		SortOrder:     req.SortOrder,
	}

	if rule.Status == 0 {
		rule.Status = 1 // 默认启用
	}

	if err := h.replyRepo.Create(c.Request.Context(), rule); err != nil {
		h.logger.Error("创建回复规则失败", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"code": 50001,
			"msg":  "创建回复规则失败",
			"data": nil,
		})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"code": 0,
		"msg":  "ok",
		"data": toReplyVO(rule),
	})
}

// Update PUT /api/v1/replies/:id — 编辑回复规则
func (h *ReplyHandler) Update(c *gin.Context) {
	ruleID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code": 40001,
			"msg":  "无效的规则 ID",
			"data": nil,
		})
		return
	}

	tenantID, ok := h.getTenantID(c)
	if !ok {
		return
	}
	_ = tenantID

	// 查询现有规则
	rule, err := h.replyRepo.GetByID(c.Request.Context(), ruleID)
	if err != nil {
		h.logger.Error("查询回复规则失败", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"code": 50001,
			"msg":  "查询回复规则失败",
			"data": nil,
		})
		return
	}
	if rule == nil {
		c.JSON(http.StatusNotFound, gin.H{
			"code": 40401,
			"msg":  "回复规则不存在",
			"data": nil,
		})
		return
	}

	var req UpdateReplyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code": 40001,
			"msg":  "参数错误: " + err.Error(),
			"data": nil,
		})
		return
	}

	// 更新字段
	if req.Keyword != nil {
		rule.Keyword = req.Keyword
	}
	if req.MatchType != nil {
		rule.MatchType = *req.MatchType
	}
	if req.ReplyType != nil {
		rule.ReplyType = *req.ReplyType
	}
	if req.ReplyContent != nil {
		rule.ReplyContent = *req.ReplyContent
	}
	if req.ReplyTitle != nil {
		rule.ReplyTitle = req.ReplyTitle
	}
	if req.ReplyDesc != nil {
		rule.ReplyDesc = req.ReplyDesc
	}
	if req.ReplyCoverURL != nil {
		rule.ReplyCoverURL = req.ReplyCoverURL
	}
	if req.ReplyURL != nil {
		rule.ReplyURL = req.ReplyURL
	}
	if req.Status != nil {
		rule.Status = *req.Status
	}
	if req.SortOrder != nil {
		rule.SortOrder = *req.SortOrder
	}

	if err := h.replyRepo.Update(c.Request.Context(), rule); err != nil {
		h.logger.Error("更新回复规则失败", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"code": 50001,
			"msg":  "更新回复规则失败",
			"data": nil,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"msg":  "ok",
		"data": toReplyVO(rule),
	})
}

// Delete DELETE /api/v1/replies/:id — 删除回复规则
func (h *ReplyHandler) Delete(c *gin.Context) {
	ruleID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code": 40001,
			"msg":  "无效的规则 ID",
			"data": nil,
		})
		return
	}

	tenantID, ok := h.getTenantID(c)
	if !ok {
		return
	}
	_ = tenantID

	deleted, err := h.replyRepo.SoftDelete(c.Request.Context(), ruleID)
	if err != nil {
		h.logger.Error("删除回复规则失败", zap.Error(err))
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
			"msg":  "回复规则不存在或已删除",
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

// getTenantID extract tenant_id from context (shared helper)
func (h *ReplyHandler) getTenantID(c *gin.Context) (int64, bool) {
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
