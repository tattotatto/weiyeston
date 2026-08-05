// Package api T6: 微信自定义菜单管理 Handler 测试
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

// ========== Mock MenuRepo ==========

type mockMenuRepo struct {
	getByAccountIDFn func(ctx context.Context, accountID int64) (*model.WechatMenu, error)
	createFn         func(ctx context.Context, m *model.WechatMenu) error
	updateFn         func(ctx context.Context, m *model.WechatMenu) error
	softDeleteFn     func(ctx context.Context, id int64) (bool, error)
}

func (m *mockMenuRepo) GetByAccountID(ctx context.Context, accountID int64) (*model.WechatMenu, error) {
	if m.getByAccountIDFn != nil {
		return m.getByAccountIDFn(ctx, accountID)
	}
	return nil, nil
}

func (m *mockMenuRepo) Create(ctx context.Context, menu *model.WechatMenu) error {
	if m.createFn != nil {
		return m.createFn(ctx, menu)
	}
	menu.ID = 1
	menu.CreatedAt = time.Now()
	menu.UpdatedAt = time.Now()
	return nil
}

func (m *mockMenuRepo) Update(ctx context.Context, menu *model.WechatMenu) error {
	if m.updateFn != nil {
		return m.updateFn(ctx, menu)
	}
	return nil
}

func (m *mockMenuRepo) SoftDelete(ctx context.Context, id int64) (bool, error) {
	if m.softDeleteFn != nil {
		return m.softDeleteFn(ctx, id)
	}
	return true, nil
}

// ========== Helpers ==========

func setupMenuTestRouter(handler *MenuHandler) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("tenant_id", int64(1))
		c.Set("user_id", int64(1))
		c.Next()
	})
	r.GET("/api/v1/accounts/:id/menu", handler.GetMenu)
	r.POST("/api/v1/accounts/:id/menu", handler.SaveDraft)
	r.PUT("/api/v1/accounts/:id/menu/publish", handler.Publish)
	r.DELETE("/api/v1/accounts/:id/menu", handler.DeleteDraft)
	return r
}

func newMenuHandler(repo MenuRepo) *MenuHandler {
	return NewMenuHandler(repo, zap.NewNop())
}

func sampleMenuJSON() json.RawMessage {
	raw := json.RawMessage(`{
		"button": [
			{"type": "click", "name": "关于我们", "key": "ABOUT"},
			{"type": "view", "name": "官网", "url": "https://www.example.com"},
			{
				"name": "更多",
				"sub_button": [
					{"type": "click", "name": "联系客服", "key": "CONTACT"},
					{"type": "view", "name": "帮助中心", "url": "https://help.example.com"}
				]
			}
		]
	}`)
	return raw
}

// ========== GetMenu Tests ==========

func TestGetMenu_Success(t *testing.T) {
	menuJSON := sampleMenuJSON()
	mockRepo := &mockMenuRepo{
		getByAccountIDFn: func(ctx context.Context, accountID int64) (*model.WechatMenu, error) {
			return &model.WechatMenu{
				ID:        1,
				AccountID: accountID,
				MenuJSON:  &menuJSON,
				Status:    model.MenuStatusDraft,
				UpdatedAt: time.Now(),
			}, nil
		},
	}

	handler := newMenuHandler(mockRepo)
	r := setupMenuTestRouter(handler)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/accounts/10/menu", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, float64(0), resp["code"])

	data := resp["data"].(map[string]interface{})
	assert.Equal(t, float64(1), data["id"])
	assert.Equal(t, float64(10), data["account_id"])
	assert.Equal(t, "草稿", data["status_text"])
}

func TestGetMenu_NoMenu(t *testing.T) {
	mockRepo := &mockMenuRepo{
		getByAccountIDFn: func(ctx context.Context, accountID int64) (*model.WechatMenu, error) {
			return nil, nil // 没有菜单
		},
	}

	handler := newMenuHandler(mockRepo)
	r := setupMenuTestRouter(handler)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/accounts/999/menu", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	assert.Equal(t, float64(0), resp["code"])
	assert.Nil(t, resp["data"])
}

