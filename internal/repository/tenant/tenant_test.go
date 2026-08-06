package tenant

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

// newMockRepo 创建包含 mock DB 的 Repo 用于测试
func newMockRepo(t *testing.T) (*Repo, sqlmock.Sqlmock) {
	t.Helper()
	db, mock, err := sqlmock.New()
	require.NoError(t, err, "创建 sqlmock 失败")
	sqlxDB := sqlx.NewDb(db, "postgres")
	return &Repo{DB: sqlxDB}, mock
}

// anyTime 用于 sqlmock 匹配任意 time.Time 参数
type anyTime struct{}

func (a anyTime) Match(v driver.Value) bool {
	_, ok := v.(time.Time)
	return ok
}

// TestRepoExists 验证 Repo 结构体存在
func TestRepoExists(t *testing.T) {
	repo, _ := newMockRepo(t)
	defer repo.DB.Close()
	assert.NotNil(t, repo, "Repo 不应为 nil")
	assert.NotNil(t, repo.DB, "Repo.DB 不应为 nil")
}

// TestCreate 测试创建租户（表驱动）
func TestCreate(t *testing.T) {
	tests := []struct {
		name    string
		tenant  model.Tenant
		wantErr bool
	}{
		{
			name: "创建正常租户",
			tenant: model.Tenant{
				Username:     "testuser",
				PasswordHash: "$2a$10$hashedpassword",
				Nickname:     strPtr("测试用户"),
				Status:       0,
			},
			wantErr: false,
		},
		{
			name: "创建带邮箱的租户",
			tenant: model.Tenant{
				Username:     "admin",
				PasswordHash: "$2a$10$adminhash",
				Email:        strPtr("admin@test.com"),
				Status:       1,
			},
			wantErr: false,
		},
		{
			name: "用户名已存在",
			tenant: model.Tenant{
				Username:     "duplicate",
				PasswordHash: "$2a$10$duphash",
				Status:       0,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo, mock := newMockRepo(t)
			defer repo.DB.Close()

			if tt.wantErr {
				mock.ExpectQuery(`INSERT INTO tenants`).
					WillReturnError(assert.AnError)
			} else {
				mock.ExpectQuery(`INSERT INTO tenants`).
					WithArgs(
						tt.tenant.Username,
						tt.tenant.PasswordHash,
						sqlmock.AnyArg(), // nickname (nullable)
						sqlmock.AnyArg(), // email (nullable)
						sqlmock.AnyArg(), // phone (nullable)
						sqlmock.AnyArg(), // avatar_url (nullable)
						tt.tenant.Status,
					sqlmock.AnyArg(), // role
					sqlmock.AnyArg(), // vip_level
					).
					WillReturnRows(sqlmock.NewRows([]string{"id", "created_at", "updated_at"}).
						AddRow(1, time.Now(), time.Now()))
			}

			err := repo.Create(context.Background(), &tt.tenant)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestGetByID 测试按 ID 查询租户（表驱动）
func TestGetByID(t *testing.T) {
	tests := []struct {
		name      string
		id        int64
		mockSetup func(mock sqlmock.Sqlmock)
		wantNil   bool
		wantErr   bool
	}{
		{
			name: "查询存在的租户",
			id:   1,
			mockSetup: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{
					"id", "username", "password_hash", "nickname", "email", "phone",
					"avatar_url", "status", "last_login_at", "deleted_at", "created_at", "updated_at",
				}).AddRow(1, "testuser", "$2a$10$hash", "测试", nil, nil, nil, 1, nil, nil, time.Now(), time.Now())
				mock.ExpectQuery(`SELECT \* FROM tenants WHERE id = \$1 AND deleted_at IS NULL`).
					WithArgs(int64(1)).
					WillReturnRows(rows)
			},
			wantNil: false,
			wantErr: false,
		},
		{
			name: "查询不存在的租户",
			id:   999,
			mockSetup: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(`SELECT \* FROM tenants WHERE id = \$1 AND deleted_at IS NULL`).
					WithArgs(int64(999)).
					WillReturnRows(sqlmock.NewRows([]string{"id", "username", "password_hash", "nickname", "email", "phone", "avatar_url", "status", "last_login_at", "deleted_at", "created_at", "updated_at"}))
			},
			wantNil: true,
			wantErr: false,
		},
		{
			name: "无效 ID（零值）",
			id:   0,
			mockSetup: func(mock sqlmock.Sqlmock) {
				// 无效 ID 不应查询数据库
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo, mock := newMockRepo(t)
			defer repo.DB.Close()

			if tt.mockSetup != nil {
				tt.mockSetup(mock)
			}

			tenant, err := repo.GetByID(context.Background(), tt.id)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				if tt.wantNil {
					assert.Nil(t, tenant)
				} else {
					assert.NotNil(t, tenant)
					assert.Equal(t, tt.id, tenant.ID)
				}
			}
		})
	}
}

