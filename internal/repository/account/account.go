// Package account 公众号 Repository
package account

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/weiyeston/weiyeston-v2/internal/model"
)

// Repo 公众号数据访问层
type Repo struct {
	DB *sqlx.DB
}

// Create 创建公众号
func (r *Repo) Create(ctx context.Context, a *model.WechatAccount) error {
	query := `INSERT INTO wechat_accounts (tenant_id, name, wx_original_id, wx_app_id, wx_app_secret,
		auth_type, auth_status, refresh_token, access_token, token_expire_at,
		avatar_url, qr_code_url, description, fans_count,
		authorizer_appid, func_info, service_type_info, verify_type_info,
		nick_name, head_img, user_name, alias, principal_name, qrcode_url, signature)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14,
			$15, $16, $17, $18, $19, $20, $21, $22, $23, $24, $25)
		RETURNING id, created_at, updated_at`
	row := r.DB.QueryRowContext(ctx, query,
		a.TenantID, a.Name, a.WxOriginalID, a.WxAppID, a.WxAppSecret,
		a.AuthType, a.AuthStatus, a.RefreshToken, a.AccessToken, a.TokenExpireAt,
		a.AvatarURL, a.QRCodeURL, a.Description, a.FansCount,
		a.AuthorizerAppid, a.FuncInfo, a.ServiceTypeInfo, a.VerifyTypeInfo,
		a.NickName, a.HeadImg, a.UserName, a.Alias, a.PrincipalName, a.QrcodeUrl, a.Signature,
	)
	return row.Scan(&a.ID, &a.CreatedAt, &a.UpdatedAt)
}

