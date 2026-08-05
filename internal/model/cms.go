package model

import (
	"encoding/json"
	"time"
)

// Channel 微官网栏目（树形结构）
// 对应表: cms_channels
type Channel struct {
	ID          int64      `db:"id"`
	AccountID   int64      `db:"account_id"`
	ParentID    *int64     `db:"parent_id"`
	Name        string     `db:"name"`
	Slug        *string    `db:"slug"`
	Level       int16      `db:"level"`
	SortOrder   int        `db:"sort_order"`
	CoverURL    *string    `db:"cover_url"`
	Description *string    `db:"description"`
	Status      int16      `db:"status"`
	DeletedAt   *time.Time `db:"deleted_at"`
	CreatedAt   time.Time  `db:"created_at"`
	UpdatedAt   time.Time  `db:"updated_at"`
}

// Article 微官网文章
// 对应表: cms_articles
type Article struct {
	ID          int64           `db:"id"`
	AccountID   int64           `db:"account_id"`
	ChannelID   *int64          `db:"channel_id"`
	Title       *string         `db:"title"`
	CoverURL    *string         `db:"cover_url"`
	Summary     *string         `db:"summary"`
	Author      *string         `db:"author"`
	Content     json.RawMessage `db:"content"`
	HTMLCache   *string         `db:"html_cache"`
	Status      int16           `db:"status"`
	IsTemplate  bool            `db:"is_template"`
	TemplateCat *string         `db:"template_cat"`
	SortOrder   int             `db:"sort_order"`
	ViewCount   int             `db:"view_count"`
	PublishedAt *time.Time      `db:"published_at"`
	DeletedAt   *time.Time      `db:"deleted_at"`
	CreatedAt   time.Time       `db:"created_at"`
	UpdatedAt   time.Time       `db:"updated_at"`
}
