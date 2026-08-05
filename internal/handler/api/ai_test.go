// Package api T11: AI 集成 Handler 测试
package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	aiservice "github.com/weiyeston/weiyeston-v2/internal/service/ai"
)

// ========== Helpers ==========

func setupAITestRouter(handler *AIHandler) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.POST("/api/v1/ai/write", handler.Write)
	r.POST("/api/v1/ai/layout", handler.Layout)
	r.POST("/api/v1/ai/proofread", handler.Proofread)
	return r
}

func newAIHandlerWithMock(chatFunc func(ctx context.Context, messages []aiservice.Message, opts aiservice.ChatOpts) (*aiservice.ChatResult, error)) *AIHandler {
	mock := &aiservice.MockLLMProvider{ChatFunc: chatFunc}
	svc := aiservice.NewAIService(mock)
	return NewAIHandler(svc)
}

// ========== Write Tests ==========

func TestAIWrite_Success(t *testing.T) {
	handler := newAIHandlerWithMock(func(ctx context.Context, messages []aiservice.Message, opts aiservice.ChatOpts) (*aiservice.ChatResult, error) {
		return &aiservice.ChatResult{
			Content: "# 生成的文章\n\n这是AI生成的内容。",
			Usage: aiservice.Usage{
				PromptTokens:     100,
				CompletionTokens: 200,
				TotalTokens:      300,
			},
		}, nil
	})

	r := setupAITestRouter(handler)

	body := `{"title":"测试文章","keywords":"测试,AI","style":"轻松","max_words":500}`
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/ai/write", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, float64(0), resp["code"])
	assert.Equal(t, "ok", resp["msg"])

	data := resp["data"].(map[string]interface{})
	assert.NotEmpty(t, data["content"])
	assert.Contains(t, data["content"], "生成的文章")

	usage := data["usage"].(map[string]interface{})
	assert.Equal(t, float64(300), usage["total_tokens"])
}

func TestAIWrite_MissingTitle(t *testing.T) {
	handler := newAIHandlerWithMock(nil)
	r := setupAITestRouter(handler)

	body := `{"keywords":"测试"}`
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/ai/write", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	assert.Equal(t, float64(40001), resp["code"])
}

func TestAIWrite_ServiceError(t *testing.T) {
	handler := newAIHandlerWithMock(func(ctx context.Context, messages []aiservice.Message, opts aiservice.ChatOpts) (*aiservice.ChatResult, error) {
		return nil, assert.AnError
	})

	r := setupAITestRouter(handler)

	body := `{"title":"测试"}`
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/ai/write", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	assert.Equal(t, float64(50001), resp["code"])
}

