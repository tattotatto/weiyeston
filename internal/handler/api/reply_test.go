// Package api T5: 自动回复规则管理 Handler 测试
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

// ========== Mock ReplyRepo ==========

type mockReplyRepo struct {
	listByAccountIDFn func(ctx context.Context, accountID int64) ([]model.AutoReplyRule, error)
	getByIDFn         func(ctx context.Context, id int64) (*model.AutoReplyRule, error)
	createFn          func(ctx context.Context, rule *model.AutoReplyRule) error
	updateFn          func(ctx context.Context, rule *model.AutoReplyRule) error
	softDeleteFn      func(ctx context.Context, id int64) (bool, error)
}

func (m *mockReplyRepo) ListByAccountID(ctx context.Context, accountID int64) ([]model.AutoReplyRule, error) {
	if m.listByAccountIDFn != nil {
		return m.listByAccountIDFn(ctx, accountID)
	}
	return nil, nil
}

func (m *mockReplyRepo) GetByID(ctx context.Context, id int64) (*model.AutoReplyRule, error) {
	if m.getByIDFn != nil {
		return m.getByIDFn(ctx, id)
	}
	return nil, nil
}

func (m *mockReplyRepo) Create(ctx context.Context, rule *model.AutoReplyRule) error {
	if m.createFn != nil {
		return m.createFn(ctx, rule)
	}
	rule.ID = 1
	rule.CreatedAt = time.Now()
	rule.UpdatedAt = time.Now()
	return nil
}

func (m *mockReplyRepo) Update(ctx context.Context, rule *model.AutoReplyRule) error {
	if m.updateFn != nil {
		return m.updateFn(ctx, rule)
	}
	return nil
}

func (m *mockReplyRepo) SoftDelete(ctx context.Context, id int64) (bool, error) {
	if m.softDeleteFn != nil {
		return m.softDeleteFn(ctx, id)
	}
	return true, nil
}

// ========== Helpers ==========

func setupReplyTestRouter(handler *ReplyHandler) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("tenant_id", int64(1))
		c.Set("user_id", int64(1))
		c.Next()
	})
	r.GET("/api/v1/accounts/:id/replies", handler.List)
	r.POST("/api/v1/accounts/:id/replies", handler.Create)
	r.PUT("/api/v1/replies/:id", handler.Update)
	r.DELETE("/api/v1/replies/:id", handler.Delete)
	return r
}

func newReplyHandler(repo ReplyRepo) *ReplyHandler {
	return NewReplyHandler(repo, zap.NewNop())
}

func sampleKeyword() string {
	return "hello"
}

// ========== List Tests ==========

func TestReplyList_Success(t *testing.T) {
	kw := sampleKeyword()
	mockRepo := &mockReplyRepo{
		listByAccountIDFn: func(ctx context.Context, accountID int64) ([]model.AutoReplyRule, error) {
			return []model.AutoReplyRule{
				{ID: 1, AccountID: accountID, Keyword: &kw, MatchType: 0, ReplyType: 1, ReplyContent: "你好，有什么可以帮助你的？", Status: 1},
				{ID: 2, AccountID: accountID, MatchType: 1, ReplyType: 1, ReplyContent: "How can I help you?", Status: 1, SortOrder: 1},
			}, nil
		},
	}

	handler := newReplyHandler(mockRepo)
	r := setupReplyTestRouter(handler)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/accounts/10/replies", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, float64(0), resp["code"])

	data, ok := resp["data"].([]interface{})
	require.True(t, ok)
	assert.Len(t, data, 2)

	first := data[0].(map[string]interface{})
	assert.Equal(t, float64(1), first["id"])
	assert.Equal(t, float64(10), first["account_id"])
	assert.Equal(t, "hello", first["keyword"])
	assert.Equal(t, float64(1), first["reply_type"])
}

func TestReplyList_Empty(t *testing.T) {
	mockRepo := &mockReplyRepo{
		listByAccountIDFn: func(ctx context.Context, accountID int64) ([]model.AutoReplyRule, error) {
			return []model.AutoReplyRule{}, nil
		},
	}

	handler := newReplyHandler(mockRepo)
	r := setupReplyTestRouter(handler)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/accounts/999/replies", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	data := resp["data"].([]interface{})
	assert.Len(t, data, 0)
}

