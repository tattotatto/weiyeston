package model

import "time"

// Vote 投票活动
// 对应表: votes
type Vote struct {
	ID          int64      `db:"id"`
	AccountID   int64      `db:"account_id"`
	Title       string     `db:"title"`
	Description *string    `db:"description"`
	CoverURL    *string    `db:"cover_url"`
	VoteType    int16      `db:"vote_type"`
	MaxChoices  int        `db:"max_choices"`
	MaxVotes    int        `db:"max_votes"`
	StartTime   *time.Time `db:"start_time"`
	EndTime     *time.Time `db:"end_time"`
	TotalVotes  int        `db:"total_votes"`
	Status      int16      `db:"status"`
	DeletedAt   *time.Time `db:"deleted_at"`
	CreatedAt   time.Time  `db:"created_at"`
	UpdatedAt   time.Time  `db:"updated_at"`
}

// VoteOption 投票选项
// 对应表: vote_options
type VoteOption struct {
	ID        int64   `db:"id"`
	VoteID    int64   `db:"vote_id"`
	Content   string  `db:"content"`
	ImageURL  *string `db:"image_url"`
	SortOrder int     `db:"sort_order"`
	VoteCount int     `db:"vote_count"`
}

// VoteRecord 投票记录
// 对应表: vote_records
type VoteRecord struct {
	ID        int64     `db:"id"`
	VoteID    int64     `db:"vote_id"`
	OptionID  int64     `db:"option_id"`
	Openid    string    `db:"openid"`
	IPAddress *string   `db:"ip_address"`
	UserAgent *string   `db:"user_agent"`
	CreatedAt time.Time `db:"created_at"`
}
