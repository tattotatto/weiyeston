package model

import "time"

// Material 素材
// 对应表: materials
type Material struct {
	ID           int64      `db:"id"`
	AccountID    int64      `db:"account_id"`
	MediaID      *string    `db:"media_id"`
	Type         string     `db:"type"`
	Name         *string    `db:"name"`
	URL          string     `db:"url"`
	ThumbnailURL *string    `db:"thumbnail_url"`
	FileSize     *int64     `db:"file_size"`
	Width        *int       `db:"width"`
	Height       *int       `db:"height"`
	Format       *string    `db:"format"`
	DeletedAt    *time.Time `db:"deleted_at"`
	CreatedAt    time.Time  `db:"created_at"`
	UpdatedAt    time.Time  `db:"updated_at"`
}
