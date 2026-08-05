// Package api T12: 微官网 CMS Handler 测试
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

// ========== Mock CMSRepo ==========

type mockCMSRepo struct {
	createChannelFn       func(ctx context.Context, ch *model.Channel) error
	getChannelTreeFn      func(ctx context.Context, accountID int64) ([]model.Channel, error)
	updateChannelFn       func(ctx context.Context, ch *model.Channel) error
	softDeleteChannelFn   func(ctx context.Context, id int64) error
	createArticleFn       func(ctx context.Context, a *model.Article) error
	getArticleFn          func(ctx context.Context, id int64) (*model.Article, error)
	listArticlesFn        func(ctx context.Context, accountID int64, channelID *int64, status *int16, offset, limit int) ([]model.Article, error)
	updateArticleFn       func(ctx context.Context, a *model.Article) error
	softDeleteArticleFn   func(ctx context.Context, id int64) error
}

func (m *mockCMSRepo) CreateChannel(ctx context.Context, ch *model.Channel) error {
	if m.createChannelFn != nil {
		return m.createChannelFn(ctx, ch)
	}
	ch.ID = 1
	ch.CreatedAt = time.Now()
	ch.UpdatedAt = time.Now()
	return nil
}

func (m *mockCMSRepo) GetChannelTree(ctx context.Context, accountID int64) ([]model.Channel, error) {
	if m.getChannelTreeFn != nil {
		return m.getChannelTreeFn(ctx, accountID)
	}
	return nil, nil
}

func (m *mockCMSRepo) UpdateChannel(ctx context.Context, ch *model.Channel) error {
	if m.updateChannelFn != nil {
		return m.updateChannelFn(ctx, ch)
	}
	return nil
}

func (m *mockCMSRepo) SoftDeleteChannel(ctx context.Context, id int64) error {
	if m.softDeleteChannelFn != nil {
		return m.softDeleteChannelFn(ctx, id)
	}
	return nil
}

func (m *mockCMSRepo) CreateArticle(ctx context.Context, a *model.Article) error {
	if m.createArticleFn != nil {
		return m.createArticleFn(ctx, a)
	}
	a.ID = 1
	a.CreatedAt = time.Now()
	a.UpdatedAt = time.Now()
	return nil
}

func (m *mockCMSRepo) GetArticle(ctx context.Context, id int64) (*model.Article, error) {
	if m.getArticleFn != nil {
		return m.getArticleFn(ctx, id)
	}
	return nil, nil
}

func (m *mockCMSRepo) ListArticles(ctx context.Context, accountID int64, channelID *int64, status *int16, offset, limit int) ([]model.Article, error) {
	if m.listArticlesFn != nil {
		return m.listArticlesFn(ctx, accountID, channelID, status, offset, limit)
	}
	return nil, nil
}

func (m *mockCMSRepo) UpdateArticle(ctx context.Context, a *model.Article) error {
	if m.updateArticleFn != nil {
		return m.updateArticleFn(ctx, a)
	}
	return nil
}

func (m *mockCMSRepo) SoftDeleteArticle(ctx context.Context, id int64) error {
	if m.softDeleteArticleFn != nil {
		return m.softDeleteArticleFn(ctx, id)
	}
	return nil
}

// ========== Helpers ==========

func setupCMSTestRouter(handler *CMSHandler) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("tenant_id", int64(1))
		c.Set("user_id", int64(1))
		c.Next()
	})

	// 栏目
	r.GET("/api/v1/cms/channels", handler.ListChannels)
	r.POST("/api/v1/cms/channels", handler.CreateChannel)
	r.PUT("/api/v1/cms/channels/:id", handler.UpdateChannel)
	r.DELETE("/api/v1/cms/channels/:id", handler.DeleteChannel)
	// 文章
	r.GET("/api/v1/cms/articles", handler.ListArticles)
	r.POST("/api/v1/cms/articles", handler.CreateArticle)
	r.GET("/api/v1/cms/articles/:id", handler.GetArticle)
	r.PUT("/api/v1/cms/articles/:id", handler.UpdateArticle)
	r.DELETE("/api/v1/cms/articles/:id", handler.DeleteArticle)
	r.GET("/api/v1/cms/articles/:id/preview", handler.PreviewArticle)

	return r
}

func newCMSHandler(repo CMSRepo) *CMSHandler {
	return NewCMSHandler(repo, zap.NewNop())
}

// ======================== 栏目测试 ========================

func TestCMS_ListChannels_Success(t *testing.T) {
	mockRepo := &mockCMSRepo{
		getChannelTreeFn: func(ctx context.Context, accountID int64) ([]model.Channel, error) {
			return []model.Channel{
				{ID: 1, AccountID: 1, Name: "新闻中心", Level: 0, Status: 1, CreatedAt: time.Now(), UpdatedAt: time.Now()},
				{ID: 2, AccountID: 1, ParentID: ptrInt64(1), Name: "行业动态", Level: 1, Status: 1, CreatedAt: time.Now(), UpdatedAt: time.Now()},
			}, nil
		},
	}

	handler := newCMSHandler(mockRepo)
	r := setupCMSTestRouter(handler)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/cms/channels", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, float64(0), resp["code"])

	data := resp["data"].([]interface{})
	assert.GreaterOrEqual(t, len(data), 1)
}

