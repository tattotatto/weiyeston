package model

import "time"

// SystemConfig 系统/租户配置
// 对应表: system_configs
type SystemConfig struct {
	ID          int64     `db:"id"`
	AccountID   *int64    `db:"account_id"`
	Key         string    `db:"key"`
	Value       *string   `db:"value"`
	Type        string    `db:"type"`
	Description *string   `db:"description"`
	CreatedAt   time.Time `db:"created_at"`
	UpdatedAt   time.Time `db:"updated_at"`
}
