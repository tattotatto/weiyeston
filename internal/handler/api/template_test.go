// Package api T10: 模板系统 Handler 测试
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

	"github.com/weiyeston/weiyeston-v2/internal/model"
)

// ========== Helpers ==========

func setupTemplateTestRouter(handler *TemplateHandler) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("tenant_id", int64(1))
		c.Set("user_id", int64(1))
		c.Next()
	})
	r.GET("/api/v1/templates", handler.ListSystemTemplates)
	r.POST("/api/v1/templates", handler.SaveTemplate)
	return r
}

// ========== ListSystemTemplates Tests ==========

func TestListSystemTemplates_Success(t *testing.T) {
	t.Run("returns template list", func(t *testing.T) {
		handler := &TemplateHandler{
			selectFunc: func(ctx context.Context, dest interface{}, query string, args ...interface{}) error {
				articles := dest.(*[]model.Article)
				title1 := "公司介绍模板"
				cover1 := "/uploads/cover1.jpg"
				summary1 := "企业介绍页面模板"
				author1 := "admin"
				cat1 := "企业模板"
				*articles = append(*articles, model.Article{
					ID: 1, AccountID: 1, Title: &title1, CoverURL: &cover1,
					Summary: &summary1, Author: &author1, TemplateCat: &cat1,
					Content: json.RawMessage(`{"sections":[]}`), Status: 1,
					IsTemplate: true, CreatedAt: time.Now(), UpdatedAt: time.Now(),
				})
				title2 := "产品展示模板"
				cover2 := "/uploads/cover2.jpg"
				summary2 := "产品展示页面模板"
				author2 := "admin"
				cat2 := "产品模板"
				*articles = append(*articles, model.Article{
					ID: 2, AccountID: 1, Title: &title2, CoverURL: &cover2,
					Summary: &summary2, Author: &author2, TemplateCat: &cat2,
					Content: json.RawMessage(`{"sections":[{"type":"gallery"}]}`), Status: 1,
					IsTemplate: true, CreatedAt: time.Now(), UpdatedAt: time.Now(),
				})
				return nil
			},
		}

		r := setupTemplateTestRouter(handler)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodGet, "/api/v1/templates", nil)
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
		assert.Equal(t, "公司介绍模板", first["title"])
		assert.Equal(t, "企业模板", first["template_cat"])
	})
}

func TestListSystemTemplates_WithCategoryFilter(t *testing.T) {
	t.Run("filters by category", func(t *testing.T) {
		handler := &TemplateHandler{
			selectFunc: func(ctx context.Context, dest interface{}, query string, args ...interface{}) error {
				assert.Contains(t, query, "template_cat = $1")
				assert.Len(t, args, 1)
				assert.Equal(t, "产品模板", args[0])

				articles := dest.(*[]model.Article)
				title := "产品模板"
				cat := "产品模板"
				*articles = append(*articles, model.Article{
					ID: 1, AccountID: 1, Title: &title, TemplateCat: &cat,
					Content: json.RawMessage(`{}`), Status: 1, IsTemplate: true,
					CreatedAt: time.Now(), UpdatedAt: time.Now(),
				})
				return nil
			},
		}

		r := setupTemplateTestRouter(handler)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodGet, "/api/v1/templates?category=产品模板", nil)
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var resp map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &resp)
		assert.Equal(t, float64(0), resp["code"])

		data := resp["data"].([]interface{})
		assert.Len(t, data, 1)
	})
}

func TestListSystemTemplates_EmptyList(t *testing.T) {
	t.Run("returns empty list", func(t *testing.T) {
		handler := &TemplateHandler{
			selectFunc: func(ctx context.Context, dest interface{}, query string, args ...interface{}) error {
				return nil
			},
		}

		r := setupTemplateTestRouter(handler)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodGet, "/api/v1/templates", nil)
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var resp map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &resp)
		assert.Equal(t, float64(0), resp["code"])
		data := resp["data"].([]interface{})
		assert.Empty(t, data)
	})
}

func TestListSystemTemplates_DBError(t *testing.T) {
	t.Run("returns 500 on db error", func(t *testing.T) {
		handler := &TemplateHandler{
			selectFunc: func(ctx context.Context, dest interface{}, query string, args ...interface{}) error {
				return errors.New("database connection error")
			},
		}

		r := setupTemplateTestRouter(handler)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodGet, "/api/v1/templates", nil)
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusInternalServerError, w.Code)

		var resp map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &resp)
		assert.Equal(t, float64(50001), resp["code"])
	})
}

// ========== SaveTemplate Tests ==========

func TestSaveTemplate_Success(t *testing.T) {
	t.Run("saves template successfully", func(t *testing.T) {
		now := time.Now()
		handler := &TemplateHandler{
			saveFunc: func(ctx context.Context, query string, args ...interface{}) (int64, time.Time, time.Time, error) {
				assert.Contains(t, query, "INSERT INTO cms_articles")
				return 10, now, now, nil
			},
		}

		r := setupTemplateTestRouter(handler)

		body := `{
			"title": "我的模板",
			"cover_url": "/cover.jpg",
			"summary": "模板摘要",
			"content": {"sections":[{"type":"text","content":"正文内容"}]},
			"template_cat": "自定义",
			"author": "作者名"
		}`

		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodPost, "/api/v1/templates", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusCreated, w.Code)

		var resp map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.Equal(t, float64(0), resp["code"])
		assert.Equal(t, "模板保存成功", resp["msg"])

		data := resp["data"].(map[string]interface{})
		assert.Equal(t, float64(10), data["id"])
		assert.Equal(t, "我的模板", data["title"])
		assert.Equal(t, "自定义", data["template_cat"])
	})
}