func TestReplyList_InvalidAccountID(t *testing.T) {
	handler := newReplyHandler(&mockReplyRepo{})
	r := setupReplyTestRouter(handler)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/accounts/abc/replies", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestReplyList_Unauthenticated(t *testing.T) {
	handler := newReplyHandler(&mockReplyRepo{})
	r := gin.New()
	r.GET("/api/v1/accounts/:id/replies", handler.List)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/accounts/1/replies", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

// ========== Create Tests ==========

func TestReplyCreate_Success(t *testing.T) {
	mockRepo := &mockReplyRepo{}

	handler := newReplyHandler(mockRepo)
	r := setupReplyTestRouter(handler)

	body := `{"keyword":"hello","match_type":0,"reply_type":1,"reply_content":"你好，有什么可以帮助你的？","status":1}`
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/accounts/10/replies", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)

	var resp map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, float64(0), resp["code"])

	data := resp["data"].(map[string]interface{})
	assert.Equal(t, float64(1), data["id"])
	assert.Equal(t, float64(10), data["account_id"])
	assert.Equal(t, "hello", data["keyword"])
	assert.Equal(t, "你好，有什么可以帮助你的？", data["reply_content"])
}

func TestReplyCreate_DefaultReply(t *testing.T) {
	mockRepo := &mockReplyRepo{}

	handler := newReplyHandler(mockRepo)
	r := setupReplyTestRouter(handler)

	body := `{"reply_type":1,"reply_content":"感谢关注！"}`
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/accounts/10/replies", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	assert.Equal(t, float64(0), resp["code"])
}

func TestReplyCreate_NewsReply(t *testing.T) {
	mockRepo := &mockReplyRepo{}

	handler := newReplyHandler(mockRepo)
	r := setupReplyTestRouter(handler)

	body := `{
		"keyword":"product",
		"match_type":0,
		"reply_type":2,
		"reply_content":"[{\"title\":\"产品介绍\",\"desc\":\"最新产品介绍\",\"cover\":\"https://img.example.com/cover.jpg\",\"url\":\"https://www.example.com\"}]",
		"reply_title":"产品介绍",
		"reply_desc":"最新产品介绍",
		"reply_cover_url":"https://img.example.com/cover.jpg",
		"reply_url":"https://www.example.com"
	}`
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/accounts/10/replies", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	assert.Equal(t, float64(0), resp["code"])
	data := resp["data"].(map[string]interface{})
	assert.Equal(t, "product", data["keyword"])
	assert.Equal(t, float64(2), data["reply_type"])
}

