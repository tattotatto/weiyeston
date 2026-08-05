// Package api T13: 快讯管理 Handler 测试
package api

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/weiyeston/weiyeston-v2/internal/model"
)

// ========== Mock NewsRepo ==========

type mockNewsRepo struct {
	listNewsFn       func(ctx context.Context, accountID int64, offset, limit int) ([]model.QuickNews, error)
	getNewsFn        func(ctx context.Context, id int64) (*model.QuickNews, error)
	createNewsFn     func(ctx context.Context, n *model.QuickNews) error
	updateNewsFn     func(ctx context.Context, n *model.QuickNews) error
	softDeleteNewsFn func(ctx context.Context, id int64) error
	listUsersFn      func(ctx context.Context, accountID int64, offset, limit int) ([]model.QuickNewsUser, error)
}

func (m *mockNewsRepo) ListNews(ctx context.Context, accountID int64, offset, limit int) ([]model.QuickNews, error) {
	if m.listNewsFn != nil {
		return m.listNewsFn(ctx, accountID, offset, limit)
	}
	return nil, nil
}

func (m *mockNewsRepo) GetNews(ctx context.Context, id int64) (*model.QuickNews, error) {
	if m.getNewsFn != nil {
		return m.getNewsFn(ctx, id)
	}
	return nil, nil
}

func (m *mockNewsRepo) CreateNews(ctx context.Context, n *model.QuickNews) error {
	if m.createNewsFn != nil {
		return m.createNewsFn(ctx, n)
	}
	n.ID = 1
	n.CreatedAt = time.Now()
	n.UpdatedAt = time.Now()
	return nil
}

func (m *mockNewsRepo) UpdateNews(ctx context.Context, n *model.QuickNews) error {
	if m.updateNewsFn != nil {
		return m.updateNewsFn(ctx, n)
	}
	return nil
}

func (m *mockNewsRepo) SoftDeleteNews(ctx context.Context, id int64) error {
	if m.softDeleteNewsFn != nil {
		return m.softDeleteNewsFn(ctx, id)
	}
	return nil
}

func (m *mockNewsRepo) ListUsers(ctx context.Context, accountID int64, offset, limit int) ([]model.QuickNewsUser, error) {
	if m.listUsersFn != nil {
		return m.listUsersFn(ctx, accountID, offset, limit)
	}
	return nil, nil
}

// ========== Helpers ==========

func setupNewsTestRouter(handler *NewsHandler) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("tenant_id", int64(1))
		c.Set("user_id", int64(1))
		c.Next()
	})

	r.GET("/api/v1/quicknews/news", handler.ListNews)
	r.POST("/api/v1/quicknews/news", handler.CreateNews)
	r.GET("/api/v1/quicknews/news/:id", handler.GetNews)
	r.PUT("/api/v1/quicknews/news/:id", handler.UpdateNews)
	r.DELETE("/api/v1/quicknews/news/:id", handler.DeleteNews)
	r.GET("/api/v1/quicknews/users", handler.ListUsers)

	return r
}

func newNewsHandler(repo NewsRepo) *NewsHandler {
	return NewNewsHandler(repo, zap.NewNop())
}

// ======================== News List Tests ========================

func TestNews_ListNews_Success(t *testing.T) {
	mockRepo := &mockNewsRepo{
		listNewsFn: func(ctx context.Context, accountID int64, offset, limit int) ([]model.QuickNews, error) {
			return []model.QuickNews{
				{ID: 1, AccountID: 1, ChannelID: 1, Content: "测试快讯", Status: 1, CreatedAt: time.Now(), UpdatedAt: time.Now()},
			}, nil
		},
	}

	handler := newNewsHandler(mockRepo)
	r := setupNewsTestRouter(handler)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/quicknews/news?page=1&size=10", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, float64(0), resp["code"])
}

func TestNews_ListNews_Empty(t *testing.T) {
	mockRepo := &mockNewsRepo{}
	handler := newNewsHandler(mockRepo)
	r := setupNewsTestRouter(handler)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/quicknews/news", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

// ======================== News Create Tests ========================

func TestNews_CreateNews_Success(t *testing.T) {
	mockRepo := &mockNewsRepo{}
	handler := newNewsHandler(mockRepo)
	r := setupNewsTestRouter(handler)

	body := `{"channel_id": 1, "content": "这是一条测试快讯", "status": 1}`
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/quicknews/news", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)

	var resp map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, float64(0), resp["code"])
	assert.Equal(t, "快讯已发布", resp["msg"])
}

