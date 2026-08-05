package reply

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

// replyColumns returns the full column list for auto_reply_rules
func replyColumns() []string {
	return []string{
		"id", "account_id", "keyword", "match_type", "reply_type",
		"reply_content", "reply_title", "reply_desc", "reply_cover_url", "reply_url",
		"status", "sort_order",
		"deleted_at", "created_at", "updated_at",
	}
}

// replyRowValues returns a row of test data
func replyRowValues(id, accountID int64, keyword string, matchType, replyType int16) []driver.Value {
	var kw interface{} = keyword
	if keyword == "" {
		kw = nil
	}
	return []driver.Value{
		id, accountID, kw, matchType, replyType,
		"reply content text", nil, nil, nil, nil,
		int16(1), 0,
		nil, time.Now(), time.Now(),
	}
}

// newMockRepo creates Repo with mocked DB
func newMockRepo(t *testing.T) (*Repo, sqlmock.Sqlmock) {
	t.Helper()
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	sqlxDB := sqlx.NewDb(db, "postgres")
	return &Repo{DB: sqlxDB}, mock
}

func TestRepoExists(t *testing.T) {
	repo, _ := newMockRepo(t)
	defer repo.DB.Close()
	assert.NotNil(t, repo)
	assert.NotNil(t, repo.DB)
}

func TestListByAccountID(t *testing.T) {
	tests := []struct {
		name      string
		accountID int64
		wantCount int
		wantErr   bool
	}{
		{name: "有多个规则", accountID: 1, wantCount: 3, wantErr: false},
		{name: "有一个规则", accountID: 2, wantCount: 1, wantErr: false},
		{name: "无规则", accountID: 999, wantCount: 0, wantErr: false},
		{name: "无效 account_id", accountID: 0, wantCount: 0, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo, mock := newMockRepo(t)
			defer repo.DB.Close()

			if !tt.wantErr {
				columns := replyColumns()
				rows := sqlmock.NewRows(columns)
				for i := 0; i < tt.wantCount; i++ {
					kw := ""
					if i > 0 {
						kw = "keyword" + string(rune('A'+i))
					}
					rows.AddRow(replyRowValues(int64(i+1), tt.accountID, kw, 0, 1)...)
				}
				mock.ExpectQuery(`SELECT \* FROM auto_reply_rules WHERE account_id = \$1 AND deleted_at IS NULL`).
					WithArgs(tt.accountID).
					WillReturnRows(rows)
			}

			rules, err := repo.ListByAccountID(context.Background(), tt.accountID)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Len(t, rules, tt.wantCount)
			}
		})
	}
}

func TestGetByID(t *testing.T) {
	t.Run("找到规则", func(t *testing.T) {
		repo, mock := newMockRepo(t)
		defer repo.DB.Close()

		columns := replyColumns()
		rows := sqlmock.NewRows(columns).AddRow(replyRowValues(1, 10, "hello", 0, 1)...)
		mock.ExpectQuery(`SELECT \* FROM auto_reply_rules WHERE id = \$1 AND deleted_at IS NULL`).
			WithArgs(int64(1)).
			WillReturnRows(rows)

		rule, err := repo.GetByID(context.Background(), 1)
		assert.NoError(t, err)
		require.NotNil(t, rule)
		assert.Equal(t, int64(1), rule.ID)
		assert.Equal(t, int64(10), rule.AccountID)
	})

	t.Run("规则不存在", func(t *testing.T) {
		repo, mock := newMockRepo(t)
		defer repo.DB.Close()

		mock.ExpectQuery(`SELECT \* FROM auto_reply_rules WHERE id = \$1 AND deleted_at IS NULL`).
			WithArgs(int64(999)).
			WillReturnError(sqlmock.ErrCancelled)

		// sqlmock.ErrCancelled won't produce sql.ErrNoRows — let's use WillReturnRows with no data instead
		// We'll use a different approach: empty rows
		_ = mock
	})

	t.Run("无效 id", func(t *testing.T) {
		repo, _ := newMockRepo(t)
		defer repo.DB.Close()

		rule, err := repo.GetByID(context.Background(), 0)
		assert.Error(t, err)
		assert.Nil(t, rule)
	})
}

func TestGetByID_NotFound(t *testing.T) {
	repo, mock := newMockRepo(t)
	defer repo.DB.Close()

	// Return empty rows for sqlx GetContext — will result in sql.ErrNoRows
	mock.ExpectQuery(`SELECT \* FROM auto_reply_rules WHERE id = \$1 AND deleted_at IS NULL`).
		WithArgs(int64(999)).
		WillReturnRows(sqlmock.NewRows(replyColumns()))

	rule, err := repo.GetByID(context.Background(), 999)
	assert.NoError(t, err) // GetByID returns nil, nil for not found
	assert.Nil(t, rule)
}

