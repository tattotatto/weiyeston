package model

import "time"

// QuickNewsChannel 快讯栏目
// 对应表: quicknews_channels
type QuickNewsChannel struct {
	ID          int64      `db:"id"`
	AccountID   int64      `db:"account_id"`
	Name        string     `db:"name"`
	CoverURL    *string    `db:"cover_url"`
	Description *string    `db:"description"`
	SortOrder   int        `db:"sort_order"`
	Status      int16      `db:"status"`
	DeletedAt   *time.Time `db:"deleted_at"`
	CreatedAt   time.Time  `db:"created_at"`
	UpdatedAt   time.Time  `db:"updated_at"`
}

// QuickNewsUser 快讯用户（微信 OAuth 注册）
// 对应表: quicknews_users
type QuickNewsUser struct {
	ID        int64     `db:"id"`
	AccountID int64     `db:"account_id"`
	Openid    string    `db:"openid"`
	Unionid   *string   `db:"unionid"`
	Nickname  *string   `db:"nickname"`
	AvatarURL *string   `db:"avatar_url"`
	Status    int16     `db:"status"`
	CreatedAt time.Time `db:"created_at"`
	UpdatedAt time.Time `db:"updated_at"`
}

// QuickNews 快讯内容
// 对应表: quicknews_news
type QuickNews struct {
	ID           int64      `db:"id"`
	AccountID    int64      `db:"account_id"`
	ChannelID    int64      `db:"channel_id"`
	UserID       *int64     `db:"user_id"`
	AuthorName   *string    `db:"author_name"`
	AuthorAvatar *string    `db:"author_avatar"`
	Content      string     `db:"content"`
	LikeCount    int        `db:"like_count"`
	CommentCount int        `db:"comment_count"`
	Status       int16      `db:"status"`
	IsTop        bool       `db:"is_top"`
	DeletedAt    *time.Time `db:"deleted_at"`
	CreatedAt    time.Time  `db:"created_at"`
	UpdatedAt    time.Time  `db:"updated_at"`
}

// QuickNewsPhoto 快讯图片
// 对应表: quicknews_photos
type QuickNewsPhoto struct {
	ID        int64     `db:"id"`
	NewsID    int64     `db:"news_id"`
	URL       string    `db:"url"`
	SortOrder int       `db:"sort_order"`
	CreatedAt time.Time `db:"created_at"`
}

// QuickNewsLike 快讯点赞记录
// 对应表: quicknews_likes
type QuickNewsLike struct {
	ID        int64     `db:"id"`
	NewsID    int64     `db:"news_id"`
	UserID    *int64    `db:"user_id"`
	Openid    string    `db:"openid"`
	CreatedAt time.Time `db:"created_at"`
}

// QuickNewsComment 快讯评论
// 对应表: quicknews_comments
type QuickNewsComment struct {
	ID        int64      `db:"id"`
	NewsID    int64      `db:"news_id"`
	UserID    int64      `db:"user_id"`
	ParentID  *int64     `db:"parent_id"`
	Content   string     `db:"content"`
	LikeCount int        `db:"like_count"`
	Status    int16      `db:"status"`
	DeletedAt *time.Time `db:"deleted_at"`
	CreatedAt time.Time  `db:"created_at"`
	UpdatedAt time.Time  `db:"updated_at"`
}