func TestNews_CreateNews_InvalidJSON(t *testing.T) {
	handler := newNewsHandler(&mockNewsRepo{})
	r := setupNewsTestRouter(handler)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/quicknews/news", strings.NewReader("bad"))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestNews_CreateNews_ValidationError(t *testing.T) {
	handler := newNewsHandler(&mockNewsRepo{})
	r := setupNewsTestRouter(handler)

	// Missing channel_id
	body := `{"content": "no channel"}`
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/quicknews/news", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// ======================== News Get Tests ========================

func TestNews_GetNews_Success(t *testing.T) {
	mockRepo := &mockNewsRepo{
		getNewsFn: func(ctx context.Context, id int64) (*model.QuickNews, error) {
			return &model.QuickNews{
				ID: id, AccountID: 1, ChannelID: 1, Content: "详情",
				Status: 1, CreatedAt: time.Now(), UpdatedAt: time.Now(),
			}, nil
		},
	}

	handler := newNewsHandler(mockRepo)
	r := setupNewsTestRouter(handler)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/quicknews/news/1", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestNews_GetNews_NotFound(t *testing.T) {
	mockRepo := &mockNewsRepo{
		getNewsFn: func(ctx context.Context, id int64) (*model.QuickNews, error) {
			return nil, nil
		},
	}

	handler := newNewsHandler(mockRepo)
	r := setupNewsTestRouter(handler)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/quicknews/news/999", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

// ======================== News Update Tests ========================

func TestNews_UpdateNews_Success(t *testing.T) {
	mockRepo := &mockNewsRepo{
		getNewsFn: func(ctx context.Context, id int64) (*model.QuickNews, error) {
			return &model.QuickNews{
				ID: id, AccountID: 1, ChannelID: 1, Content: "旧内容",
				Status: 1, CreatedAt: time.Now(), UpdatedAt: time.Now(),
			}, nil
		},
	}
	handler := newNewsHandler(mockRepo)
	r := setupNewsTestRouter(handler)

	body := `{"content": "更新后的内容", "is_top": true}`
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPut, "/api/v1/quicknews/news/1", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	assert.Equal(t, float64(0), resp["code"])
	assert.Equal(t, "快讯已更新", resp["msg"])
}

func TestNews_UpdateNews_InvalidID(t *testing.T) {
	handler := newNewsHandler(&mockNewsRepo{})
	r := setupNewsTestRouter(handler)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPut, "/api/v1/quicknews/news/abc", strings.NewReader(`{}`))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// ======================== News Delete Tests ========================

func TestNews_DeleteNews_Success(t *testing.T) {
	mockRepo := &mockNewsRepo{}
	handler := newNewsHandler(mockRepo)
	r := setupNewsTestRouter(handler)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodDelete, "/api/v1/quicknews/news/1", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	assert.Equal(t, float64(0), resp["code"])
	assert.Equal(t, "快讯已删除", resp["msg"])
}

func TestNews_DeleteNews_InvalidID(t *testing.T) {
	handler := newNewsHandler(&mockNewsRepo{})
	r := setupNewsTestRouter(handler)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodDelete, "/api/v1/quicknews/news/abc", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// ======================== Users Tests ========================

func TestNews_ListUsers_Success(t *testing.T) {
	mockRepo := &mockNewsRepo{
		listUsersFn: func(ctx context.Context, accountID int64, offset, limit int) ([]model.QuickNewsUser, error) {
			return []model.QuickNewsUser{
				{ID: 1, AccountID: 1, Openid: "openid_001", Nickname: strPtr("用户A"), Status: 1, CreatedAt: time.Now()},
			}, nil
		},
	}

	handler := newNewsHandler(mockRepo)
	r := setupNewsTestRouter(handler)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/quicknews/users?account_id=1", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, float64(0), resp["code"])
}

// ======================== 未认证测试 ========================

func TestNews_Unauthenticated(t *testing.T) {
	handler := newNewsHandler(&mockNewsRepo{})
	r := gin.New()
	r.GET("/api/v1/quicknews/news", handler.ListNews)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/quicknews/news", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

// ======================== 错误处理测试 ========================

func TestNews_ListNews_RepoError(t *testing.T) {
	mockRepo := &mockNewsRepo{
		listNewsFn: func(ctx context.Context, accountID int64, offset, limit int) ([]model.QuickNews, error) {
			return nil, errors.New("database error")
		},
	}

	handler := newNewsHandler(mockRepo)
	r := setupNewsTestRouter(handler)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/quicknews/news", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

// ======================== 接口满足性测试 ========================

func TestNewsRepoInterface(t *testing.T) {
	var _ NewsRepo = (*mockNewsRepo)(nil)
	assert.True(t, true, "mockNewsRepo implements NewsRepo")
}

// ======================== VO 转换测试 ========================

func TestNewsVO_StatusText(t *testing.T) {
	tests := []struct {
		status     int16
		statusText string
	}{
		{0, "草稿"},
		{1, "已发布"},
		{2, "隐藏"},
	}

	for _, tt := range tests {
		n := &model.QuickNews{ID: 1, Status: tt.status, CreatedAt: time.Now(), UpdatedAt: time.Now()}
		vo := toNewsVO(n)
		assert.Equal(t, tt.statusText, vo.StatusText, "status %d should map to %s", tt.status, tt.statusText)
	}
}
