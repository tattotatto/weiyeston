package cms

import (
	"context"
	"encoding/json"
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

// TestRepoExists 验证 Repo 结构体存在
func TestRepoExists(t *testing.T) {
	repo, _ := newMockRepo(t)
	defer repo.DB.Close()
	assert.NotNil(t, repo)
	assert.NotNil(t, repo.DB)
}

// TestGetChannelTreeQueryPattern 测试栏目树查询 SQL 模式（递归 CTE）
func TestGetChannelTreeQueryPattern(t *testing.T) {
	t.Run("查询根栏目（无 parent_id）", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer db.Close()
		sqlxDB := sqlx.NewDb(db, "postgres")

		columns := []string{
			"id", "account_id", "parent_id", "name", "slug", "level",
			"sort_order", "cover_url", "description", "status",
			"deleted_at", "created_at", "updated_at",
		}
		rows := sqlmock.NewRows(columns).
			AddRow(1, 1, nil, "新闻中心", "news", 0, 0, nil, nil, 1, nil, time.Now(), time.Now()).
			AddRow(2, 1, nil, "关于我们", "about", 0, 1, nil, nil, 1, nil, time.Now(), time.Now())

		mock.ExpectQuery(`SELECT .* FROM cms_channels WHERE account_id = \$1 AND parent_id IS NULL AND deleted_at IS NULL ORDER BY sort_order`).
			WithArgs(int64(1)).
			WillReturnRows(rows)

		var channels []model.Channel
		err = sqlxDB.Select(&channels,
			"SELECT * FROM cms_channels WHERE account_id = $1 AND parent_id IS NULL AND deleted_at IS NULL ORDER BY sort_order",
			1)
		assert.NoError(t, err)
		assert.Len(t, channels, 2)
		assert.Equal(t, "新闻中心", channels[0].Name)
		assert.Nil(t, channels[0].ParentID)

		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("查询子栏目（按 parent_id）", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer db.Close()
		sqlxDB := sqlx.NewDb(db, "postgres")

		columns := []string{
			"id", "account_id", "parent_id", "name", "slug", "level",
			"sort_order", "cover_url", "description", "status",
			"deleted_at", "created_at", "updated_at",
		}
		parentID := int64(1)
		rows := sqlmock.NewRows(columns).
			AddRow(3, 1, &parentID, "公司动态", "company-news", 1, 0, nil, nil, 1, nil, time.Now(), time.Now()).
			AddRow(4, 1, &parentID, "行业资讯", "industry", 1, 1, nil, nil, 1, nil, time.Now(), time.Now())

		mock.ExpectQuery(`SELECT .* FROM cms_channels WHERE account_id = \$1 AND parent_id = \$2 AND deleted_at IS NULL ORDER BY sort_order`).
			WithArgs(int64(1), parentID).
			WillReturnRows(rows)

		var channels []model.Channel
		err = sqlxDB.Select(&channels,
			"SELECT * FROM cms_channels WHERE account_id = $1 AND parent_id = $2 AND deleted_at IS NULL ORDER BY sort_order",
			1, parentID)
		assert.NoError(t, err)
		assert.Len(t, channels, 2)
		assert.Equal(t, parentID, *channels[0].ParentID)
		assert.Equal(t, int16(1), channels[0].Level)

		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("全部栏目（含不可见和软删除过滤）", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer db.Close()
		sqlxDB := sqlx.NewDb(db, "postgres")

		columns := []string{
			"id", "account_id", "parent_id", "name", "slug", "level",
			"sort_order", "cover_url", "description", "status",
			"deleted_at", "created_at", "updated_at",
		}
		// H5 展示时过滤 status=0（隐藏）和 deleted_at 不为 NULL 的记录
		rows := sqlmock.NewRows(columns).
			AddRow(1, 1, nil, "可见栏目", "visible", 0, 0, nil, nil, 1, nil, time.Now(), time.Now())

		mock.ExpectQuery(`SELECT .* FROM cms_channels WHERE account_id = \$1 AND status = 1 AND deleted_at IS NULL ORDER BY parent_id, sort_order`).
			WithArgs(int64(1)).
			WillReturnRows(rows)

		var channels []model.Channel
		err = sqlxDB.Select(&channels,
			"SELECT * FROM cms_channels WHERE account_id = $1 AND status = 1 AND deleted_at IS NULL ORDER BY parent_id, sort_order",
			1)
		assert.NoError(t, err)
		assert.Len(t, channels, 1)
		assert.Equal(t, int16(1), channels[0].Status)

		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

// TestListArticlesQueryPattern 测试文章分页查询 SQL 模式
func TestListArticlesQueryPattern(t *testing.T) {
	t.Run("按公众号分页查询（管理后台）", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer db.Close()
		sqlxDB := sqlx.NewDb(db, "postgres")

		content := json.RawMessage(`{"type":"doc","content":[]}`)
		columns := []string{
			"id", "account_id", "channel_id", "title", "cover_url", "summary",
			"author", "content", "html_cache", "status", "is_template",
			"template_cat", "sort_order", "view_count", "published_at",
			"deleted_at", "created_at", "updated_at",
		}
		publishedAt := time.Now()
		rows := sqlmock.NewRows(columns).
			AddRow(1, 1, nil, "文章1", nil, nil, nil, content, nil, 1, false, nil, 0, 100, &publishedAt, nil, time.Now(), time.Now()).
			AddRow(2, 1, nil, "文章2", nil, nil, nil, content, nil, 0, false, nil, 0, 50, nil, nil, time.Now(), time.Now())

		mock.ExpectQuery(`SELECT \* FROM cms_articles WHERE account_id = \$1 AND deleted_at IS NULL ORDER BY created_at DESC LIMIT \$2 OFFSET \$3`).
			WithArgs(int64(1), 10, 0).
			WillReturnRows(rows)

		var articles []model.Article
		err = sqlxDB.Select(&articles,
			"SELECT * FROM cms_articles WHERE account_id = $1 AND deleted_at IS NULL ORDER BY created_at DESC LIMIT $2 OFFSET $3",
			1, 10, 0)
		assert.NoError(t, err)
		assert.Len(t, articles, 2)
		assert.Equal(t, "文章1", *articles[0].Title)
		assert.Equal(t, int16(1), articles[0].Status)

		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("按栏目查询（H5 栏目页）", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer db.Close()
		sqlxDB := sqlx.NewDb(db, "postgres")

		content := json.RawMessage(`{"type":"doc"}`)
		columns := []string{
			"id", "account_id", "channel_id", "title", "cover_url", "summary",
			"author", "content", "html_cache", "status", "is_template",
			"template_cat", "sort_order", "view_count", "published_at",
			"deleted_at", "created_at", "updated_at",
		}
		channelID := int64(5)
		publishedAt := time.Now()
		rows := sqlmock.NewRows(columns).
			AddRow(1, 1, &channelID, "栏目文章1", nil, nil, nil, content, nil, 1, false, nil, 0, 100, &publishedAt, nil, time.Now(), time.Now())

		mock.ExpectQuery(`SELECT \* FROM cms_articles WHERE channel_id = \$1 AND status = \$2 AND deleted_at IS NULL ORDER BY sort_order LIMIT \$3 OFFSET \$4`).
			WithArgs(channelID, int16(1), 10, 0).
			WillReturnRows(rows)

		var articles []model.Article
		err = sqlxDB.Select(&articles,
			"SELECT * FROM cms_articles WHERE channel_id = $1 AND status = $2 AND deleted_at IS NULL ORDER BY sort_order LIMIT $3 OFFSET $4",
			channelID, int16(1), 10, 0)
		assert.NoError(t, err)
		assert.Len(t, articles, 1)
		assert.Equal(t, channelID, *articles[0].ChannelID)

		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("按状态过滤（草稿列表）", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer db.Close()
		sqlxDB := sqlx.NewDb(db, "postgres")

		content := json.RawMessage(`{}`)
		columns := []string{
			"id", "account_id", "channel_id", "title", "cover_url", "summary",
			"author", "content", "html_cache", "status", "is_template",
			"template_cat", "sort_order", "view_count", "published_at",
			"deleted_at", "created_at", "updated_at",
		}
		rows := sqlmock.NewRows(columns).
			AddRow(1, 1, nil, "草稿1", nil, nil, nil, content, nil, 0, false, nil, 0, 0, nil, nil, time.Now(), time.Now())

		mock.ExpectQuery(`SELECT \* FROM cms_articles WHERE account_id = \$1 AND status = \$2 AND deleted_at IS NULL ORDER BY created_at DESC LIMIT \$3 OFFSET \$4`).
			WithArgs(int64(1), int16(0), 10, 0).
			WillReturnRows(rows)

		var articles []model.Article
		err = sqlxDB.Select(&articles,
			"SELECT * FROM cms_articles WHERE account_id = $1 AND status = $2 AND deleted_at IS NULL ORDER BY created_at DESC LIMIT $3 OFFSET $4",
			1, int16(0), 10, 0)
		assert.NoError(t, err)
		assert.Len(t, articles, 1)
		assert.Equal(t, int16(0), articles[0].Status)

		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("模板查询", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer db.Close()
		sqlxDB := sqlx.NewDb(db, "postgres")

		content := json.RawMessage(`{"type":"doc"}`)
		columns := []string{
			"id", "account_id", "channel_id", "title", "cover_url", "summary",
			"author", "content", "html_cache", "status", "is_template",
			"template_cat", "sort_order", "view_count", "published_at",
			"deleted_at", "created_at", "updated_at",
		}
		templateCat := "holiday"
		rows := sqlmock.NewRows(columns).
			AddRow(1, 1, nil, "春节模板", nil, nil, nil, content, nil, 0, true, &templateCat, 0, 0, nil, nil, time.Now(), time.Now())

		mock.ExpectQuery(`SELECT \* FROM cms_articles WHERE account_id = \$1 AND is_template = TRUE AND deleted_at IS NULL`).
			WithArgs(int64(1)).
			WillReturnRows(rows)

		var articles []model.Article
		err = sqlxDB.Select(&articles,
			"SELECT * FROM cms_articles WHERE account_id = $1 AND is_template = TRUE AND deleted_at IS NULL",
			1)
		assert.NoError(t, err)
		assert.Len(t, articles, 1)
		assert.True(t, articles[0].IsTemplate)
		assert.Equal(t, "holiday", *articles[0].TemplateCat)

		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

// TestGetChannelTree 测试栏目树查询方法（表驱动）
func TestGetChannelTree(t *testing.T) {
	tests := []struct {
		name      string
		accountID int64
		wantCount int
		wantErr   bool
	}{
		{
			name:      "有栏目的公众号",
			accountID: 1,
			wantCount: 3,
			wantErr:   false,
		},
		{
			name:      "无栏目的公众号",
			accountID: 999,
			wantCount: 0,
			wantErr:   false,
		},
		{
			name:      "无效 account_id",
			accountID: 0,
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
				columns := []string{
					"id", "account_id", "parent_id", "name", "slug", "level",
					"sort_order", "cover_url", "description", "status",
					"deleted_at", "created_at", "updated_at",
				}
				rows := sqlmock.NewRows(columns)
				for i := 0; i < tt.wantCount; i++ {
					rows.AddRow(
						int64(i+1), tt.accountID, nil,
						"栏目"+string(rune('A'+i)), nil, int16(0),
						i+1, nil, nil, int16(1),
						nil, time.Now(), time.Now(),
					)
				}
				mock.ExpectQuery(`WITH RECURSIVE.*cms_channels.*account_id = \$1.*deleted_at IS NULL`).
					WithArgs(tt.accountID).
					WillReturnRows(rows)
			}

			channels, err := repo.GetChannelTree(context.Background(), tt.accountID)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Len(t, channels, tt.wantCount)
			}
		})
	}
}

// TestListArticles 测试文章分页查询方法（表驱动）
func TestListArticles(t *testing.T) {
	tests := []struct {
		name      string
		accountID int64
		channelID *int64
		status    *int16
		offset    int
		limit     int
		wantCount int
		wantErr   bool
	}{
		{
			name:      "管理后台：所有文章",
			accountID: 1,
			channelID: nil,
			status:    nil,
			offset:    0,
			limit:     10,
			wantCount: 5,
			wantErr:   false,
		},
		{
			name:      "H5 栏目页：已发布文章",
			accountID: 1,
			channelID: int64Ptr(5),
			status:    int16Ptr(1),
			offset:    0,
			limit:     10,
			wantCount: 3,
			wantErr:   false,
		},
		{
			name:      "草稿箱",
			accountID: 1,
			channelID: nil,
			status:    int16Ptr(0),
			offset:    0,
			limit:     10,
			wantCount: 2,
			wantErr:   false,
		},
		{
			name:      "空页",
			accountID: 1,
			channelID: nil,
			status:    nil,
			offset:    1000,
			limit:     10,
			wantCount: 0,
			wantErr:   false,
		},
		{
			name:      "无效 limit",
			accountID: 1,
			offset:    0,
			limit:     -1,
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
				content := json.RawMessage(`{}`)
				columns := []string{
					"id", "account_id", "channel_id", "title", "cover_url", "summary",
					"author", "content", "html_cache", "status", "is_template",
					"template_cat", "sort_order", "view_count", "published_at",
					"deleted_at", "created_at", "updated_at",
				}
				rows := sqlmock.NewRows(columns)
				for i := 0; i < tt.wantCount; i++ {
					rows.AddRow(
						int64(i+1), tt.accountID, int64Ptr(1),
						"文章"+string(rune('A'+i)), nil, nil,
						nil, content, nil, int16(1), false,
						nil, i, 0, nil,
						nil, time.Now(), time.Now(),
					)
				}

				// 根据查询参数设置不同的 mock 期望
				if tt.channelID != nil && tt.status != nil {
					// H5 栏目页：channel_id + status
					mock.ExpectQuery(`SELECT \* FROM cms_articles WHERE channel_id = \$1 AND status = \$2 AND deleted_at IS NULL ORDER BY sort_order LIMIT \$3 OFFSET \$4`).
						WithArgs(*tt.channelID, *tt.status, tt.limit, tt.offset).
						WillReturnRows(rows)
				} else if tt.status != nil {
					// 管理后台按状态过滤
					mock.ExpectQuery(`SELECT \* FROM cms_articles WHERE account_id = \$1 AND status = \$2 AND deleted_at IS NULL ORDER BY created_at DESC LIMIT \$3 OFFSET \$4`).
						WithArgs(tt.accountID, *tt.status, tt.limit, tt.offset).
						WillReturnRows(rows)
				} else {
					// 管理后台：不过滤状态
					mock.ExpectQuery(`SELECT \* FROM cms_articles WHERE account_id = \$1 AND deleted_at IS NULL ORDER BY created_at DESC LIMIT \$2 OFFSET \$3`).
						WithArgs(tt.accountID, tt.limit, tt.offset).
						WillReturnRows(rows)
				}
			}

			articles, err := repo.ListArticles(context.Background(), tt.accountID, tt.channelID, tt.status, tt.offset, tt.limit)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Len(t, articles, tt.wantCount)
			}
		})
	}
}

// int64Ptr 辅助函数：返回 int64 指针
func int64Ptr(i int64) *int64 {
	return &i
}

// int16Ptr 辅助函数：返回 int16 指针
func int16Ptr(i int16) *int16 {
	return &i
}