// GetByID 按 ID 查询公众号（排除软删除记录）
func (r *Repo) GetByID(ctx context.Context, id int64) (*model.WechatAccount, error) {
	if id <= 0 {
		return nil, fmt.Errorf("无效的公众号 ID: %d", id)
	}
	var a model.WechatAccount
	query := `SELECT * FROM wechat_accounts WHERE id = $1 AND deleted_at IS NULL`
	err := r.DB.GetContext(ctx, &a, query, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &a, nil
}

// GetByTenantID 按租户 ID 查询公众号列表（排除软删除记录）
func (r *Repo) GetByTenantID(ctx context.Context, tenantID int64) ([]model.WechatAccount, error) {
	if tenantID <= 0 {
		return nil, fmt.Errorf("无效的租户 ID: %d", tenantID)
	}
	var accounts []model.WechatAccount
	query := `SELECT * FROM wechat_accounts WHERE tenant_id = $1 AND deleted_at IS NULL ORDER BY created_at DESC`
	err := r.DB.SelectContext(ctx, &accounts, query, tenantID)
	if err != nil {
		return nil, err
	}
	return accounts, nil
}

// GetByAppID 按微信 AppID 查询公众号（排除软删除记录）
func (r *Repo) GetByAppID(ctx context.Context, appID string) (*model.WechatAccount, error) {
	if appID == "" {
		return nil, fmt.Errorf("AppID 不能为空")
	}
	var a model.WechatAccount
	query := `SELECT * FROM wechat_accounts WHERE wx_app_id = $1 AND deleted_at IS NULL`
	err := r.DB.GetContext(ctx, &a, query, appID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &a, nil
}

// Update 更新公众号信息
func (r *Repo) Update(ctx context.Context, a *model.WechatAccount) error {
	query := `UPDATE wechat_accounts SET name = $1, wx_original_id = $2, wx_app_id = $3,
		wx_app_secret = $4, auth_type = $5, auth_status = $6,
		refresh_token = $7, access_token = $8, token_expire_at = $9,
		avatar_url = $10, qr_code_url = $11, description = $12, fans_count = $13,
		authorizer_appid = $14, func_info = $15, service_type_info = $16, verify_type_info = $17,
		nick_name = $18, head_img = $19, user_name = $20, alias = $21,
		principal_name = $22, qrcode_url = $23, signature = $24,
		updated_at = NOW()
		WHERE id = $25 AND deleted_at IS NULL`
	result, err := r.DB.ExecContext(ctx, query,
		a.Name, a.WxOriginalID, a.WxAppID, a.WxAppSecret,
		a.AuthType, a.AuthStatus, a.RefreshToken, a.AccessToken, a.TokenExpireAt,
		a.AvatarURL, a.QRCodeURL, a.Description, a.FansCount,
		a.AuthorizerAppid, a.FuncInfo, a.ServiceTypeInfo, a.VerifyTypeInfo,
		a.NickName, a.HeadImg, a.UserName, a.Alias,
		a.PrincipalName, a.QrcodeUrl, a.Signature, a.ID,
	)
	if err != nil {
		return err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return fmt.Errorf("公众号不存在或已删除: id=%d", a.ID)
	}
	return nil
}

// SoftDelete 软删除公众号（设置 deleted_at）
func (r *Repo) SoftDelete(ctx context.Context, id int64) error {
	query := `UPDATE wechat_accounts SET deleted_at = NOW() WHERE id = $1 AND deleted_at IS NULL`
	_, err := r.DB.ExecContext(ctx, query, id)
	return err
}

// List 分页查询公众号列表（按创建时间倒序，排除软删除记录）
func (r *Repo) List(ctx context.Context, tenantID int64, offset, limit int) ([]model.WechatAccount, error) {
	if limit <= 0 {
		return nil, fmt.Errorf("无效的分页参数: limit=%d", limit)
	}
	if offset < 0 {
		offset = 0
	}
	var accounts []model.WechatAccount
	query := `SELECT * FROM wechat_accounts WHERE tenant_id = $1 AND deleted_at IS NULL ORDER BY created_at DESC LIMIT $2 OFFSET $3`
	err := r.DB.SelectContext(ctx, &accounts, query, tenantID, limit, offset)
	if err != nil {
		return nil, err
	}
	return accounts, nil
}

// ListParams 分页列表查询参数
type ListParams struct {
	TenantID   int64
	Page       int
	PageSize   int
	Keyword    string
	AuthType   int
	AuthStatus int
}

// ListPaginated 分页+搜索+筛选列表（返回结果+总数）
func (r *Repo) ListPaginated(ctx context.Context, params ListParams) ([]model.WechatAccount, int64, error) {
	var total int64

	// 构建参数化查询
	type queryArg struct {
		sql  string
		args []interface{}
	}

	// 基础条件
	whereClause := "WHERE tenant_id = $1 AND deleted_at IS NULL"
	baseArgs := []interface{}{params.TenantID}
	argIdx := 2

	// 关键词搜索
	if params.Keyword != "" {
		whereClause += fmt.Sprintf(" AND (name ILIKE $%d OR wx_app_id ILIKE $%d)", argIdx, argIdx)
		baseArgs = append(baseArgs, "%"+params.Keyword+"%")
		argIdx++
	}

	// 接入方式筛选
	if params.AuthType > 0 {
		whereClause += fmt.Sprintf(" AND auth_type = $%d", argIdx)
		baseArgs = append(baseArgs, params.AuthType)
		argIdx++
	}

	// 状态筛选
	if params.AuthStatus >= 0 {
		whereClause += fmt.Sprintf(" AND auth_status = $%d", argIdx)
		baseArgs = append(baseArgs, params.AuthStatus)
		argIdx++
	}

	// 计数查询
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM wechat_accounts %s", whereClause)
	err := r.DB.GetContext(ctx, &total, countQuery, baseArgs...)
	if err != nil {
		return nil, 0, err
	}

	// 分页数据查询
	columns := `id, tenant_id, name, wx_original_id, wx_app_id,
		auth_type, auth_status, avatar_url, qr_code_url,
		description, fans_count, token_expire_at,
		nick_name, head_img, principal_name,
		created_at, updated_at`
	dataQuery := fmt.Sprintf("SELECT %s FROM wechat_accounts %s ORDER BY created_at DESC LIMIT $%d OFFSET $%d",
		columns, whereClause, argIdx, argIdx+1)
	offset := (params.Page - 1) * params.PageSize
	dataArgs := append(baseArgs, params.PageSize, offset)
	dataArgsCopied := make([]interface{}, len(dataArgs))
	copy(dataArgsCopied, dataArgs)

	var accounts []model.WechatAccount
	err = r.DB.SelectContext(ctx, &accounts, dataQuery, dataArgsCopied...)
	if err != nil {
		return nil, 0, err
	}
	if accounts == nil {
		accounts = make([]model.WechatAccount, 0)
	}

	return accounts, total, nil
}

// GetByAppIDAndTenant 按 AppId + tenant_id 查询（唯一性校验用）
func (r *Repo) GetByAppIDAndTenant(ctx context.Context, appID string, tenantID int64) (*model.WechatAccount, error) {
	if appID == "" {
		return nil, fmt.Errorf("AppID 不能为空")
	}
	var a model.WechatAccount
	query := `SELECT * FROM wechat_accounts WHERE wx_app_id = $1 AND tenant_id = $2 AND auth_type = 1 AND auth_status IN (0,1) AND deleted_at IS NULL`
	err := r.DB.GetContext(ctx, &a, query, appID, tenantID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &a, nil
}

// GetManualAccountsNeedingRefresh 查询需要刷新 token 的手动接入账号
func (r *Repo) GetManualAccountsNeedingRefresh(ctx context.Context, beforeTime time.Time) ([]model.WechatAccount, error) {
	var accounts []model.WechatAccount
	query := `SELECT * FROM wechat_accounts
		WHERE auth_type = 1 AND auth_status = 1 AND deleted_at IS NULL
		AND (token_expire_at IS NULL OR token_expire_at < $1)
		ORDER BY created_at ASC`
	err := r.DB.SelectContext(ctx, &accounts, query, beforeTime)
	if err != nil {
		return nil, err
	}
	if accounts == nil {
		accounts = make([]model.WechatAccount, 0)
	}
	return accounts, nil
}

// CountByTenant 统计租户下的公众号总数
func (r *Repo) CountByTenant(ctx context.Context, tenantID int64) (int64, error) {
	var count int64
	query := `SELECT COUNT(*) FROM wechat_accounts WHERE tenant_id = $1 AND deleted_at IS NULL`
	err := r.DB.GetContext(ctx, &count, query, tenantID)
	return count, err
}

// SoftDeleteWithReturn 软删除并返回是否影响行
func (r *Repo) SoftDeleteWithReturn(ctx context.Context, id int64) (bool, error) {
	query := `UPDATE wechat_accounts SET deleted_at = NOW() WHERE id = $1 AND deleted_at IS NULL`
	result, err := r.DB.ExecContext(ctx, query, id)
	if err != nil {
		return false, err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return false, err
	}
	return rows > 0, nil
}

// GetByAuthorizerAppid 通过授权方 AppId 查询（用于事件分发）
func (r *Repo) GetByAuthorizerAppid(ctx context.Context, authorizerAppid string) (*model.WechatAccount, error) {
	if authorizerAppid == "" {
		return nil, fmt.Errorf("authorizer_appid 不能为空")
	}
	var a model.WechatAccount
	query := `SELECT * FROM wechat_accounts WHERE authorizer_appid = $1 AND deleted_at IS NULL`
	err := r.DB.GetContext(ctx, &a, query, authorizerAppid)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &a, nil
}

// UpdateToken 更新 access_token + refresh_token + 过期时间
func (r *Repo) UpdateToken(ctx context.Context, id int64, accessToken, refreshToken string, expireAt time.Time) error {
	query := `UPDATE wechat_accounts SET
		access_token = $1, refresh_token = $2, token_expire_at = $3,
		auth_status = 1, updated_at = NOW()
		WHERE id = $4 AND deleted_at IS NULL`
	_, err := r.DB.ExecContext(ctx, query, accessToken, refreshToken, expireAt, id)
	return err
}

// UpdateAuthStatus 更新授权状态
func (r *Repo) UpdateAuthStatus(ctx context.Context, id int64, status int16) error {
	query := `UPDATE wechat_accounts SET auth_status = $1, updated_at = NOW()
		WHERE id = $2 AND deleted_at IS NULL`
	_, err := r.DB.ExecContext(ctx, query, status, id)
	return err
}
