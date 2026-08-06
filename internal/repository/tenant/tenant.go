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
	query := `INSERT INTO tenants (username, password_hash, nickname, email, phone, avatar_url, status, role, vip_level, company)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10) RETURNING id, created_at, updated_at`
	row := r.DB.QueryRowContext(ctx, query,
		t.Username, t.PasswordHash, t.Nickname, t.Email, t.Phone, t.AvatarURL, t.Status, t.Role, t.VipLevel, t.Company,
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

// Update 更新租户信息
func (r *Repo) Update(ctx context.Context, t *model.Tenant) error {
	query := `UPDATE tenants SET nickname = $1, email = $2, phone = $3, avatar_url = $4, status = $5,
		vip_level = $6, vip_end_time = $7, role = $8, updated_at = NOW()
		WHERE id = $9 AND deleted_at IS NULL`
	result, err := r.DB.ExecContext(ctx, query,
		t.Nickname, t.Email, t.Phone, t.AvatarURL, t.Status,
		t.VipLevel, t.VipEndTime, t.Role, t.ID,
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

// GetByPhone 按手机号查询租户（排除软删除记录）
func (r *Repo) GetByPhone(ctx context.Context, phone string) (*model.Tenant, error) {
	if phone == "" {
		return nil, fmt.Errorf("手机号不能为空")
	}
	var t model.Tenant
	query := `SELECT * FROM tenants WHERE phone = $1 AND deleted_at IS NULL`
	err := r.DB.GetContext(ctx, &t, query, phone)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &t, nil
}

// UpdatePassword 更新租户密码哈希
func (r *Repo) UpdatePassword(ctx context.Context, id int64, passwordHash string) error {
	query := `UPDATE tenants SET password_hash = $1, updated_at = NOW() WHERE id = $2 AND deleted_at IS NULL`
	result, err := r.DB.ExecContext(ctx, query, passwordHash, id)
	if err != nil {
		return err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return fmt.Errorf("租户不存在或已删除: id=%d", id)
	}
	return nil
}

// UpdateProfile 更新用户个人资料（仅限安全字段：nickname, email, phone, company, avatar_url）
func (r *Repo) UpdateProfile(ctx context.Context, id int64, nickname, email, phone, company, avatarURL *string) error {
	query := `UPDATE tenants SET nickname = $1, email = $2, phone = $3, company = $4, avatar_url = $5, updated_at = NOW()
		WHERE id = $6 AND deleted_at IS NULL`
	result, err := r.DB.ExecContext(ctx, query, nickname, email, phone, company, avatarURL, id)
	if err != nil {
		return err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return fmt.Errorf("租户不存在或已删除: id=%d", id)
	}
	return nil
}

// List 分页查询租户列表（支持关键词搜索和状态筛选，按创建时间倒序，排除软删除记录）
func (r *Repo) List(ctx context.Context, keyword string, status *int16, limit, offset int) ([]model.Tenant, int64, error) {
	if limit <= 0 {
		limit = 20
	}
	if offset < 0 {
		offset = 0
	}

	var total int64
	var users []model.Tenant

	// Build query
	where := "WHERE deleted_at IS NULL"
	args := []interface{}{}
	argIdx := 1

	if keyword != "" {
		where += fmt.Sprintf(" AND (username ILIKE $%d OR nickname ILIKE $%d)", argIdx, argIdx)
		args = append(args, "%"+keyword+"%")
		argIdx++
	}
	if status != nil {
		where += fmt.Sprintf(" AND status = $%d", argIdx)
		args = append(args, *status)
		argIdx++
	}

	// Count total
	countQuery := "SELECT COUNT(*) FROM tenants " + where
	if err := r.DB.GetContext(ctx, &total, countQuery, args...); err != nil {
		return nil, 0, err
	}

	// Fetch list
	query := fmt.Sprintf("SELECT * FROM tenants %s ORDER BY created_at DESC LIMIT $%d OFFSET $%d", where, argIdx, argIdx+1)
	args = append(args, limit, offset)

	if err := r.DB.SelectContext(ctx, &users, query, args...); err != nil {
		return nil, 0, err
	}

	if users == nil {
		users = make([]model.Tenant, 0)
	}

	return users, total, nil
}