func TestAIWrite_InvalidJSON(t *testing.T) {
	handler := newAIHandlerWithMock(nil)
	r := setupAITestRouter(handler)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/ai/write", strings.NewReader("not json"))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// ========== Layout Tests ==========

func TestAILayout_Success(t *testing.T) {
	handler := newAIHandlerWithMock(func(ctx context.Context, messages []aiservice.Message, opts aiservice.ChatOpts) (*aiservice.ChatResult, error) {
		return &aiservice.ChatResult{
			Content: `{"sections":[{"type":"text","content":"优化后内容"}]}`,
			Usage: aiservice.Usage{
				PromptTokens:     50,
				CompletionTokens: 80,
				TotalTokens:      130,
			},
		}, nil
	})

	r := setupAITestRouter(handler)

	body := `{"content":{"sections":[{"type":"text","content":"原文"}]}}`
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/ai/layout", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, float64(0), resp["code"])

	data := resp["data"].(map[string]interface{})
	assert.NotNil(t, data["content"])
}

func TestAILayout_MissingContent(t *testing.T) {
	handler := newAIHandlerWithMock(nil)
	r := setupAITestRouter(handler)

	body := `{}`
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/ai/layout", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	assert.Equal(t, float64(40001), resp["code"])
}

func TestAILayout_ServiceError(t *testing.T) {
	handler := newAIHandlerWithMock(func(ctx context.Context, messages []aiservice.Message, opts aiservice.ChatOpts) (*aiservice.ChatResult, error) {
		return nil, assert.AnError
	})

	r := setupAITestRouter(handler)

	body := `{"content":{}}`
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/ai/layout", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	assert.Equal(t, float64(50001), resp["code"])
}

// ========== Proofread Tests ==========

func TestAIProofread_Success(t *testing.T) {
	handler := newAIHandlerWithMock(func(ctx context.Context, messages []aiservice.Message, opts aiservice.ChatOpts) (*aiservice.ChatResult, error) {
		return &aiservice.ChatResult{
			Content: `{"corrections":[{"position":0,"length":3,"type":"typo","original":"措别字","suggestion":"错别字","explanation":"汉字错误"}]}`,
			Usage: aiservice.Usage{
				PromptTokens:     30,
				CompletionTokens: 40,
				TotalTokens:      70,
			},
		}, nil
	})

	r := setupAITestRouter(handler)

	body := `{"text":"这是一段有措别字的文字"}`
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/ai/proofread", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, float64(0), resp["code"])

	data := resp["data"].(map[string]interface{})
	corrections := data["corrections"].([]interface{})
	assert.Len(t, corrections, 1)

	first := corrections[0].(map[string]interface{})
	assert.Equal(t, float64(0), first["position"])
	assert.Equal(t, "typo", first["type"])
}

func TestAIProofread_NoCorrections(t *testing.T) {
	handler := newAIHandlerWithMock(func(ctx context.Context, messages []aiservice.Message, opts aiservice.ChatOpts) (*aiservice.ChatResult, error) {
		return &aiservice.ChatResult{
			Content: `{"corrections":[]}`,
			Usage:   aiservice.Usage{TotalTokens: 20},
		}, nil
	})

	r := setupAITestRouter(handler)

	body := `{"text":"完全正确的文字"}`
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/ai/proofread", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	assert.Equal(t, float64(0), resp["code"])

	data := resp["data"].(map[string]interface{})
	corrections := data["corrections"].([]interface{})
	assert.Empty(t, corrections)
}

func TestAIProofread_MissingText(t *testing.T) {
	handler := newAIHandlerWithMock(nil)
	r := setupAITestRouter(handler)

	body := `{}`
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/ai/proofread", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	assert.Equal(t, float64(40001), resp["code"])
}

func TestAIProofread_ServiceError(t *testing.T) {
	handler := newAIHandlerWithMock(func(ctx context.Context, messages []aiservice.Message, opts aiservice.ChatOpts) (*aiservice.ChatResult, error) {
		return nil, assert.AnError
	})

	r := setupAITestRouter(handler)

	body := `{"text":"test"}`
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/ai/proofread", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	assert.Equal(t, float64(50001), resp["code"])
}

// ========== API Response Format Tests ==========

func TestAIResponseFormat(t *testing.T) {
	handler := newAIHandlerWithMock(func(ctx context.Context, messages []aiservice.Message, opts aiservice.ChatOpts) (*aiservice.ChatResult, error) {
		return &aiservice.ChatResult{
			Content: "# Test",
			Usage:   aiservice.Usage{TotalTokens: 10},
		}, nil
	})

	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.POST("/api/v1/ai/write", handler.Write)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/ai/write", strings.NewReader(`{"title":"test"}`))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "application/json; charset=utf-8", w.Header().Get("Content-Type"))

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	assert.Equal(t, float64(0), resp["code"])
	assert.Equal(t, "ok", resp["msg"])
	assert.NotNil(t, resp["data"])
}

// ========== NewAIHandler Tests ==========

func TestNewAIHandler_CreatesHandler(t *testing.T) {
	mock := &aiservice.MockLLMProvider{}
	svc := aiservice.NewAIService(mock)
	handler := NewAIHandler(svc)
	assert.NotNil(t, handler)
	assert.NotNil(t, handler.service)
}
