// Package api 投票管理 Handler (T14)
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

// VoteRepo 投票数据访问接口（依赖注入 + 测试 mock）
type VoteRepo interface {
	ListVotes(ctx context.Context, accountID int64, offset, limit int) ([]model.Vote, error)
	GetVote(ctx context.Context, id int64) (*model.Vote, error)
	CreateVote(ctx context.Context, v *model.Vote) error
	UpdateVote(ctx context.Context, v *model.Vote) error
	SoftDeleteVote(ctx context.Context, id int64) error
	// 选项
	GetOptions(ctx context.Context, voteID int64) ([]model.VoteOption, error)
	CreateOption(ctx context.Context, o *model.VoteOption) error
	DeleteOptionsByVoteID(ctx context.Context, voteID int64) error
	// 投票
	SubmitVote(ctx context.Context, record *model.VoteRecord) error
	CountVotesByUser(ctx context.Context, voteID int64, openid string) (int, error)
	// 结果
	GetResults(ctx context.Context, voteID int64) ([]model.VoteOption, error)
}

// VoteHandler 投票管理 API 处理器
type VoteHandler struct {
	voteRepo VoteRepo
	logger   *zap.Logger
}

// NewVoteHandler 创建投票 Handler
func NewVoteHandler(voteRepo VoteRepo, logger *zap.Logger) *VoteHandler {
	return &VoteHandler{
		voteRepo: voteRepo,
		logger:   logger,
	}
}

// ========== 请求/响应结构体 ==========

// CreateVoteRequest 创建投票请求
type CreateVoteRequest struct {
	Title       string              `json:"title"        binding:"required,min=1,max=200"`
	Description *string             `json:"description"`
	CoverURL    *string             `json:"cover_url"`
	VoteType    int16               `json:"vote_type"`
	MaxChoices  int                 `json:"max_choices"`
	MaxVotes    int                 `json:"max_votes"`
	StartTime   *time.Time          `json:"start_time"`
	EndTime     *time.Time          `json:"end_time"`
	Status      int16               `json:"status"`
	Options     []CreateOptionItem  `json:"options"      binding:"required,min=1"`
}

// CreateOptionItem 创建选项项
type CreateOptionItem struct {
	Content   string  `json:"content"    binding:"required,min=1,max=500"`
	ImageURL  *string `json:"image_url"`
	SortOrder int     `json:"sort_order"`
}

// UpdateVoteRequest 更新投票请求
type UpdateVoteRequest struct {
	Title       *string             `json:"title"`
	Description *string             `json:"description"`
	CoverURL    *string             `json:"cover_url"`
	VoteType    *int16              `json:"vote_type"`
	MaxChoices  *int                `json:"max_choices"`
	MaxVotes    *int                `json:"max_votes"`
	StartTime   *time.Time          `json:"start_time"`
	EndTime     *time.Time          `json:"end_time"`
	Status      *int16              `json:"status"`
	Options     []CreateOptionItem  `json:"options,omitempty"`
}

// VoteVO 投票视图对象
type VoteVO struct {
	ID          int64         `json:"id"`
	AccountID   int64         `json:"account_id"`
	Title       string        `json:"title"`
	Description *string       `json:"description"`
	CoverURL    *string       `json:"cover_url"`
	VoteType    int16         `json:"vote_type"`
	MaxChoices  int           `json:"max_choices"`
	MaxVotes    int           `json:"max_votes"`
	StartTime   *time.Time    `json:"start_time"`
	EndTime     *time.Time    `json:"end_time"`
	TotalVotes  int           `json:"total_votes"`
	Status      int16         `json:"status"`
	StatusText  string        `json:"status_text"`
	Options     []VoteOptionVO `json:"options,omitempty"`
	CreatedAt   time.Time     `json:"created_at"`
	UpdatedAt   time.Time     `json:"updated_at"`
}

// VoteOptionVO 选项视图对象
type VoteOptionVO struct {
	ID        int64   `json:"id"`
	VoteID    int64   `json:"vote_id"`
	Content   string  `json:"content"`
	ImageURL  *string `json:"image_url"`
	SortOrder int     `json:"sort_order"`
	VoteCount int     `json:"vote_count"`
}

// VoteResultVO 投票结果
type VoteResultVO struct {
	VoteID    int64          `json:"vote_id"`
	Title     string         `json:"title"`
	Total     int            `json:"total_votes"`
	Options   []VoteOptionVO  `json:"options"`
}

// SubmitVoteRequest H5 投票提交请求
type SubmitVoteRequest struct {
	OptionIDs []int64 `json:"option_ids" binding:"required,min=1"`
	Openid    string  `json:"openid"     binding:"required"`
}