func TestCreate(t *testing.T) {
	repo, mock := newMockRepo(t)
	defer repo.DB.Close()

	keyword := "帮助"
	title := "图文标题"
	desc := "图文描述"
	coverURL := "https://example.com/cover.jpg"
	linkURL := "https://example.com"

	rule := &model.AutoReplyRule{
		AccountID:     1,
		Keyword:       &keyword,
		MatchType:     1, // 模糊匹配
		ReplyType:     2, // 图文
		ReplyContent:  `[{"title":"图文标题","desc":"图文描述","cover":"https://example.com/cover.jpg","url":"https://example.com"}]`,
		ReplyTitle:    &title,
		ReplyDesc:     &desc,
		ReplyCoverURL: &coverURL,
		ReplyURL:      &linkURL,
		Status:        1,
		SortOrder:     0,
	}

	mock.ExpectQuery(`INSERT INTO auto_reply_rules`).
		WithArgs(
			rule.AccountID, rule.Keyword, rule.MatchType, rule.ReplyType,
			rule.ReplyContent, rule.ReplyTitle, rule.ReplyDesc, rule.ReplyCoverURL, rule.ReplyURL,
			rule.Status, rule.SortOrder,
		).
		WillReturnRows(sqlmock.NewRows([]string{"id", "created_at", "updated_at"}).
			AddRow(int64(1), time.Now(), time.Now()))

	err := repo.Create(context.Background(), rule)
	assert.NoError(t, err)
	assert.Equal(t, int64(1), rule.ID)
	assert.False(t, rule.CreatedAt.IsZero())
}

func TestUpdate(t *testing.T) {
	repo, mock := newMockRepo(t)
	defer repo.DB.Close()

	keyword := "帮助"
	rule := &model.AutoReplyRule{
		ID:        1,
		AccountID: 1,
		Keyword:   &keyword,
		MatchType: 0,
		ReplyType: 1,
	}

	mock.ExpectExec(`UPDATE auto_reply_rules SET`).
		WithArgs(
			rule.Keyword, rule.MatchType, rule.ReplyType,
			rule.ReplyContent, rule.ReplyTitle, rule.ReplyDesc,
			rule.ReplyCoverURL, rule.ReplyURL,
			rule.Status, rule.SortOrder, rule.ID,
		).
		WillReturnResult(sqlmock.NewResult(0, 1))

	err := repo.Update(context.Background(), rule)
	assert.NoError(t, err)
}

func TestUpdate_NotFound(t *testing.T) {
	repo, mock := newMockRepo(t)
	defer repo.DB.Close()

	keyword := "帮助"
	rule := &model.AutoReplyRule{
		ID:        999,
		AccountID: 1,
		Keyword:   &keyword,
		MatchType: 0,
		ReplyType: 1,
	}

	mock.ExpectExec(`UPDATE auto_reply_rules SET`).
		WithArgs(
			rule.Keyword, rule.MatchType, rule.ReplyType,
			rule.ReplyContent, rule.ReplyTitle, rule.ReplyDesc,
			rule.ReplyCoverURL, rule.ReplyURL,
			rule.Status, rule.SortOrder, rule.ID,
		).
		WillReturnResult(sqlmock.NewResult(0, 0))

	err := repo.Update(context.Background(), rule)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "不存在或已删除")
}

func TestSoftDelete(t *testing.T) {
	t.Run("删除成功", func(t *testing.T) {
		repo, mock := newMockRepo(t)
		defer repo.DB.Close()

		mock.ExpectExec(`UPDATE auto_reply_rules SET deleted_at = NOW\(\) WHERE id = \$1 AND deleted_at IS NULL`).
			WithArgs(int64(1)).
			WillReturnResult(sqlmock.NewResult(0, 1))

		deleted, err := repo.SoftDelete(context.Background(), 1)
		assert.NoError(t, err)
		assert.True(t, deleted)
	})

	t.Run("规则已删除", func(t *testing.T) {
		repo, mock := newMockRepo(t)
		defer repo.DB.Close()

		mock.ExpectExec(`UPDATE auto_reply_rules SET deleted_at = NOW\(\) WHERE id = \$1 AND deleted_at IS NULL`).
			WithArgs(int64(999)).
			WillReturnResult(sqlmock.NewResult(0, 0))

		deleted, err := repo.SoftDelete(context.Background(), 999)
		assert.NoError(t, err)
		assert.False(t, deleted)
	})
}

func TestModelAutoReplyRule(t *testing.T) {
	// Verify model struct fields exist
	rule := model.AutoReplyRule{
		ID:       1,
		AccountID: 10,
		MatchType: 0,
		ReplyType: 1,
		Status:   1,
	}
	assert.Equal(t, int64(1), rule.ID)
	assert.Equal(t, int64(10), rule.AccountID)
	assert.Equal(t, int16(0), rule.MatchType)
	assert.Equal(t, int16(1), rule.ReplyType)
	assert.Equal(t, int16(1), rule.Status)
	assert.Equal(t, 0, rule.SortOrder)
}
