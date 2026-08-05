// Package material 素材 Repository
package material

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/jmoiron/sqlx"
	"github.com/weiyeston/weiyeston-v2/internal/model"
)

// Repo 素材数据访问层
type Repo struct {
	DB *sqlx.DB
}

// ListByAccount 分页查询某公众号下的素材（按类型可选筛选，排除软删除记录）
func (r *Repo) ListByAccount(ctx context.Context, accountID int64, materialType string, offset, limit int) ([]model.Material, error) {
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

	err := r.DB.SelectContext(ctx, &materials, query, args...)
	if err != nil {
		return nil, err
	}
	if materials == nil {
		materials = make([]model.Material, 0)
	}
	return materials, nil
}

// GetByID 按 ID 查询素材（排除软删除记录）
func (r *Repo) GetByID(ctx context.Context, id int64) (*model.Material, error) {
	if id <= 0 {
		return nil, fmt.Errorf("无效的素材 ID: %d", id)
	}
	var m model.Material
	query := `SELECT * FROM materials WHERE id = $1 AND deleted_at IS NULL`
	err := r.DB.GetContext(ctx, &m, query, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &m, nil
}

// Create 创建素材记录
func (r *Repo) Create(ctx context.Context, m *model.Material) error {
	query := `INSERT INTO materials (account_id, media_id, type, name, url, thumbnail_url,
		file_size, width, height, format)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		RETURNING id, created_at, updated_at`
	row := r.DB.QueryRowContext(ctx, query,
		m.AccountID, m.MediaID, m.Type, m.Name, m.URL,
		m.ThumbnailURL, m.FileSize, m.Width, m.Height, m.Format,
	)
	return row.Scan(&m.ID, &m.CreatedAt, &m.UpdatedAt)
}

// SoftDelete 软删除素材
func (r *Repo) SoftDelete(ctx context.Context, id int64) (bool, error) {
	query := `UPDATE materials SET deleted_at = NOW() WHERE id = $1 AND deleted_at IS NULL`
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

// CountByAccount 统计某公众号下某类型素材的总数（排除软删除记录）
func (r *Repo) CountByAccount(ctx context.Context, accountID int64, materialType string) (int, error) {
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

	err := r.DB.GetContext(ctx, &count, query, args...)
	if err != nil {
		return 0, err
	}
	return count, nil
}
