package model

import "time"

// AutoReplyRule 自动回复规则
// 对应表: auto_reply_rules
type AutoReplyRule struct {
	ID            int64      `db:"id"`
	AccountID     int64      `db:"account_id"`
	Keyword       *string    `db:"keyword"`
	MatchType     int16      `db:"match_type"`
	ReplyType     int16      `db:"reply_type"`
	ReplyContent  string     `db:"reply_content"`
	ReplyTitle    *string    `db:"reply_title"`
	ReplyDesc     *string    `db:"reply_desc"`
	ReplyCoverURL *string    `db:"reply_cover_url"`
	ReplyURL      *string    `db:"reply_url"`
	Status        int16      `db:"status"`
	SortOrder     int        `db:"sort_order"`
	DeletedAt     *time.Time `db:"deleted_at"`
	CreatedAt     time.Time  `db:"created_at"`
	UpdatedAt     time.Time  `db:"updated_at"`
}