func toVoteVO(v *model.Vote, options []model.VoteOption) VoteVO {
	statusText := "草稿"
	switch v.Status {
	case 0:
		statusText = "草稿"
	case 1:
		statusText = "进行中"
	case 2:
		statusText = "已结束"
	}

	var optionVOs []VoteOptionVO
	for i := range options {
		optionVOs = append(optionVOs, VoteOptionVO{
			ID:        options[i].ID,
			VoteID:    options[i].VoteID,
			Content:   options[i].Content,
			ImageURL:  options[i].ImageURL,
			SortOrder: options[i].SortOrder,
			VoteCount: options[i].VoteCount,
		})
	}

	return VoteVO{
		ID:          v.ID,
		AccountID:   v.AccountID,
		Title:       v.Title,
		Description: v.Description,
		CoverURL:    v.CoverURL,
		VoteType:    v.VoteType,
		MaxChoices:  v.MaxChoices,
		MaxVotes:    v.MaxVotes,
		StartTime:   v.StartTime,
		EndTime:     v.EndTime,
		TotalVotes:  v.TotalVotes,
		Status:      v.Status,
		StatusText:  statusText,
		Options:     optionVOs,
		CreatedAt:   v.CreatedAt,
		UpdatedAt:   v.UpdatedAt,
	}
}

// ========== Tenant helper ==========

func (h *VoteHandler) getTenantID(c *gin.Context) (int64, bool) {
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

// ======================== 投票管理 API ========================

// ListVotes GET /api/v1/votes — 获取投票列表
func (h *VoteHandler) ListVotes(c *gin.Context) {
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

	votes, err := h.voteRepo.ListVotes(c.Request.Context(), tenantID, offset, size)
	if err != nil {
		h.logger.Error("查询投票列表失败", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"code": 50001,
			"msg":  "查询投票列表失败",
			"data": nil,
		})
		return
	}

	var vos []VoteVO
	for i := range votes {
		vos = append(vos, toVoteVO(&votes[i], nil))
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

// CreateVote POST /api/v1/votes — 创建投票
func (h *VoteHandler) CreateVote(c *gin.Context) {
	tenantID, ok := h.getTenantID(c)
	if !ok {
		return
	}

	var req CreateVoteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code": 40001,
			"msg":  "参数错误: " + err.Error(),
			"data": nil,
		})
		return
	}

	if req.VoteType == 0 {
		req.VoteType = 1
	}
	if req.MaxChoices == 0 {
		req.MaxChoices = 1
	}
	if req.MaxVotes == 0 {
		req.MaxVotes = 1
	}

	v := &model.Vote{
		AccountID:   tenantID,
		Title:       req.Title,
		Description: req.Description,
		CoverURL:    req.CoverURL,
		VoteType:    req.VoteType,
		MaxChoices:  req.MaxChoices,
		MaxVotes:    req.MaxVotes,
		StartTime:   req.StartTime,
		EndTime:     req.EndTime,
		Status:      req.Status,
	}

	if err := h.voteRepo.CreateVote(c.Request.Context(), v); err != nil {
		h.logger.Error("创建投票失败", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"code": 50001,
			"msg":  "创建投票失败",
			"data": nil,
		})
		return
	}

	// 创建选项
	for i := range req.Options {
		opt := &model.VoteOption{
			VoteID:    v.ID,
			Content:   req.Options[i].Content,
			ImageURL:  req.Options[i].ImageURL,
			SortOrder: req.Options[i].SortOrder,
		}
		if err := h.voteRepo.CreateOption(c.Request.Context(), opt); err != nil {
			h.logger.Error("创建投票选项失败", zap.Error(err))
			// 投票已创建但选项创建失败，返回部分成功
		}
	}

	// 查询完整数据返回
	options, _ := h.voteRepo.GetOptions(c.Request.Context(), v.ID)

	c.JSON(http.StatusCreated, gin.H{
		"code": 0,
		"msg":  "投票已创建",
		"data": toVoteVO(v, options),
	})
}

// GetVote GET /api/v1/votes/:id — 获取投票详情
func (h *VoteHandler) GetVote(c *gin.Context) {
	_, ok := h.getTenantID(c)
	if !ok {
		return
	}

	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code": 40001,
			"msg":  "无效的投票 ID",
			"data": nil,
		})
		return
	}

	v, err := h.voteRepo.GetVote(c.Request.Context(), id)
	if err != nil {
		h.logger.Error("查询投票失败", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"code": 50001,
			"msg":  "查询投票失败",
			"data": nil,
		})
		return
	}
	if v == nil {
		c.JSON(http.StatusNotFound, gin.H{
			"code": 40401,
			"msg":  "投票不存在",
			"data": nil,
		})
		return
	}

	options, _ := h.voteRepo.GetOptions(c.Request.Context(), id)

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"msg":  "ok",
		"data": toVoteVO(v, options),
	})
}

