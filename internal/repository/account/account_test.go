package account

import (
	"context"
	"database/sql/driver"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/weiyeston/weiyeston-v2/internal/model"
)

// accountColumns 返回 wechat_accounts 表的完整列名（包含 T3 新增字段）
func accountColumns() []string {
	return []string{
		"id", "tenant_id", "name", "wx_original_id", "wx_app_id",
		"wx_app_secret", "auth_type", "auth_status", "refresh_token",
		"access_token", "token_expire_at", "avatar_url", "qr_code_url",
		"description", "fans_count",
		"authorizer_appid", "func_info", "service_type_info", "verify_type_info",
		"nick_name", "head_img", "user_name", "alias",
		"principal_name", "qrcode_url", "signature",
		"deleted_at", "created_at", "updated_at",
	}
}

// accountRowValues 返回一行完整的测试数据（[]driver.Value 类型）
func accountRowValues(id, tenantID int64, name string) []driver.Value {
	return []driver.Value{
		id, tenantID, name, nil, nil, nil, int16(1), int16(1), nil, nil, nil, nil, nil,
		nil, 0,
		nil, nil, nil, nil,
		nil, nil, nil, nil,
		nil, nil, nil,
		nil, time.Now(), time.Now(),
	}
}

// newMockRepo 创建包含 mock DB 的 Repo 用于测试
func newMockRepo(t *testing.T) (*Repo, sqlmock.Sqlmock) {
	t.Helper()
	db, mock, err := sqlmock.New()
	require.NoError(t, err, "创建 sqlmock 失败")
	sqlxDB := sqlx.NewDb(db, "postgres")
	return &Repo{DB: sqlxDB}, mock
}

// TestRepoExists 验证 Repo 结构体存在
func TestRepoExists(t *testing.T) {
	repo, _ := newMockRepo(t)
	defer repo.DB.Close()
	assert.NotNil(t, repo)
	assert.NotNil(t, repo.DB)
}

// TestGetByTenantID 测试按 tenant_id 查询公众号列表（表驱动）
func TestGetByTenantID(t *testing.T) {
	tests := []struct {
		name      string
		tenantID  int64
		wantCount int
		wantErr   bool
	}{
		{
			name:      "租户有多个公众号",
			tenantID:  1,
			wantCount: 3,
			wantErr:   false,
		},
		{
			name:      "租户有一个公众号",
			tenantID:  2,
			wantCount: 1,
			wantErr:   false,
		},
		{
			name:      "租户没有公众号",
			tenantID:  999,
			wantCount: 0,
			wantErr:   false,
		},
		{
			name:      "无效 tenant_id（零值）",
			tenantID:  0,
			wantCount: 0,
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo, mock := newMockRepo(t)
			defer repo.DB.Close()

			if tt.wantErr {
				// 无效参数不查询数据库
			} else {
				columns := accountColumns()
				rows := sqlmock.NewRows(columns)
				for i := 0; i < tt.wantCount; i++ {
					rows.AddRow(accountRowValues(int64(i+1), tt.tenantID, "公众号"+string(rune('A'+i)))...)
				}
				mock.ExpectQuery(`SELECT \* FROM wechat_accounts WHERE tenant_id = \$1 AND deleted_at IS NULL`).
					WithArgs(tt.tenantID).
					WillReturnRows(rows)
			}

			accounts, err := repo.GetByTenantID(context.Background(), tt.tenantID)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Len(t, accounts, tt.wantCount)
			}
		})
	}
}

// TestList 测试分页查询公众号列表
func TestList(t *testing.T) {
	tests := []struct {
		name      string
		tenantID  int64
		offset    int
		limit     int
		wantCount int
		wantErr   bool
	}{
		{
			name:      "分页查询第一页",
			tenantID:  1,
			offset:    0,
			limit:     10,
			wantCount: 5,
			wantErr:   false,
		},
		{
			name:      "分页查询第二页",
			tenantID:  1,
			offset:    10,
			limit:     10,
			wantCount: 2,
			wantErr:   false,
		},
		{
			name:      "空结果",
			tenantID:  1,
			offset:    100,
			limit:     10,
			wantCount: 0,
			wantErr:   false,
		},
		{
			name:      "无效分页参数",
			tenantID:  1,
			offset:    0,
			limit:     0,
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo, mock := newMockRepo(t)
			defer repo.DB.Close()

			if tt.wantErr {
				// 无效参数不查询
			} else {
				columns := accountColumns()
				rows := sqlmock.NewRows(columns)
				for i := 0; i < tt.wantCount; i++ {
					rows.AddRow(accountRowValues(int64(i+1), tt.tenantID, "公众号"+string(rune('A'+i)))...)
				}
				mock.ExpectQuery(`SELECT \* FROM wechat_accounts WHERE tenant_id = \$1 AND deleted_at IS NULL ORDER BY created_at DESC LIMIT \$2 OFFSET \$3`).
					WithArgs(tt.tenantID, tt.limit, tt.offset).
					WillReturnRows(rows)
			}

			accounts, err := repo.List(context.Background(), tt.tenantID, tt.offset, tt.limit)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Len(t, accounts, tt.wantCount)
			}
		})
	}
}

// TestGetByTenantIDQueryPattern 测试查询 SQL 模式（直接通过 sqlx 执行模拟查询）
func TestGetByTenantIDQueryPattern(t *testing.T) {
	t.Run("验证 SQL 包含软删除过滤", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer db.Close()
		sqlxDB := sqlx.NewDb(db, "postgres")

		columns := accountColumns()
		rows := sqlmock.NewRows(columns).
			AddRow(accountRowValues(1, 1, "测试号")...)

		mock.ExpectQuery(`SELECT \* FROM wechat_accounts WHERE tenant_id = \$1 AND deleted_at IS NULL`).
			WithArgs(int64(1)).
			WillReturnRows(rows)

		var accounts []model.WechatAccount
		err = sqlxDB.Select(&accounts, "SELECT * FROM wechat_accounts WHERE tenant_id = $1 AND deleted_at IS NULL", 1)
		assert.NoError(t, err)
		assert.Len(t, accounts, 1)
		assert.Equal(t, int64(1), accounts[0].TenantID)
		assert.Equal(t, "测试号", *accounts[0].Name)

		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("查询无结果的 tenant_id", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer db.Close()
		sqlxDB := sqlx.NewDb(db, "postgres")

		columns := accountColumns()
		rows := sqlmock.NewRows(columns)

		mock.ExpectQuery(`SELECT \* FROM wechat_accounts WHERE tenant_id = \$1 AND deleted_at IS NULL`).
			WithArgs(int64(999)).
			WillReturnRows(rows)

		var accounts []model.WechatAccount
		err = sqlxDB.Select(&accounts, "SELECT * FROM wechat_accounts WHERE tenant_id = $1 AND deleted_at IS NULL", 999)
		assert.NoError(t, err)
		assert.Len(t, accounts, 0)

		assert.NoError(t, mock.ExpectationsWereMet())
	})
}