func TestCMS_CreateChannel_Success(t *testing.T) {
	mockRepo := &mockCMSRepo{}
	handler := newCMSHandler(mockRepo)
	r := setupCMSTestRouter(handler)

	body := `{"name": "公司新闻", "level": 0, "status": 1}`
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/cms/channels", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)

	var resp map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, float64(0), resp["code"])
	assert.Equal(t, "栏目已创建", resp["msg"])
}

func TestCMS_CreateChannel_InvalidJSON(t *testing.T) {
	mockRepo := &mockCMSRepo{}
	handler := newCMSHandler(mockRepo)
	r := setupCMSTestRouter(handler)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/cms/channels", strings.NewReader("invalid"))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestCMS_UpdateChannel_Success(t *testing.T) {
	mockRepo := &mockCMSRepo{}
	handler := newCMSHandler(mockRepo)
	r := setupCMSTestRouter(handler)

	body := `{"name": "更新名称", "status": 0}`
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPut, "/api/v1/cms/channels/1", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	assert.Equal(t, float64(0), resp["code"])
	assert.Equal(t, "栏目已更新", resp["msg"])
}

func TestCMS_UpdateChannel_InvalidID(t *testing.T) {
	handler := newCMSHandler(&mockCMSRepo{})
	r := setupCMSTestRouter(handler)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPut, "/api/v1/cms/channels/abc", strings.NewReader(`{}`))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestCMS_DeleteChannel_Success(t *testing.T) {
	mockRepo := &mockCMSRepo{}
	handler := newCMSHandler(mockRepo)
	r := setupCMSTestRouter(handler)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodDelete, "/api/v1/cms/channels/1", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	assert.Equal(t, float64(0), resp["code"])
	assert.Equal(t, "栏目已删除", resp["msg"])
}

func TestCMS_DeleteChannel_InvalidID(t *testing.T) {
	handler := newCMSHandler(&mockCMSRepo{})
	r := setupCMSTestRouter(handler)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodDelete, "/api/v1/cms/channels/abc", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// ======================== 文章测试 ========================

func TestCMS_ListArticles_Success(t *testing.T) {
	mockRepo := &mockCMSRepo{
		listArticlesFn: func(ctx context.Context, accountID int64, channelID *int64, status *int16, offset, limit int) ([]model.Article, error) {
			return []model.Article{
				{ID: 1, AccountID: 1, Title: strPtr("测试文章"), Status: 1, CreatedAt: time.Now(), UpdatedAt: time.Now()},
			}, nil
		},
	}

	handler := newCMSHandler(mockRepo)
	r := setupCMSTestRouter(handler)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/cms/articles?page=1&size=10", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, float64(0), resp["code"])

	data := resp["data"].(map[string]interface{})
	assert.NotNil(t, data["list"])
	assert.Equal(t, float64(1), data["page"])
}

func TestCMS_ListArticles_WithFilters(t *testing.T) {
	mockRepo := &mockCMSRepo{
		listArticlesFn: func(ctx context.Context, accountID int64, channelID *int64, status *int16, offset, limit int) ([]model.Article, error) {
			assert.NotNil(t, channelID)
			assert.Equal(t, int64(5), *channelID)
			assert.NotNil(t, status)
			assert.Equal(t, int16(1), *status)
			return nil, nil
		},
	}

	handler := newCMSHandler(mockRepo)
	r := setupCMSTestRouter(handler)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/cms/articles?channel_id=5&status=1", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestCMS_GetArticle_Success(t *testing.T) {
	mockRepo := &mockCMSRepo{
		getArticleFn: func(ctx context.Context, id int64) (*model.Article, error) {
			return &model.Article{
				ID: id, AccountID: 1, Title: strPtr("测试"),
				Content: json.RawMessage(`{}`), Status: 1,
				CreatedAt: time.Now(), UpdatedAt: time.Now(),
			}, nil
		},
	}

	handler := newCMSHandler(mockRepo)
	r := setupCMSTestRouter(handler)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/cms/articles/1", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	assert.Equal(t, float64(0), resp["code"])
	assert.NotNil(t, resp["data"])
}

func TestCMS_GetArticle_NotFound(t *testing.T) {
	mockRepo := &mockCMSRepo{
		getArticleFn: func(ctx context.Context, id int64) (*model.Article, error) {
			return nil, nil
		},
	}

	handler := newCMSHandler(mockRepo)
	r := setupCMSTestRouter(handler)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/cms/articles/999", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestCMS_CreateArticle_Success(t *testing.T) {
	mockRepo := &mockCMSRepo{}
	handler := newCMSHandler(mockRepo)
	r := setupCMSTestRouter(handler)

	body := `{"content": {"type":"doc","content":[]}, "title": "新文章", "status": 1}`
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/cms/articles", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)

	var resp map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, float64(0), resp["code"])
	assert.Equal(t, "文章已创建", resp["msg"])
}

