// Package model 数据库模型结构体定义
// 对应 PostgreSQL 表结构，使用 db tag 映射列名
package model

import "time"

// Tenant 租户（平台用户）
// 对应表: tenants
type Tenant struct {
	ID           int64      `db:"id"`
	Username     string     `db:"username"`
	PasswordHash string     `db:"password_hash"`
	Nickname     *string    `db:"nickname"`
	Email        *string    `db:"email"`
	Phone        *string    `db:"phone"`
	AvatarURL    *string    `db:"avatar_url"`
	Role         string     `db:"role"`         // admin | user
	Status       int16      `db:"status"`
	VipEndTime   *time.Time `db:"vip_end_time"`
	VipLevel     string     `db:"vip_level"`
	LastLoginAt  *time.Time `db:"last_login_at"`
	DeletedAt    *time.Time `db:"deleted_at"`
	CreatedAt    time.Time  `db:"created_at"`
	UpdatedAt    time.Time  `db:"updated_at"`
}
