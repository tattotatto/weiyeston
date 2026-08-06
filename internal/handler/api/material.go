// Package api 素材管理 Handler
package api

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"
	"go.uber.org/zap"

	"github.com/weiyeston/weiyeston-v2/internal/model"
	"github.com/weiyeston/weiyeston-v2/internal/storage"
)

// MaterialRepo 素材数据访问接口（依赖注入 + 测试 mock）
type MaterialRepo interface {
	ListByAccount(ctx context.Context, accountID int64, materialType string, offset, limit int) ([]model.Material, error)
	GetByID(ctx context.Context, id int64) (*model.Material, error)
	Create(ctx context.Context, m *model.Material) error
	SoftDelete(ctx context.Context, id int64) (bool, error)
	CountByAccount(ctx context.Context, accountID int64, materialType string) (int, error)
}

// materialRepoImpl MaterialRepo 的 PostgreSQL 实现
type materialRepoImpl struct {
	Repo interface {
		ListByAccount(ctx context.Context, accountID int64, materialType string, offset, limit int) ([]model.Material, error)
		GetByID(ctx context.Context, id int64) (*model.Material, error)
		Create(ctx context.Context, m *model.Material) error
		SoftDelete(ctx context.Context, id int64) (bool, error)
		CountByAccount(ctx context.Context, accountID int64, materialType string) (int, error)
	}
}

// materialRepoAdapter adapts the material.Repo to the handler's MaterialRepo interface
type materialRepoAdapter struct {
	DB *sqlx.DB
}

func (a *materialRepoAdapter) ListByAccount(ctx context.Context, accountID int64, materialType string, offset, limit int) ([]model.Material, error) {
	if accountID <= 0 {
		return nil, fmt.Errorf("无效的公众号 ID: %d", accountID)
	}
	var materials []model.Material
	query := `SELECT * FROM materials WHERE account_id = $1 AND deleted_at IS NULL`
	args := []interface{}{accountID}

	if materialType != "" {
		query += ` AND type = $2`
		args = append(args, materialType)
		query += ` ORDER BY created_at DESC LIMIT $3 OFFSET $4`
		args = append(args, limit, offset)
	} else {
		query += ` ORDER BY created_at DESC LIMIT $2 OFFSET $3`
		args = append(args, limit, offset)
	}

	err := a.DB.SelectContext(ctx, &materials, query, args...)
	if err != nil {
		return nil, err
	}
	if materials == nil {
		materials = make([]model.Material, 0)
	}
	return materials, nil
}

