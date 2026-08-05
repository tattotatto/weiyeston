// Package cms 微官网 CMS Repository
package cms

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/jmoiron/sqlx"
	"github.com/weiyeston/weiyeston-v2/internal/model"
)

// Repo 微官网 CMS 数据访问层
type Repo struct {
	DB *sqlx.DB
}

// ======================== 栏目操作 ========================

// CreateChannel 创建栏目
func (r *Repo) CreateChannel(ctx context.Context, ch *model.Channel) error {
	query := `INSERT INTO cms_channels (account_id, parent_id, name, slug, level, sort_order, cover_url, description, status)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING id, created_at, updated_at`
	row := r.DB.QueryRowContext(ctx, query,
		ch.AccountID, ch.ParentID, ch.Name, ch.Slug, ch.Level, ch.SortOrder, ch.CoverURL, ch.Description, ch.Status,
	)
	return row.Scan(&ch.ID, &ch.CreatedAt, &ch.UpdatedAt)
}

// GetChannelTree 查询栏目树，使用递归 CTE 按层级获取所有栏目
func (r *Repo) GetChannelTree(ctx context.Context, accountID int64) ([]model.Channel, error) {
	if accountID <= 0 {
		return nil, fmt.Errorf("无效的公众号 ID: %d", accountID)
	}
	var channels []model.Channel
	query := `WITH RECURSIVE channel_tree AS (SELECT * FROM cms_channels WHERE account_id = $1 AND parent_id IS NULL AND deleted_at IS NULL UNION ALL SELECT c.* FROM cms_channels c INNER JOIN channel_tree ct ON c.parent_id = ct.id WHERE c.deleted_at IS NULL) SELECT * FROM channel_tree ORDER BY level, sort_order`
	err := r.DB.SelectContext(ctx, &channels, query, accountID)
	if err != nil {
		return nil, err
	}
	return channels, nil
}

// UpdateChannel 更新栏目信息
func (r *Repo) UpdateChannel(ctx context.Context, ch *model.Channel) error {
	query := `UPDATE cms_channels SET parent_id = $1, name = $2, slug = $3, level = $4,
		sort_order = $5, cover_url = $6, description = $7, status = $8, updated_at = NOW()
		WHERE id = $9 AND deleted_at IS NULL`
	result, err := r.DB.ExecContext(ctx, query,
		ch.ParentID, ch.Name, ch.Slug, ch.Level, ch.SortOrder, ch.CoverURL, ch.Description, ch.Status, ch.ID,
	)
	if err != nil {
		return err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return fmt.Errorf("栏目不存在或已删除: id=%d", ch.ID)
	}
	return nil
}

// SoftDeleteChannel 软删除栏目（设置 deleted_at）
func (r *Repo) SoftDeleteChannel(ctx context.Context, id int64) error {
	query := `UPDATE cms_channels SET deleted_at = NOW() WHERE id = $1 AND deleted_at IS NULL`
	_, err := r.DB.ExecContext(ctx, query, id)
	return err
}

// ======================== 文章操作 ========================

// CreateArticle 创建文章
func (r *Repo) CreateArticle(ctx context.Context, a *model.Article) error {
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

// GetArticle 按 ID 查询文章（排除软删除记录）
func (r *Repo) GetArticle(ctx context.Context, id int64) (*model.Article, error) {
	if id <= 0 {
		return nil, fmt.Errorf("无效的文章 ID: %d", id)
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

// ListArticles 分页查询文章列表，支持按公众号、栏目、状态过滤
func (r *Repo) ListArticles(ctx context.Context, accountID int64, channelID *int64, status *int16, offset, limit int) ([]model.Article, error) {
	if limit <= 0 {
		return nil, fmt.Errorf("无效的分页参数: limit=%d", limit)
	}
	if offset < 0 {
		offset = 0
	}

	var articles []model.Article
	var err error

	switch {
	case channelID != nil && status != nil:
		// H5 栏目页：按栏目 + 状态查询，按排序字段排序
		query := `SELECT * FROM cms_articles WHERE channel_id = $1 AND status = $2 AND deleted_at IS NULL ORDER BY sort_order LIMIT $3 OFFSET $4`
		err = r.DB.SelectContext(ctx, &articles, query, *channelID, *status, limit, offset)
	case status != nil:
		// 管理后台：按状态过滤，按创建时间倒序
		query := `SELECT * FROM cms_articles WHERE account_id = $1 AND status = $2 AND deleted_at IS NULL ORDER BY created_at DESC LIMIT $3 OFFSET $4`
		err = r.DB.SelectContext(ctx, &articles, query, accountID, *status, limit, offset)
	default:
		// 管理后台：不按状态过滤，按创建时间倒序
		query := `SELECT * FROM cms_articles WHERE account_id = $1 AND deleted_at IS NULL ORDER BY created_at DESC LIMIT $2 OFFSET $3`
		err = r.DB.SelectContext(ctx, &articles, query, accountID, limit, offset)
	}

	if err != nil {
		return nil, err
	}
	return articles, nil
}

// UpdateArticle 更新文章信息
func (r *Repo) UpdateArticle(ctx context.Context, a *model.Article) error {
	query := `UPDATE cms_articles SET channel_id = $1, title = $2, cover_url = $3, summary = $4,
		author = $5, content = $6, html_cache = $7, status = $8,
		is_template = $9, template_cat = $10, sort_order = $11, view_count = $12,
		published_at = $13, updated_at = NOW()
		WHERE id = $14 AND deleted_at IS NULL`
	result, err := r.DB.ExecContext(ctx, query,
		a.ChannelID, a.Title, a.CoverURL, a.Summary, a.Author,
		a.Content, a.HTMLCache, a.Status, a.IsTemplate, a.TemplateCat,
		a.SortOrder, a.ViewCount, a.PublishedAt, a.ID,
	)
	if err != nil {
		return err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return fmt.Errorf("文章不存在或已删除: id=%d", a.ID)
	}
	return nil
}

// SoftDeleteArticle 软删除文章（设置 deleted_at）
func (r *Repo) SoftDeleteArticle(ctx context.Context, id int64) error {
	query := `UPDATE cms_articles SET deleted_at = NOW() WHERE id = $1 AND deleted_at IS NULL`
	_, err := r.DB.ExecContext(ctx, query, id)
	return err
}