// UpdateVote PUT /api/v1/votes/:id — 更新投票
func (h *VoteHandler) UpdateVote(c *gin.Context) {
	_, ok := h.getTenantID(c)
	if !ok {
		return
	}

	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code": 40001,
			"msg":  "无效的投票 ID",
			"data": nil,
		})
		return
	}

	var req UpdateVoteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code": 40001,
			"msg":  "参数错误: " + err.Error(),
			"data": nil,
		})
		return
	}

	v := &model.Vote{ID: id}
	if req.Title != nil {
		v.Title = *req.Title
	}
	if req.Description != nil {
		v.Description = req.Description
	}
	if req.CoverURL != nil {
		v.CoverURL = req.CoverURL
	}
	if req.VoteType != nil {
		v.VoteType = *req.VoteType
	}
	if req.MaxChoices != nil {
		v.MaxChoices = *req.MaxChoices
	}
	if req.MaxVotes != nil {
		v.MaxVotes = *req.MaxVotes
	}
	if req.StartTime != nil {
		v.StartTime = req.StartTime
	}
	if req.EndTime != nil {
		v.EndTime = req.EndTime
	}
	if req.Status != nil {
		v.Status = *req.Status
	}

	if err := h.voteRepo.UpdateVote(c.Request.Context(), v); err != nil {
		h.logger.Error("更新投票失败", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"code": 50001,
			"msg":  "更新投票失败: " + err.Error(),
			"data": nil,
		})
		return
	}

	// 如果提供了选项，则删除旧选项并创建新选项
	if len(req.Options) > 0 {
		_ = h.voteRepo.DeleteOptionsByVoteID(c.Request.Context(), id)
		for i := range req.Options {
			opt := &model.VoteOption{
				VoteID:    id,
				Content:   req.Options[i].Content,
				ImageURL:  req.Options[i].ImageURL,
				SortOrder: req.Options[i].SortOrder,
			}
			_ = h.voteRepo.CreateOption(c.Request.Context(), opt)
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"msg":  "投票已更新",
		"data": nil,
	})
}

// DeleteVote DELETE /api/v1/votes/:id — 删除投票
func (h *VoteHandler) DeleteVote(c *gin.Context) {
	_, ok := h.getTenantID(c)
	if !ok {
		return
	}

	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code": 40001,
			"msg":  "无效的投票 ID",
			"data": nil,
		})
		return
	}

	if err := h.voteRepo.SoftDeleteVote(c.Request.Context(), id); err != nil {
		h.logger.Error("删除投票失败", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"code": 50001,
			"msg":  "删除投票失败",
			"data": nil,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"msg":  "投票已删除",
		"data": nil,
	})
}

// GetResults GET /api/v1/votes/:id/results — 获取投票实时结果
func (h *VoteHandler) GetResults(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code": 40001,
			"msg":  "无效的投票 ID",
			"data": nil,
		})
		return
	}

	v, err := h.voteRepo.GetVote(c.Request.Context(), id)
	if err != nil {
		h.logger.Error("查询投票失败", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"code": 50001,
			"msg":  "查询投票失败",
			"data": nil,
		})
		return
	}
	if v == nil {
		c.JSON(http.StatusNotFound, gin.H{
			"code": 40401,
			"msg":  "投票不存在",
			"data": nil,
		})
		return
	}

	options, err := h.voteRepo.GetResults(c.Request.Context(), id)
	if err != nil {
		h.logger.Error("查询投票结果失败", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"code": 50001,
			"msg":  "查询投票结果失败",
			"data": nil,
		})
		return
	}

	var optionVOs []VoteOptionVO
	total := 0
	for i := range options {
		optionVOs = append(optionVOs, VoteOptionVO{
			ID:        options[i].ID,
			VoteID:    options[i].VoteID,
			Content:   options[i].Content,
			ImageURL:  options[i].ImageURL,
			SortOrder: options[i].SortOrder,
			VoteCount: options[i].VoteCount,
		})
		total += options[i].VoteCount
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"msg":  "ok",
		"data": VoteResultVO{
			VoteID:    v.ID,
			Title:     v.Title,
			Total:     total,
			Options:   optionVOs,
		},
	})
}