func (a *materialRepoAdapter) GetByID(ctx context.Context, id int64) (*model.Material, error) {
	if id <= 0 {
		return nil, fmt.Errorf("无效的素材 ID: %d", id)
	}
	var m model.Material
	query := `SELECT * FROM materials WHERE id = $1 AND deleted_at IS NULL`
	err := a.DB.GetContext(ctx, &m, query, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &m, nil
}

func (a *materialRepoAdapter) Create(ctx context.Context, m *model.Material) error {
	query := `INSERT INTO materials (account_id, media_id, type, name, url, thumbnail_url,
		file_size, width, height, format)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		RETURNING id, created_at, updated_at`
	row := a.DB.QueryRowContext(ctx, query,
		m.AccountID, m.MediaID, m.Type, m.Name, m.URL,
		m.ThumbnailURL, m.FileSize, m.Width, m.Height, m.Format,
	)
	return row.Scan(&m.ID, &m.CreatedAt, &m.UpdatedAt)
}

func (a *materialRepoAdapter) SoftDelete(ctx context.Context, id int64) (bool, error) {
	query := `UPDATE materials SET deleted_at = NOW() WHERE id = $1 AND deleted_at IS NULL`
	result, err := a.DB.ExecContext(ctx, query, id)
	if err != nil {
		return false, err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return false, err
	}
	return rows > 0, nil
}

func (a *materialRepoAdapter) CountByAccount(ctx context.Context, accountID int64, materialType string) (int, error) {
	if accountID <= 0 {
		return 0, fmt.Errorf("无效的公众号 ID: %d", accountID)
	}
	var count int
	query := `SELECT COUNT(*) FROM materials WHERE account_id = $1 AND deleted_at IS NULL`
	args := []interface{}{accountID}

	if materialType != "" {
		query += ` AND type = $2`
		args = append(args, materialType)
	}

	err := a.DB.GetContext(ctx, &count, query, args...)
	if err != nil {
		return 0, err
	}
	return count, nil
}

// NewMaterialRepo 创建素材 Repository 适配器
func NewMaterialRepo(db *sqlx.DB) MaterialRepo {
	return &materialRepoAdapter{DB: db}
}

// MaterialHandler 素材管理 API 处理器
type MaterialHandler struct {
	materialRepo   MaterialRepo
	storageProvider storage.Provider
	logger         *zap.Logger
}

// NewMaterialHandler 创建素材 Handler
func NewMaterialHandler(materialRepo MaterialRepo, storageProvider storage.Provider, logger *zap.Logger) *MaterialHandler {
	return &MaterialHandler{
		materialRepo:   materialRepo,
		storageProvider: storageProvider,
		logger:         logger,
	}
}

// ========== 请求/响应结构体 ==========

// MaterialVO 返回给前端的素材视图对象
type MaterialVO struct {
	ID           int64      `json:"id"`
	AccountID    int64      `json:"account_id"`
	MediaID      *string    `json:"media_id"`
	Type         string     `json:"type"`
	Name         *string    `json:"name"`
	URL          string     `json:"url"`
	ThumbnailURL *string    `json:"thumbnail_url"`
	FileSize     *int64     `json:"file_size"`
	Width        *int       `json:"width"`
	Height       *int       `json:"height"`
	Format       *string    `json:"format"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
}

// MaterialListResponse 素材列表分页响应
type MaterialListResponse struct {
	List     []MaterialVO `json:"list"`
	Total    int          `json:"total"`
	Page     int          `json:"page"`
	PageSize int          `json:"page_size"`
}

func toMaterialVO(m *model.Material) MaterialVO {
	return MaterialVO{
		ID:           m.ID,
		AccountID:    m.AccountID,
		MediaID:      m.MediaID,
		Type:         m.Type,
		Name:         m.Name,
		URL:          m.URL,
		ThumbnailURL: m.ThumbnailURL,
		FileSize:     m.FileSize,
		Width:        m.Width,
		Height:       m.Height,
		Format:       m.Format,
		CreatedAt:    m.CreatedAt,
		UpdatedAt:    m.UpdatedAt,
	}
}

func toMaterialVOs(materials []model.Material) []MaterialVO {
	result := make([]MaterialVO, 0, len(materials))
	for i := range materials {
		result = append(result, toMaterialVO(&materials[i]))
	}
	return result
}

// ========== 允许的文件类型 ==========

var allowedExtensions = map[string]string{
	".jpg":  "image",
	".jpeg": "image",
	".png":  "image",
	".gif":  "image",
	".bmp":  "image",
	".webp": "image",
	".svg":  "image",
	".mp3":  "voice",
	".wav":  "voice",
	".amr":  "voice",
	".mp4":  "video",
	".pdf":  "file",
	".doc":  "file",
	".docx": "file",
	".xls":  "file",
	".xlsx": "file",
	".ppt":  "file",
	".pptx": "file",
	".txt":  "file",
	".zip":  "file",
}

// ========== Handler 方法 ==========

// getTenantID 从 context 提取 tenant_id（中间件注入）
func (h *MaterialHandler) getTenantID(c *gin.Context) (int64, bool) {
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

// List GET /api/v1/materials?account_id=&type=&page=&size=
func (h *MaterialHandler) List(c *gin.Context) {
	accountIDStr := c.Query("account_id")
	accountID, err := strconv.ParseInt(accountIDStr, 10, 64)
	if err != nil || accountID <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"code": 40001,
			"msg":  "无效的公众号 ID",
			"data": nil,
		})
		return
	}

	materialType := c.Query("type")

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	size, _ := strconv.Atoi(c.DefaultQuery("size", "20"))
	if page < 1 {
		page = 1
	}
	if size < 1 || size > 100 {
		size = 20
	}
	offset := (page - 1) * size

	total, err := h.materialRepo.CountByAccount(c.Request.Context(), accountID, materialType)
	if err != nil {
		h.logger.Error("统计素材数量失败", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"code": 50001,
			"msg":  "查询失败",
			"data": nil,
		})
		return
	}

	materials, err := h.materialRepo.ListByAccount(c.Request.Context(), accountID, materialType, offset, size)
	if err != nil {
		h.logger.Error("查询素材列表失败", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"code": 50001,
			"msg":  "查询失败",
			"data": nil,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"msg":  "ok",
		"data": MaterialListResponse{
			List:     toMaterialVOs(materials),
			Total:    total,
			Page:     page,
			PageSize: size,
		},
	})
}

// Upload POST /api/v1/materials/upload — multipart form 上传
func (h *MaterialHandler) Upload(c *gin.Context) {
	accountIDStr := c.PostForm("account_id")
	accountID, err := strconv.ParseInt(accountIDStr, 10, 64)
	if err != nil || accountID <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"code": 40001,
			"msg":  "无效的公众号 ID",
			"data": nil,
		})
		return
	}

	file, header, err := c.Request.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code": 40001,
			"msg":  "请选择要上传的文件",
			"data": nil,
		})
		return
	}
	defer file.Close()

	// 校验文件类型
	ext := strings.ToLower(filepath.Ext(header.Filename))
	materialType, ok := allowedExtensions[ext]
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{
			"code": 40001,
			"msg":  "不支持的文件类型: " + ext,
			"data": nil,
		})
		return
	}

	// 生成唯一文件名：时间戳_原始文件名
	uniqueName := fmt.Sprintf("%d_%s", time.Now().UnixNano(), header.Filename)

	// 通过存储驱动上传文件
	contentType := header.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	accessURL, err := h.storageProvider.Upload(c.Request.Context(), uniqueName, file, contentType)
	if err != nil {
		h.logger.Error("上传文件失败", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"code": 50001,
			"msg":  "保存文件失败",
			"data": nil,
		})
		return
	}

	// 构建 DB 记录
	fileName := header.Filename
	fileSize := header.Size
	formatStr := strings.TrimPrefix(ext, ".")

	material := &model.Material{
		AccountID: accountID,
		Type:      materialType,
		Name:      &fileName,
		URL:       accessURL,
		FileSize:  &fileSize,
		Format:    &formatStr,
	}

	if err := h.materialRepo.Create(c.Request.Context(), material); err != nil {
		h.logger.Error("保存素材记录失败", zap.Error(err))
		// 清理已上传的文件
		if delErr := h.storageProvider.Delete(c.Request.Context(), uniqueName); delErr != nil {
			h.logger.Error("清理已上传文件失败", zap.Error(delErr))
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"code": 50001,
			"msg":  "保存素材记录失败",
			"data": nil,
		})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"code": 0,
		"msg":  "上传成功",
		"data": toMaterialVO(material),
	})
}

// GetByID GET /api/v1/materials/:id
func (h *MaterialHandler) GetByID(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"code": 40001,
			"msg":  "无效的素材 ID",
			"data": nil,
		})
		return
	}

	material, err := h.materialRepo.GetByID(c.Request.Context(), id)
	if err != nil {
		h.logger.Error("查询素材失败", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"code": 50001,
			"msg":  "查询失败",
			"data": nil,
		})
		return
	}
	if material == nil {
		c.JSON(http.StatusNotFound, gin.H{
			"code": 40401,
			"msg":  "素材不存在",
			"data": nil,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"msg":  "ok",
		"data": toMaterialVO(material),
	})
}

// Delete DELETE /api/v1/materials/:id
func (h *MaterialHandler) Delete(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"code": 40001,
			"msg":  "无效的素材 ID",
			"data": nil,
		})
		return
	}

	deleted, err := h.materialRepo.SoftDelete(c.Request.Context(), id)
	if err != nil {
		h.logger.Error("删除素材失败", zap.Error(err))
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
			"msg":  "素材不存在或已删除",
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
