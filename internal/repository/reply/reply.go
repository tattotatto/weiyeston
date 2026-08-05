// Package reply 自动回复 Repository
package reply

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/jmoiron/sqlx"
	"github.com/weiyeston/weiyeston-v2/internal/model"
)

// Repo 自动回复数据访问层
type Repo struct {
	DB *sqlx.DB
}

// ListByAccountID 查询某公众号下的所有回复规则（排除软删除记录）
func (r *Repo) ListByAccountID(ctx context.Context, accountID int64) ([]model.AutoReplyRule, error) {
	if accountID <= 0 {
		return nil, fmt.Errorf("无效的公众号 ID: %d", accountID)
	}
	var rules []model.AutoReplyRule
	query := `SELECT * FROM auto_reply_rules WHERE account_id = $1 AND deleted_at IS NULL ORDER BY sort_order ASC, id ASC`
	err := r.DB.SelectContext(ctx, &rules, query, accountID)
	if err != nil {
		return nil, err
	}
	if rules == nil {
		rules = make([]model.AutoReplyRule, 0)
	}
	return rules, nil
}

// GetByID 按 ID 查询规则（排除软删除记录）
func (r *Repo) GetByID(ctx context.Context, id int64) (*model.AutoReplyRule, error) {
	if id <= 0 {
		return nil, fmt.Errorf("无效的规则 ID: %d", id)
	}
	var rule model.AutoReplyRule
	query := `SELECT * FROM auto_reply_rules WHERE id = $1 AND deleted_at IS NULL`
	err := r.DB.GetContext(ctx, &rule, query, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &rule, nil
}

// Create 创建新的回复规则
func (r *Repo) Create(ctx context.Context, rule *model.AutoReplyRule) error {
	query := `INSERT INTO auto_reply_rules (account_id, keyword, match_type, reply_type,
		reply_content, reply_title, reply_desc, reply_cover_url, reply_url,
		status, sort_order)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		RETURNING id, created_at, updated_at`
	row := r.DB.QueryRowContext(ctx, query,
		rule.AccountID, rule.Keyword, rule.MatchType, rule.ReplyType,
		rule.ReplyContent, rule.ReplyTitle, rule.ReplyDesc, rule.ReplyCoverURL, rule.ReplyURL,
		rule.Status, rule.SortOrder,
	)
	return row.Scan(&rule.ID, &rule.CreatedAt, &rule.UpdatedAt)
}

// Update 更新回复规则
func (r *Repo) Update(ctx context.Context, rule *model.AutoReplyRule) error {
	query := `UPDATE auto_reply_rules SET
		keyword = $1, match_type = $2, reply_type = $3,
		reply_content = $4, reply_title = $5, reply_desc = $6,
		reply_cover_url = $7, reply_url = $8,
		status = $9, sort_order = $10, updated_at = NOW()
		WHERE id = $11 AND deleted_at IS NULL`
	result, err := r.DB.ExecContext(ctx, query,
		rule.Keyword, rule.MatchType, rule.ReplyType,
		rule.ReplyContent, rule.ReplyTitle, rule.ReplyDesc,
		rule.ReplyCoverURL, rule.ReplyURL,
		rule.Status, rule.SortOrder, rule.ID,
	)
	if err != nil {
		return err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return fmt.Errorf("规则不存在或已删除: id=%d", rule.ID)
	}
	return nil
}

// SoftDelete 软删除规则
func (r *Repo) SoftDelete(ctx context.Context, id int64) (bool, error) {
	query := `UPDATE auto_reply_rules SET deleted_at = NOW() WHERE id = $1 AND deleted_at IS NULL`
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
