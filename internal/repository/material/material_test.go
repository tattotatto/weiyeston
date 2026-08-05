package material

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

// materialColumns returns the full column list for materials
func materialColumns() []string {
	return []string{
		"id", "account_id", "media_id", "type", "name",
		"url", "thumbnail_url", "file_size", "width", "height",
		"format", "deleted_at", "created_at", "updated_at",
	}
}

// materialRowValues returns a row of test data
func materialRowValues(id, accountID int64, mediaID *string, typ, name string, fileSize *int64, width, height *int, format *string) []driver.Value {
	return []driver.Value{
		id, accountID, mediaID, typ, name,
		"https://example.com/file.jpg", nil, fileSize, width, height,
		format, nil, time.Now(), time.Now(),
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

func TestListByAccount(t *testing.T) {
	tests := []struct {
		name         string
		accountID    int64
		materialType string
		wantCount    int
		wantErr      bool
	}{
		{name: "有多个素材", accountID: 1, materialType: "", wantCount: 3, wantErr: false},
		{name: "按类型筛选 image", accountID: 1, materialType: "image", wantCount: 2, wantErr: false},
		{name: "无素材", accountID: 999, materialType: "", wantCount: 0, wantErr: false},
		{name: "无效 account_id", accountID: 0, materialType: "", wantCount: 0, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo, mock := newMockRepo(t)
			defer repo.DB.Close()

			if !tt.wantErr {
				columns := materialColumns()
				rows := sqlmock.NewRows(columns)
				for i := 0; i < tt.wantCount; i++ {
					fileName := "test.png"
					if tt.materialType == "" && i > 0 {
						// mix types when no filter
					}
					rows.AddRow(materialRowValues(int64(i+1), tt.accountID, nil, "image", fileName, nil, nil, nil, nil)...)
				}
				mock.ExpectQuery(`SELECT \* FROM materials WHERE account_id = \$1 AND deleted_at IS NULL`).
					WithArgs(expectArgsForList(tt.accountID, tt.materialType, 20, 0)...).
					WillReturnRows(rows)
			}

			_, err := repo.ListByAccount(context.Background(), tt.accountID, tt.materialType, 0, 20)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func expectArgsForList(accountID int64, materialType string, limit, offset int) []driver.Value {
	if materialType != "" {
		return []driver.Value{accountID, materialType, limit, offset}
	}
	return []driver.Value{accountID, limit, offset}
}

func TestGetByID(t *testing.T) {
	t.Run("找到素材", func(t *testing.T) {
		repo, mock := newMockRepo(t)
		defer repo.DB.Close()

		mediaID := "wx_media_123"
		columns := materialColumns()
		rows := sqlmock.NewRows(columns).AddRow(materialRowValues(1, 10, &mediaID, "image", "test.png", nil, nil, nil, nil)...)
		mock.ExpectQuery(`SELECT \* FROM materials WHERE id = \$1 AND deleted_at IS NULL`).
			WithArgs(int64(1)).
			WillReturnRows(rows)

		m, err := repo.GetByID(context.Background(), 1)
		assert.NoError(t, err)
		require.NotNil(t, m)
		assert.Equal(t, int64(1), m.ID)
		assert.Equal(t, int64(10), m.AccountID)
		assert.Equal(t, "image", m.Type)
	})

	t.Run("素材不存在", func(t *testing.T) {
		repo, mock := newMockRepo(t)
		defer repo.DB.Close()

		mock.ExpectQuery(`SELECT \* FROM materials WHERE id = \$1 AND deleted_at IS NULL`).
			WithArgs(int64(999)).
			WillReturnRows(sqlmock.NewRows(materialColumns()))

		m, err := repo.GetByID(context.Background(), 999)
		assert.NoError(t, err)
		assert.Nil(t, m)
	})

	t.Run("无效 id", func(t *testing.T) {
		repo, _ := newMockRepo(t)
		defer repo.DB.Close()

		m, err := repo.GetByID(context.Background(), 0)
		assert.Error(t, err)
		assert.Nil(t, m)
	})
}

func TestCreate(t *testing.T) {
	repo, mock := newMockRepo(t)
	defer repo.DB.Close()

	fileName := "banner.jpg"
	fileSize := int64(102400)
	width := 800
	height := 600
	format := "jpg"

	m := &model.Material{
		AccountID: 1,
		Type:      "image",
		Name:      &fileName,
		URL:       "https://example.com/uploads/banner.jpg",
		FileSize:  &fileSize,
		Width:     &width,
		Height:    &height,
		Format:    &format,
	}

	mock.ExpectQuery(`INSERT INTO materials`).
		WithArgs(
			m.AccountID, m.MediaID, m.Type, m.Name, m.URL,
			m.ThumbnailURL, m.FileSize, m.Width, m.Height, m.Format,
		).
		WillReturnRows(sqlmock.NewRows([]string{"id", "created_at", "updated_at"}).
			AddRow(int64(1), time.Now(), time.Now()))

	err := repo.Create(context.Background(), m)
	assert.NoError(t, err)
	assert.Equal(t, int64(1), m.ID)
	assert.False(t, m.CreatedAt.IsZero())
}

func TestCreate_WithMediaID(t *testing.T) {
	repo, mock := newMockRepo(t)
	defer repo.DB.Close()

	fileName := "voice.mp3"
	mediaID := "wx_media_voice_001"
	fileSize := int64(51200)
	format := "mp3"

	m := &model.Material{
		AccountID: 1,
		MediaID:   &mediaID,
		Type:      "voice",
		Name:      &fileName,
		URL:       "https://example.com/uploads/voice.mp3",
		FileSize:  &fileSize,
		Format:    &format,
	}

	mock.ExpectQuery(`INSERT INTO materials`).
		WithArgs(
			m.AccountID, m.MediaID, m.Type, m.Name, m.URL,
			m.ThumbnailURL, m.FileSize, m.Width, m.Height, m.Format,
		).
		WillReturnRows(sqlmock.NewRows([]string{"id", "created_at", "updated_at"}).
			AddRow(int64(2), time.Now(), time.Now()))

	err := repo.Create(context.Background(), m)
	assert.NoError(t, err)
	assert.Equal(t, int64(2), m.ID)
}

func TestSoftDelete(t *testing.T) {
	t.Run("删除成功", func(t *testing.T) {
		repo, mock := newMockRepo(t)
		defer repo.DB.Close()

		mock.ExpectExec(`UPDATE materials SET deleted_at = NOW\(\) WHERE id = \$1 AND deleted_at IS NULL`).
			WithArgs(int64(1)).
			WillReturnResult(sqlmock.NewResult(0, 1))

		deleted, err := repo.SoftDelete(context.Background(), 1)
		assert.NoError(t, err)
		assert.True(t, deleted)
	})

	t.Run("素材已删除", func(t *testing.T) {
		repo, mock := newMockRepo(t)
		defer repo.DB.Close()

		mock.ExpectExec(`UPDATE materials SET deleted_at = NOW\(\) WHERE id = \$1 AND deleted_at IS NULL`).
			WithArgs(int64(999)).
			WillReturnResult(sqlmock.NewResult(0, 0))

		deleted, err := repo.SoftDelete(context.Background(), 999)
		assert.NoError(t, err)
		assert.False(t, deleted)
	})
}

func TestCountByAccount(t *testing.T) {
	tests := []struct {
		name         string
		accountID    int64
		materialType string
		wantCount    int
		wantErr      bool
	}{
		{name: "统计所有类型", accountID: 1, materialType: "", wantCount: 10, wantErr: false},
		{name: "统计 image 类型", accountID: 1, materialType: "image", wantCount: 7, wantErr: false},
		{name: "统计 voice 类型", accountID: 1, materialType: "voice", wantCount: 0, wantErr: false},
		{name: "无效 account_id", accountID: 0, materialType: "", wantCount: 0, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo, mock := newMockRepo(t)
			defer repo.DB.Close()

			if !tt.wantErr {
				countColumns := []string{"count"}
				rows := sqlmock.NewRows(countColumns).AddRow(tt.wantCount)

				args := []driver.Value{tt.accountID}
				if tt.materialType != "" {
					args = append(args, tt.materialType)
				}
				mock.ExpectQuery(`SELECT COUNT\(\*\) FROM materials WHERE account_id = \$1 AND deleted_at IS NULL`).
					WithArgs(args...).
					WillReturnRows(rows)
			}

			count, err := repo.CountByAccount(context.Background(), tt.accountID, tt.materialType)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Equal(t, 0, count)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.wantCount, count)
			}
		})
	}
}

func TestModelMaterial(t *testing.T) {
	m := model.Material{
		ID:        1,
		AccountID: 10,
		Type:      "image",
		URL:       "https://example.com/img.png",
	}
	assert.Equal(t, int64(1), m.ID)
	assert.Equal(t, int64(10), m.AccountID)
	assert.Equal(t, "image", m.Type)
	assert.Equal(t, "https://example.com/img.png", m.URL)
}