func TestGetMenu_InvalidAccountID(t *testing.T) {
	handler := newMenuHandler(&mockMenuRepo{})
	r := setupMenuTestRouter(handler)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/accounts/abc/menu", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestGetMenu_Unauthenticated(t *testing.T) {
	handler := newMenuHandler(&mockMenuRepo{})
	r := gin.New()
	r.GET("/api/v1/accounts/:id/menu", handler.GetMenu)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/accounts/1/menu", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

// ========== SaveDraft Tests ==========

func TestSaveDraft_CreateNew(t *testing.T) {
	mockRepo := &mockMenuRepo{
		getByAccountIDFn: func(ctx context.Context, accountID int64) (*model.WechatMenu, error) {
			return nil, nil // 没有已有菜单
		},
		createFn: func(ctx context.Context, m *model.WechatMenu) error {
			m.ID = 1
			m.CreatedAt = time.Now()
			m.UpdatedAt = time.Now()
			return nil
		},
	}

	handler := newMenuHandler(mockRepo)
	r := setupMenuTestRouter(handler)

	body := `{"menu_json": {"button": [{"type": "click", "name": "测试", "key": "TEST"}]}}`
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/accounts/10/menu", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)

	var resp map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, float64(0), resp["code"])
	assert.Equal(t, "草稿已保存", resp["msg"])

	data := resp["data"].(map[string]interface{})
	assert.Equal(t, float64(0), data["status"]) // 草稿状态
}

func TestSaveDraft_UpdateExisting(t *testing.T) {
	existingJSON := json.RawMessage(`{"button":[]}`)
	existing := &model.WechatMenu{
		ID:        5,
		AccountID: 10,
		MenuJSON:  &existingJSON,
		Status:    model.MenuStatusDraft,
	}

	mockRepo := &mockMenuRepo{
		getByAccountIDFn: func(ctx context.Context, accountID int64) (*model.WechatMenu, error) {
			return existing, nil
		},
	}

	handler := newMenuHandler(mockRepo)
	r := setupMenuTestRouter(handler)

	body := `{"menu_json": {"button": [{"type": "view", "name": "新按钮", "url": "https://example.com"}]}}`
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/accounts/10/menu", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	assert.Equal(t, float64(0), resp["code"])
	assert.Equal(t, "草稿已更新", resp["msg"])
}

func TestSaveDraft_InvalidJSON(t *testing.T) {
	handler := newMenuHandler(&mockMenuRepo{
		getByAccountIDFn: func(ctx context.Context, accountID int64) (*model.WechatMenu, error) {
			return nil, nil
		},
	})
	r := setupMenuTestRouter(handler)

	body := `invalid json`
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/accounts/10/menu", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// ========== Publish Tests ==========

func TestPublish_Success(t *testing.T) {
	menuJSON := sampleMenuJSON()
	mockRepo := &mockMenuRepo{
		getByAccountIDFn: func(ctx context.Context, accountID int64) (*model.WechatMenu, error) {
			return &model.WechatMenu{
				ID:        1,
				AccountID: accountID,
				MenuJSON:  &menuJSON,
				Status:    model.MenuStatusDraft,
			}, nil
		},
	}

	handler := newMenuHandler(mockRepo)
	r := setupMenuTestRouter(handler)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPut, "/api/v1/accounts/10/menu/publish", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, float64(0), resp["code"])
	assert.Equal(t, "菜单已发布", resp["msg"])

	data := resp["data"].(map[string]interface{})
	assert.Equal(t, float64(1), data["status"]) // 已发布
	assert.Equal(t, "已发布", data["status_text"])
	assert.NotNil(t, data["published_at"])
}

func TestPublish_NoDraft(t *testing.T) {
	mockRepo := &mockMenuRepo{
		getByAccountIDFn: func(ctx context.Context, accountID int64) (*model.WechatMenu, error) {
			return nil, nil
		},
	}

	handler := newMenuHandler(mockRepo)
	r := setupMenuTestRouter(handler)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPut, "/api/v1/accounts/10/menu/publish", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	assert.Contains(t, resp["msg"], "请先保存菜单草稿")
}