// TestUpdate 测试更新租户
func TestUpdate(t *testing.T) {
	tests := []struct {
		name    string
		tenant  model.Tenant
		wantErr bool
	}{
		{
			name: "更新昵称",
			tenant: model.Tenant{
				ID:       1,
				Nickname: strPtr("新昵称"),
			},
			wantErr: false,
		},
		{
			name: "更新不存在的租户",
			tenant: model.Tenant{
				ID:       999,
				Nickname: strPtr("不存在"),
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo, mock := newMockRepo(t)
			defer repo.DB.Close()

			if tt.wantErr {
				mock.ExpectExec(`UPDATE tenants SET`).
					WillReturnResult(sqlmock.NewResult(0, 0))
			} else {
				mock.ExpectExec(`UPDATE tenants SET`).
					WithArgs(tt.tenant.Nickname, sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), tt.tenant.Status, sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), tt.tenant.ID).
					WillReturnResult(sqlmock.NewResult(0, 1))
			}

			err := repo.Update(context.Background(), &tt.tenant)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestSoftDelete 测试软删除租户
func TestSoftDelete(t *testing.T) {
	tests := []struct {
		name    string
		id      int64
		wantErr bool
	}{
		{
			name:    "软删除存在的租户",
			id:      1,
			wantErr: false,
		},
		{
			name:    "软删除不存在的租户",
			id:      999,
			wantErr: false, // 软删除不存在的记录不报错，只是影响 0 行
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo, mock := newMockRepo(t)
			defer repo.DB.Close()

			mock.ExpectExec(`UPDATE tenants SET deleted_at =`).
				WithArgs(anyTime{}, tt.id).
				WillReturnResult(sqlmock.NewResult(0, 1))

			err := repo.SoftDelete(context.Background(), tt.id)
			assert.NoError(t, err)
		})
	}
}

// TestList 测试分页查询租户列表
func TestList(t *testing.T) {
	tests := []struct {
		name      string
		keyword   string
		status    *int16
		offset    int
		limit     int
		wantCount int
		wantTotal int64
		wantErr   bool
	}{
		{
			name:      "查询第一页",
			keyword:   "",
			status:    nil,
			offset:    0,
			limit:     10,
			wantCount: 3,
			wantTotal: 3,
			wantErr:   false,
		},
		{
			name:      "查询空页",
			keyword:   "",
			status:    nil,
			offset:    100,
			limit:     10,
			wantCount: 0,
			wantTotal: 0,
			wantErr:   false,
		},
		{
			name:      "无效 limit",
			keyword:   "",
			status:    nil,
			offset:    0,
			limit:     -1,
			wantCount: 0,
			wantTotal: 0,
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo, mock := newMockRepo(t)
			defer repo.DB.Close()

			limit := tt.limit
			if limit <= 0 {
				limit = 20
			}
			offset := tt.offset
			if offset < 0 {
				offset = 0
			}

			// Mock count query
			countRows := sqlmock.NewRows([]string{"count"})
			countRows.AddRow(tt.wantTotal)
			mock.ExpectQuery(`SELECT COUNT\(\*\) FROM tenants WHERE deleted_at IS NULL`).
				WillReturnRows(countRows)

			columns := []string{"id", "username", "password_hash", "nickname", "email",
				"phone", "avatar_url", "status", "role", "vip_level", "vip_end_time",
				"last_login_at", "deleted_at", "created_at", "updated_at"}
			rows := sqlmock.NewRows(columns)
			for i := 0; i < tt.wantCount; i++ {
				rows.AddRow(int64(i+1), "user"+string(rune('a'+i)), "hash", nil, nil,
					nil, nil, int16(1), "user", "trial", nil, nil, nil, time.Now(), time.Now())
			}
			mock.ExpectQuery(`SELECT \* FROM tenants WHERE deleted_at IS NULL ORDER BY created_at DESC LIMIT \$1 OFFSET \$2`).
				WithArgs(limit, offset).
				WillReturnRows(rows)

			tenants, total, err := repo.List(context.Background(), tt.keyword, tt.status, tt.limit, tt.offset)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.wantTotal, total)
				assert.Len(t, tenants, tt.wantCount)
			}
		})
	}
}

// strPtr 辅助函数：返回字符串指针
func strPtr(s string) *string {
	return &s
}