// SubmitVote POST /h5/vote/:id/submit — 用户投票（H5，需从 OAuth 获取 openid）
func (h *VoteHandler) SubmitVote(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code": 40001,
			"msg":  "无效的投票 ID",
			"data": nil,
		})
		return
	}

	var req SubmitVoteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code": 40001,
			"msg":  "参数错误: " + err.Error(),
			"data": nil,
		})
		return
	}

	// 获取投票信息以验证
	v, err := h.voteRepo.GetVote(c.Request.Context(), id)
	if err != nil {
		h.logger.Error("查询投票失败", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"code": 50001,
			"msg":  "查询投票失败",
			"data": nil,
		})
		return
	}
	if v == nil {
		c.JSON(http.StatusNotFound, gin.H{
			"code": 40401,
			"msg":  "投票不存在",
			"data": nil,
		})
		return
	}

	// 验证投票状态
	if v.Status != 1 {
		c.JSON(http.StatusBadRequest, gin.H{
			"code": 40001,
			"msg":  "投票不在进行中",
			"data": nil,
		})
		return
	}

	// 验证时间范围
	now := time.Now()
	if v.StartTime != nil && now.Before(*v.StartTime) {
		c.JSON(http.StatusBadRequest, gin.H{
			"code": 40001,
			"msg":  "投票尚未开始",
			"data": nil,
		})
		return
	}
	if v.EndTime != nil && now.After(*v.EndTime) {
		c.JSON(http.StatusBadRequest, gin.H{
			"code": 40001,
			"msg":  "投票已结束",
			"data": nil,
		})
		return
	}

	// 验证多选限制
	if v.VoteType == 1 && len(req.OptionIDs) > 1 {
		c.JSON(http.StatusBadRequest, gin.H{
			"code": 40001,
			"msg":  "单选投票只能选择一个选项",
			"data": nil,
		})
		return
	}
	if v.MaxChoices > 0 && len(req.OptionIDs) > v.MaxChoices {
		c.JSON(http.StatusBadRequest, gin.H{
			"code": 40001,
			"msg":  "最多可选" + strconv.Itoa(v.MaxChoices) + "项",
			"data": nil,
		})
		return
	}

	// 验证投票次数限制
	if v.MaxVotes > 0 {
		count, err := h.voteRepo.CountVotesByUser(c.Request.Context(), id, req.Openid)
		if err != nil {
			h.logger.Error("查询用户投票次数失败", zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{
				"code": 50001,
				"msg":  "查询投票次数失败",
				"data": nil,
			})
			return
		}
		if count >= v.MaxVotes {
			c.JSON(http.StatusBadRequest, gin.H{
				"code": 40001,
				"msg":  "已达到最大投票次数",
				"data": nil,
			})
			return
		}
	}

	// 记录投票
	for _, optID := range req.OptionIDs {
		record := &model.VoteRecord{
			VoteID:   id,
			OptionID: optID,
			Openid:   req.Openid,
		}
		if err := h.voteRepo.SubmitVote(c.Request.Context(), record); err != nil {
			h.logger.Error("提交投票失败", zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{
				"code": 50001,
				"msg":  "提交投票失败",
				"data": nil,
			})
			return
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"msg":  "投票成功",
		"data": nil,
	})
}

// ========== VoteRepo PostgreSQL 实现 ==========

// voteRepoImpl VoteRepo 的 PostgreSQL 实现
type voteRepoImpl struct {
	DB *sqlx.DB
}

// NewVoteRepo 创建投票 Repository
func NewVoteRepo(db *sqlx.DB) *voteRepoImpl {
	return &voteRepoImpl{DB: db}
}

func (r *voteRepoImpl) ListVotes(ctx context.Context, accountID int64, offset, limit int) ([]model.Vote, error) {
	var votes []model.Vote
	query := `SELECT * FROM votes WHERE account_id = $1 AND deleted_at IS NULL ORDER BY created_at DESC LIMIT $2 OFFSET $3`
	err := r.DB.SelectContext(ctx, &votes, query, accountID, limit, offset)
	return votes, err
}