func TestPublish_InvalidAccountID(t *testing.T) {
	handler := newMenuHandler(&mockMenuRepo{})
	r := setupMenuTestRouter(handler)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPut, "/api/v1/accounts/abc/menu/publish", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// ========== DeleteDraft Tests ==========

func TestDeleteDraft_Success(t *testing.T) {
	menuJSON := sampleMenuJSON()
	mockRepo := &mockMenuRepo{
		getByAccountIDFn: func(ctx context.Context, accountID int64) (*model.WechatMenu, error) {
			return &model.WechatMenu{
				ID:        1,
				AccountID: accountID,
				MenuJSON:  &menuJSON,
				Status:    model.MenuStatusDraft,
			}, nil
		},
	}

	handler := newMenuHandler(mockRepo)
	r := setupMenuTestRouter(handler)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodDelete, "/api/v1/accounts/10/menu", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	assert.Equal(t, float64(0), resp["code"])
	assert.Equal(t, "已删除", resp["msg"])
}

func TestDeleteDraft_NotExists(t *testing.T) {
	mockRepo := &mockMenuRepo{
		getByAccountIDFn: func(ctx context.Context, accountID int64) (*model.WechatMenu, error) {
			return nil, nil
		},
	}

	handler := newMenuHandler(mockRepo)
	r := setupMenuTestRouter(handler)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodDelete, "/api/v1/accounts/999/menu", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	assert.Equal(t, float64(40401), resp["code"])
}

func TestDeleteDraft_PublishedCannotDelete(t *testing.T) {
	menuJSON := sampleMenuJSON()
	mockRepo := &mockMenuRepo{
		getByAccountIDFn: func(ctx context.Context, accountID int64) (*model.WechatMenu, error) {
			now := time.Now()
			return &model.WechatMenu{
				ID:          1,
				AccountID:   accountID,
				MenuJSON:    &menuJSON,
				Status:      model.MenuStatusPublished,
				PublishedAt: &now,
			}, nil
		},
	}

	handler := newMenuHandler(mockRepo)
	r := setupMenuTestRouter(handler)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodDelete, "/api/v1/accounts/10/menu", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	assert.Contains(t, resp["msg"], "不能直接删除")
}

// ========== API Response Format Tests ==========

func TestMenuAPIResponseFormat(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("normal response", func(t *testing.T) {
		menuJSON := sampleMenuJSON()
		mockRepo := &mockMenuRepo{
			getByAccountIDFn: func(ctx context.Context, accountID int64) (*model.WechatMenu, error) {
				return &model.WechatMenu{
					ID:        1,
					AccountID: accountID,
					MenuJSON:  &menuJSON,
					Status:    model.MenuStatusDraft,
					UpdatedAt: time.Now(),
				}, nil
			},
		}
		handler := newMenuHandler(mockRepo)
		r := setupMenuTestRouter(handler)

		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodGet, "/api/v1/accounts/1/menu", nil)
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, "application/json; charset=utf-8", w.Header().Get("Content-Type"))
	})
}

// ========== Menu Status Constants Tests ==========

func TestMenuStatusConstants(t *testing.T) {
	assert.Equal(t, int16(0), model.MenuStatusDraft, "草稿状态值应为 0")
	assert.Equal(t, int16(1), model.MenuStatusPublished, "已发布状态值应为 1")
}

// ========== Menu JSON Validation Tests ==========

func TestMenuJSONStructure(t *testing.T) {
	// Verify we can unmarshal the sample menu JSON
	menuJSON := sampleMenuJSON()

	var menu map[string]interface{}
	err := json.Unmarshal(menuJSON, &menu)
	require.NoError(t, err, "sample menu JSON should be valid JSON")

	buttons, ok := menu["button"].([]interface{})
	require.True(t, ok, "menu should have button array")
	assert.Len(t, buttons, 3, "sample menu should have 3 top-level buttons")

	// Verify sub_button structure
	thirdButton := buttons[2].(map[string]interface{})
	assert.Equal(t, "更多", thirdButton["name"])
	subButtons := thirdButton["sub_button"].([]interface{})
	assert.Len(t, subButtons, 2)
}

// ========== MenuRepoImpl Error Handling Tests ==========

func TestMenuRepo_GetByAccountID_InvalidID(t *testing.T) {
	repo := NewMenuRepo(nil)
	menu, err := repo.GetByAccountID(context.Background(), 0)
	assert.Error(t, err)
	assert.Nil(t, menu)
	assert.Contains(t, err.Error(), "无效的公众号 ID")
}

// ========== MenuRepo Interface Satisfaction Tests ==========

func TestMenuRepoInterface(t *testing.T) {
	// Verify menuRepoImpl satisfies MenuRepo interface
	var _ MenuRepo = (*menuRepoImpl)(nil)
	assert.True(t, true, "menuRepoImpl implements MenuRepo")
}

// ========== Edge Cases ==========

func TestSaveDraft_RepoError(t *testing.T) {
	mockRepo := &mockMenuRepo{
		getByAccountIDFn: func(ctx context.Context, accountID int64) (*model.WechatMenu, error) {
			return nil, errors.New("database connection lost")
		},
	}

	handler := newMenuHandler(mockRepo)
	r := setupMenuTestRouter(handler)

	body := `{"menu_json": {"button": []}}`
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/accounts/10/menu", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	assert.Equal(t, float64(50001), resp["code"])
}

func TestGetMenu_RepoError(t *testing.T) {
	mockRepo := &mockMenuRepo{
		getByAccountIDFn: func(ctx context.Context, accountID int64) (*model.WechatMenu, error) {
			return nil, errors.New("connection refused")
		},
	}

	handler := newMenuHandler(mockRepo)
	r := setupMenuTestRouter(handler)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/accounts/10/menu", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	assert.Equal(t, float64(50001), resp["code"])
}