func TestSaveTemplate_MissingTitle(t *testing.T) {
	t.Run("returns 400 when title missing", func(t *testing.T) {
		handler := &TemplateHandler{}
		r := setupTemplateTestRouter(handler)

		body := `{"content":{}}`
		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodPost, "/api/v1/templates", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)

		var resp map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &resp)
		assert.Equal(t, float64(40001), resp["code"])
	})
}

func TestSaveTemplate_MissingContent(t *testing.T) {
	t.Run("returns 400 when content missing", func(t *testing.T) {
		handler := &TemplateHandler{}
		r := setupTemplateTestRouter(handler)

		body := `{"title":"测试"}`
		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodPost, "/api/v1/templates", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)

		var resp map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &resp)
		assert.Equal(t, float64(40001), resp["code"])
	})
}

func TestSaveTemplate_InvalidJSON(t *testing.T) {
	t.Run("returns 400 on invalid JSON", func(t *testing.T) {
		handler := &TemplateHandler{}
		r := setupTemplateTestRouter(handler)

		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodPost, "/api/v1/templates", strings.NewReader("not json"))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestSaveTemplate_DBError(t *testing.T) {
	t.Run("returns 500 on db error", func(t *testing.T) {
		handler := &TemplateHandler{
			saveFunc: func(ctx context.Context, query string, args ...interface{}) (int64, time.Time, time.Time, error) {
				return 0, time.Time{}, time.Time{}, errors.New("insert failed")
			},
		}

		r := setupTemplateTestRouter(handler)
		body := `{"title":"模板","content":{}}`
		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodPost, "/api/v1/templates", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusInternalServerError, w.Code)

		var resp map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &resp)
		assert.Equal(t, float64(50001), resp["code"])
	})
}

// ========== TemplateVO Conversion Tests ==========

func TestToTemplateVO_NilFields(t *testing.T) {
	a := &model.Article{
		ID:        1,
		AccountID: 1,
		Status:    1,
		Content:   json.RawMessage(`{}`),
	}

	vo := toTemplateVO(a)
	assert.Equal(t, int64(1), vo.ID)
	assert.Empty(t, vo.Title)
	assert.Empty(t, vo.CoverURL)
	assert.Empty(t, vo.Summary)
	assert.Empty(t, vo.TemplateCat)
	assert.Empty(t, vo.Author)
	assert.Equal(t, json.RawMessage(`{}`), vo.Content)
}

func TestToTemplateVO_AllFields(t *testing.T) {
	title := "完整模板"
	coverURL := "/cover.jpg"
	summary := "这是描述"
	author := "编辑"
	templateCat := "企业模板"
	content := json.RawMessage(`{"sections":[]}`)

	a := &model.Article{
		ID:          5,
		Title:       &title,
		CoverURL:    &coverURL,
		Summary:     &summary,
		Author:      &author,
		TemplateCat: &templateCat,
		Content:     content,
	}

	vo := toTemplateVO(a)
	assert.Equal(t, int64(5), vo.ID)
	assert.Equal(t, "完整模板", vo.Title)
	assert.Equal(t, "/cover.jpg", vo.CoverURL)
	assert.Equal(t, "这是描述", vo.Summary)
	assert.Equal(t, "编辑", vo.Author)
	assert.Equal(t, "企业模板", vo.TemplateCat)
	assert.Equal(t, content, vo.Content)
}

func TestToTemplateVOs_EmptySlice(t *testing.T) {
	vos := toTemplateVOs([]model.Article{})
	assert.Empty(t, vos)
}

func TestToTemplateVOs_NilSlice(t *testing.T) {
	vos := toTemplateVOs(nil)
	assert.Empty(t, vos)
}

// ========== API Response Format Tests ==========

func TestTemplateAPIResponseFormat(t *testing.T) {
	t.Run("response has correct content type", func(t *testing.T) {
		handler := &TemplateHandler{
			selectFunc: func(ctx context.Context, dest interface{}, query string, args ...interface{}) error {
				title := "模板标题"
				cover := "/cover.jpg"
				summary := "摘要"
				author := "作者"
				cat := "分类"
				articles := dest.(*[]model.Article)
				*articles = append(*articles, model.Article{
					ID: 1, AccountID: 1, Title: &title, CoverURL: &cover,
					Summary: &summary, Author: &author, TemplateCat: &cat,
					Content: json.RawMessage(`{"sections":[]}`), Status: 1,
					IsTemplate: true, CreatedAt: time.Now(), UpdatedAt: time.Now(),
				})
				return nil
			},
		}

		r := setupTemplateTestRouter(handler)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodGet, "/api/v1/templates", nil)
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, "application/json; charset=utf-8", w.Header().Get("Content-Type"))
	})
}