func (r *voteRepoImpl) GetVote(ctx context.Context, id int64) (*model.Vote, error) {
	if id <= 0 {
		return nil, errors.New("无效的投票 ID")
	}
	var v model.Vote
	query := `SELECT * FROM votes WHERE id = $1 AND deleted_at IS NULL`
	err := r.DB.GetContext(ctx, &v, query, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &v, nil
}

func (r *voteRepoImpl) CreateVote(ctx context.Context, v *model.Vote) error {
	query := `INSERT INTO votes (account_id, title, description, cover_url, vote_type, max_choices, max_votes, start_time, end_time, total_votes, status)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		RETURNING id, created_at, updated_at`
	row := r.DB.QueryRowContext(ctx, query,
		v.AccountID, v.Title, v.Description, v.CoverURL, v.VoteType,
		v.MaxChoices, v.MaxVotes, v.StartTime, v.EndTime, v.TotalVotes, v.Status,
	)
	return row.Scan(&v.ID, &v.CreatedAt, &v.UpdatedAt)
}

func (r *voteRepoImpl) UpdateVote(ctx context.Context, v *model.Vote) error {
	query := `UPDATE votes SET
		title = COALESCE(NULLIF($1, ''), title),
		description = COALESCE($2, description),
		cover_url = COALESCE($3, cover_url),
		vote_type = COALESCE(NULLIF($4, 0), vote_type),
		max_choices = COALESCE(NULLIF($5, 0), max_choices),
		max_votes = COALESCE(NULLIF($6, 0), max_votes),
		start_time = COALESCE($7, start_time),
		end_time = COALESCE($8, end_time),
		status = COALESCE(NULLIF($9, -1), status),
		updated_at = NOW()
		WHERE id = $10 AND deleted_at IS NULL`
	result, err := r.DB.ExecContext(ctx, query,
		v.Title, v.Description, v.CoverURL, v.VoteType,
		v.MaxChoices, v.MaxVotes, v.StartTime, v.EndTime,
		v.Status, v.ID,
	)
	if err != nil {
		return err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return errors.New("投票不存在或已删除")
	}
	return nil
}

func (r *voteRepoImpl) SoftDeleteVote(ctx context.Context, id int64) error {
	query := `UPDATE votes SET deleted_at = NOW() WHERE id = $1 AND deleted_at IS NULL`
	_, err := r.DB.ExecContext(ctx, query, id)
	return err
}

func (r *voteRepoImpl) GetOptions(ctx context.Context, voteID int64) ([]model.VoteOption, error) {
	var options []model.VoteOption
	query := `SELECT * FROM vote_options WHERE vote_id = $1 ORDER BY sort_order`
	err := r.DB.SelectContext(ctx, &options, query, voteID)
	return options, err
}

func (r *voteRepoImpl) CreateOption(ctx context.Context, o *model.VoteOption) error {
	query := `INSERT INTO vote_options (vote_id, content, image_url, sort_order, vote_count)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id`
	return r.DB.QueryRowContext(ctx, query, o.VoteID, o.Content, o.ImageURL, o.SortOrder, o.VoteCount).Scan(&o.ID)
}

func (r *voteRepoImpl) DeleteOptionsByVoteID(ctx context.Context, voteID int64) error {
	query := `DELETE FROM vote_options WHERE vote_id = $1`
	_, err := r.DB.ExecContext(ctx, query, voteID)
	return err
}

func (r *voteRepoImpl) SubmitVote(ctx context.Context, record *model.VoteRecord) error {
	// 插入投票记录
	query := `INSERT INTO vote_records (vote_id, option_id, openid, ip_address, user_agent)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, created_at`
	err := r.DB.QueryRowContext(ctx, query,
		record.VoteID, record.OptionID, record.Openid, record.IPAddress, record.UserAgent,
	).Scan(&record.ID, &record.CreatedAt)
	if err != nil {
		return err
	}

	// 更新选项计数
	updateOpt := `UPDATE vote_options SET vote_count = vote_count + 1 WHERE id = $1`
	_, _ = r.DB.ExecContext(ctx, updateOpt, record.OptionID)

	// 更新总投票人数
	updateVote := `UPDATE votes SET total_votes = total_votes + 1 WHERE id = $1`
	_, _ = r.DB.ExecContext(ctx, updateVote, record.VoteID)

	return nil
}

func (r *voteRepoImpl) CountVotesByUser(ctx context.Context, voteID int64, openid string) (int, error) {
	var count int
	query := `SELECT COUNT(*) FROM vote_records WHERE vote_id = $1 AND openid = $2`
	err := r.DB.GetContext(ctx, &count, query, voteID, openid)
	return count, err
}

func (r *voteRepoImpl) GetResults(ctx context.Context, voteID int64) ([]model.VoteOption, error) {
	var options []model.VoteOption
	query := `SELECT * FROM vote_options WHERE vote_id = $1 ORDER BY vote_count DESC, sort_order`
	err := r.DB.SelectContext(ctx, &options, query, voteID)
	return options, err
}
