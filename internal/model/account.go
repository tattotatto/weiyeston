package model

import (
	"encoding/json"
	"time"
)

// WechatAccount 微信公众号
// 对应表: wechat_accounts
type WechatAccount struct {
	ID             int64      `db:"id"`
	TenantID       int64      `db:"tenant_id"`
	Name           *string    `db:"name"`
	WxOriginalID   *string    `db:"wx_original_id"`
	WxAppID        *string    `db:"wx_app_id"`
	WxAppSecret    *string    `db:"wx_app_secret"`
	AuthType       int16      `db:"auth_type"`
	AuthStatus     int16      `db:"auth_status"`
	RefreshToken   *string    `db:"refresh_token"`
	AccessToken    *string    `db:"access_token"`
	TokenExpireAt  *time.Time `db:"token_expire_at"`
	AvatarURL      *string    `db:"avatar_url"`
	QRCodeURL      *string    `db:"qr_code_url"`
	Description    *string    `db:"description"`
	FansCount      int        `db:"fans_count"`

	// T3 新增字段
	AuthorizerAppid  *string           `db:"authorizer_appid"`
	FuncInfo         *json.RawMessage  `db:"func_info"`          // 设计修正: *json.RawMessage
	ServiceTypeInfo  *int16            `db:"service_type_info"`
	VerifyTypeInfo   *int16            `db:"verify_type_info"`
	NickName         *string           `db:"nick_name"`
	HeadImg          *string           `db:"head_img"`
	UserName         *string           `db:"user_name"`
	Alias            *string           `db:"alias"`
	PrincipalName    *string           `db:"principal_name"`
	QrcodeUrl        *string           `db:"qrcode_url"`         // 微信同步的二维码（复用已有 qr_code_url）
	Signature        *string           `db:"signature"`

	DeletedAt      *time.Time `db:"deleted_at"`
	CreatedAt      time.Time  `db:"created_at"`
	UpdatedAt      time.Time  `db:"updated_at"`
}
