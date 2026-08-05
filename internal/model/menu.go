package model

import (
	"encoding/json"
	"time"
)

// WechatMenu 微信自定义菜单
// 对应表: wechat_menus
type WechatMenu struct {
	ID          int64            `db:"id"`
	AccountID   int64            `db:"account_id"`
	MenuJSON    *json.RawMessage `db:"menu_json"`
	Status      int16            `db:"status"` // 0=草稿 1=已发布
	PublishedAt *time.Time       `db:"published_at"`
	DeletedAt   *time.Time       `db:"deleted_at"`
	CreatedAt   time.Time        `db:"created_at"`
	UpdatedAt   time.Time        `db:"updated_at"`
}

// MenuStatus constants
const (
	MenuStatusDraft     int16 = 0 // 草稿
	MenuStatusPublished int16 = 1 // 已发布
)