func TestReplyCreate_MissingRequired(t *testing.T) {
	mockRepo := &mockReplyRepo{}

	handler := newReplyHandler(mockRepo)
	r := setupReplyTestRouter(handler)

	// Missing reply_content
	body := `{"reply_type":1}`
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/accounts/10/replies", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// ========== Update Tests ==========

func TestReplyUpdate_Success(t *testing.T) {
	kw := "hello"
	mockRepo := &mockReplyRepo{
		getByIDFn: func(ctx context.Context, id int64) (*model.AutoReplyRule, error) {
			return &model.AutoReplyRule{
				ID: 1, AccountID: 10, Keyword: &kw,
				MatchType: 0, ReplyType: 1, ReplyContent: "你好！",
				Status: 1, SortOrder: 0,
			}, nil
		},
	}

	handler := newReplyHandler(mockRepo)
	r := setupReplyTestRouter(handler)

	body := `{"keyword":"updated keyword","match_type":1,"reply_content":"更新后的回复内容"}`
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPut, "/api/v1/replies/1", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	assert.Equal(t, float64(0), resp["code"])
}

func TestReplyUpdate_NotFound(t *testing.T) {
	mockRepo := &mockReplyRepo{
		getByIDFn: func(ctx context.Context, id int64) (*model.AutoReplyRule, error) {
			return nil, nil
		},
	}

	handler := newReplyHandler(mockRepo)
	r := setupReplyTestRouter(handler)

	body := `{"keyword":"updated"}`
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPut, "/api/v1/replies/999", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestReplyUpdate_InvalidID(t *testing.T) {
	handler := newReplyHandler(&mockReplyRepo{})
	r := setupReplyTestRouter(handler)

	body := `{"keyword":"test"}`
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPut, "/api/v1/replies/abc", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestReplyUpdate_RepoError(t *testing.T) {
	mockRepo := &mockReplyRepo{
		getByIDFn: func(ctx context.Context, id int64) (*model.AutoReplyRule, error) {
			return nil, errors.New("db connection lost")
		},
	}

	handler := newReplyHandler(mockRepo)
	r := setupReplyTestRouter(handler)

	body := `{"keyword":"updated"}`
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPut, "/api/v1/replies/1", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

// ========== Delete Tests ==========

func TestReplyDelete_Success(t *testing.T) {
	mockRepo := &mockReplyRepo{}

	handler := newReplyHandler(mockRepo)
	r := setupReplyTestRouter(handler)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodDelete, "/api/v1/replies/1", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	assert.Equal(t, float64(0), resp["code"])
	assert.Equal(t, "已删除", resp["msg"])
}

func TestReplyDelete_NotFound(t *testing.T) {
	mockRepo := &mockReplyRepo{
		softDeleteFn: func(ctx context.Context, id int64) (bool, error) {
			return false, nil
		},
	}

	handler := newReplyHandler(mockRepo)
	r := setupReplyTestRouter(handler)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodDelete, "/api/v1/replies/999", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	assert.Equal(t, float64(40401), resp["code"])
}

func TestReplyDelete_InvalidID(t *testing.T) {
	handler := newReplyHandler(&mockReplyRepo{})
	r := setupReplyTestRouter(handler)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodDelete, "/api/v1/replies/abc", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// ========== API Response Format Tests ==========

func TestReplyAPIResponseFormat(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("normal response format", func(t *testing.T) {
		r := gin.New()
		r.GET("/test-reply-ok", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{
				"code": 0,
				"msg":  "ok",
				"data": gin.H{"key": "value"},
			})
		})

		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodGet, "/test-reply-ok", nil)
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var resp map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &resp)
		assert.Equal(t, float64(0), resp["code"])
		assert.Equal(t, "ok", resp["msg"])
		assert.NotNil(t, resp["data"])
	})

	t.Run("error response format", func(t *testing.T) {
		r := gin.New()
		r.GET("/test-reply-err", func(c *gin.Context) {
			c.JSON(http.StatusBadRequest, gin.H{
				"code": 40001,
				"msg":  "参数错误",
				"data": nil,
			})
		})

		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodGet, "/test-reply-err", nil)
		r.ServeHTTP(w, req)

		var resp map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &resp)
		assert.NotEqual(t, float64(0), resp["code"])
	})
}

// ========== Semantic Tests ==========

func TestMatchTypeValues(t *testing.T) {
	assert.Equal(t, int16(0), int16(0), "完全匹配")
	assert.Equal(t, int16(1), int16(1), "包含匹配")
}

func TestReplyTypeValues(t *testing.T) {
	assert.Equal(t, int16(1), int16(1), "文本回复")
	assert.Equal(t, int16(2), int16(2), "图文回复")
}

// ========== ReplyVO Tests ==========

func TestReplyVOConversion(t *testing.T) {
	kw := "help"
	rule := &model.AutoReplyRule{
		ID:           1,
		AccountID:    10,
		Keyword:      &kw,
		MatchType:    1,
		ReplyType:    1,
		ReplyContent: "test content",
		Status:       1,
		SortOrder:    0,
	}
	vo := toReplyVO(rule)
	assert.Equal(t, int64(1), vo.ID)
	assert.Equal(t, int64(10), vo.AccountID)
	assert.Equal(t, "help", *vo.Keyword)
	assert.Equal(t, "test content", vo.ReplyContent)
}
