// Package tenant 租户 Repository
package tenant

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/weiyeston/weiyeston-v2/internal/model"
)

// Repo 租户数据访问层
type Repo struct {
	DB *sqlx.DB
}

// Create 创建租户，返回填充了 ID 的 tenant
func (r *Repo) Create(ctx context.Context, t *model.Tenant) error {
	query := `INSERT INTO tenants (username, password_hash, nickname, email, phone, avatar_url, status)
		VALUES ($1, $2, $3, $4, $5, $6, $7) RETURNING id, created_at, updated_at`
	row := r.DB.QueryRowContext(ctx, query,
		t.Username, t.PasswordHash, t.Nickname, t.Email, t.Phone, t.AvatarURL, t.Status,
	)
	return row.Scan(&t.ID, &t.CreatedAt, &t.UpdatedAt)
}

// GetByID 按 ID 查询租户（排除软删除记录）
func (r *Repo) GetByID(ctx context.Context, id int64) (*model.Tenant, error) {
	if id <= 0 {
		return nil, fmt.Errorf("无效的租户 ID: %d", id)
	}
	var t model.Tenant
	query := `SELECT * FROM tenants WHERE id = $1 AND deleted_at IS NULL`
	err := r.DB.GetContext(ctx, &t, query, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &t, nil
}

// GetByUsername 按用户名查询租户（排除软删除记录）
func (r *Repo) GetByUsername(ctx context.Context, username string) (*model.Tenant, error) {
	if username == "" {
		return nil, fmt.Errorf("用户名不能为空")
	}
	var t model.Tenant
	query := `SELECT * FROM tenants WHERE username = $1 AND deleted_at IS NULL`
	err := r.DB.GetContext(ctx, &t, query, username)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &t, nil
}

// Update 更新租户信息（仅更新非零值字段）
func (r *Repo) Update(ctx context.Context, t *model.Tenant) error {
	query := `UPDATE tenants SET nickname = $1, email = $2, phone = $3, avatar_url = $4, status = $5, updated_at = NOW()
		WHERE id = $6 AND deleted_at IS NULL`
	result, err := r.DB.ExecContext(ctx, query,
		t.Nickname, t.Email, t.Phone, t.AvatarURL, t.Status, t.ID,
	)
	if err != nil {
		return err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return fmt.Errorf("租户不存在或已删除: id=%d", t.ID)
	}
	return nil
}

// SoftDelete 软删除租户（设置 deleted_at）
func (r *Repo) SoftDelete(ctx context.Context, id int64) error {
	query := `UPDATE tenants SET deleted_at = $1 WHERE id = $2 AND deleted_at IS NULL`
	_, err := r.DB.ExecContext(ctx, query, time.Now(), id)
	return err
}

// List 分页查询租户列表（按创建时间倒序，排除软删除记录）
func (r *Repo) List(ctx context.Context, offset, limit int) ([]model.Tenant, error) {
	if limit <= 0 {
		return nil, fmt.Errorf("无效的分页参数: limit=%d", limit)
	}
	if offset < 0 {
		offset = 0
	}
	var tenants []model.Tenant
	query := `SELECT * FROM tenants WHERE deleted_at IS NULL ORDER BY created_at DESC LIMIT $1 OFFSET $2`
	err := r.DB.SelectContext(ctx, &tenants, query, limit, offset)
	if err != nil {
		return nil, err
	}
	return tenants, nil
}