func TestCMS_CreateArticle_InvalidJSON(t *testing.T) {
	handler := newCMSHandler(&mockCMSRepo{})
	r := setupCMSTestRouter(handler)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/cms/articles", strings.NewReader("bad json"))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestCMS_UpdateArticle_Success(t *testing.T) {
	mockRepo := &mockCMSRepo{}
	handler := newCMSHandler(mockRepo)
	r := setupCMSTestRouter(handler)

	body := `{"title": "更新标题", "status": 1}`
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPut, "/api/v1/cms/articles/1", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	assert.Equal(t, float64(0), resp["code"])
	assert.Equal(t, "文章已更新", resp["msg"])
}

func TestCMS_DeleteArticle_Success(t *testing.T) {
	mockRepo := &mockCMSRepo{}
	handler := newCMSHandler(mockRepo)
	r := setupCMSTestRouter(handler)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodDelete, "/api/v1/cms/articles/1", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	assert.Equal(t, float64(0), resp["code"])
	assert.Equal(t, "文章已删除", resp["msg"])
}

func TestCMS_PreviewArticle_Success(t *testing.T) {
	mockRepo := &mockCMSRepo{
		getArticleFn: func(ctx context.Context, id int64) (*model.Article, error) {
			return &model.Article{
				ID: id, AccountID: 1, Title: strPtr("预览文章"),
				Content: json.RawMessage(`{"type":"doc"}`), Status: 1,
				CreatedAt: time.Now(), UpdatedAt: time.Now(),
			}, nil
		},
	}

	handler := newCMSHandler(mockRepo)
	r := setupCMSTestRouter(handler)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/cms/articles/1/preview", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

// ======================== 未认证测试 ========================

func TestCMS_Unauthenticated(t *testing.T) {
	handler := newCMSHandler(&mockCMSRepo{})
	r := gin.New()
	r.GET("/api/v1/cms/channels", handler.ListChannels)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/cms/channels", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

// ======================== 错误处理测试 ========================

func TestCMS_ListChannels_RepoError(t *testing.T) {
	mockRepo := &mockCMSRepo{
		getChannelTreeFn: func(ctx context.Context, accountID int64) ([]model.Channel, error) {
			return nil, errors.New("database error")
		},
	}

	handler := newCMSHandler(mockRepo)
	r := setupCMSTestRouter(handler)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/cms/channels", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestCMS_GetArticle_RepoError(t *testing.T) {
	mockRepo := &mockCMSRepo{
		getArticleFn: func(ctx context.Context, id int64) (*model.Article, error) {
			return nil, errors.New("connection refused")
		},
	}

	handler := newCMSHandler(mockRepo)
	r := setupCMSTestRouter(handler)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/cms/articles/1", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

// ======================== 栏目树构建测试 ========================

func TestBuildChannelTree(t *testing.T) {
	channels := []model.Channel{
		{ID: 1, Name: "根栏目1", Level: 0, Status: 1, CreatedAt: time.Now(), UpdatedAt: time.Now()},
		{ID: 2, ParentID: ptrInt64(1), Name: "子栏目1", Level: 1, Status: 1, CreatedAt: time.Now(), UpdatedAt: time.Now()},
		{ID: 3, ParentID: ptrInt64(1), Name: "子栏目2", Level: 1, Status: 1, CreatedAt: time.Now(), UpdatedAt: time.Now()},
		{ID: 4, Name: "根栏目2", Level: 0, Status: 1, CreatedAt: time.Now(), UpdatedAt: time.Now()},
	}

	tree := buildChannelTree(channels)

	assert.Len(t, tree, 2, "应该有2个根栏目")
	assert.Equal(t, "根栏目1", tree[0].Name)
	assert.Len(t, tree[0].Children, 2)
	assert.Equal(t, "子栏目1", tree[0].Children[0].Name)
	assert.Equal(t, "子栏目2", tree[0].Children[1].Name)
	assert.Equal(t, "根栏目2", tree[1].Name)
	assert.Len(t, tree[1].Children, 0)
}

// ======================== 帮助函数 ========================

func ptrInt64(v int64) *int64 {
	return &v
}

func strPtr(s string) *string {
	return &s
}

// ======================== 接口满足性测试 ========================

func TestCMSRepoInterface(t *testing.T) {
	// 验证 mockCMSRepo 满足 CMSRepo 接口
	var _ CMSRepo = (*mockCMSRepo)(nil)
	assert.True(t, true, "mockCMSRepo implements CMSRepo")
}

// ======================== 分页参数验证 ========================

func TestCMS_ListArticles_DefaultPagination(t *testing.T) {
	mockRepo := &mockCMSRepo{
		listArticlesFn: func(ctx context.Context, accountID int64, channelID *int64, status *int16, offset, limit int) ([]model.Article, error) {
			assert.Equal(t, 0, offset)
			assert.Equal(t, 20, limit)
			return nil, nil
		},
	}

	handler := newCMSHandler(mockRepo)
	r := setupCMSTestRouter(handler)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/cms/articles", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}
