// Package ai AI 服务测试
package ai

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ========== Write Tests ==========

func TestWrite_Success(t *testing.T) {
	mockContent := `# 测试文章标题

## 引言
这是由AI生成的一篇测试文章，用于验证Write功能。

## 正文
文章内容丰富多彩，结构清晰合理。

## 结尾
感谢阅读，欢迎在评论区留言互动！`

	mock := &MockLLMProvider{
		ChatFunc: func(ctx context.Context, messages []Message, opts ChatOpts) (*ChatResult, error) {
			// Verify system prompt contains the role setting
			assert.Equal(t, "system", messages[0].Role)
			assert.Contains(t, messages[0].Content, "微信公众号小编")
			assert.Equal(t, "user", messages[1].Role)
			assert.Contains(t, messages[1].Content, "测试文章标题")
			assert.Equal(t, 0.8, opts.Temperature)
			assert.Equal(t, 4096, opts.MaxTokens)
			return &ChatResult{
				Content: mockContent,
				Usage: Usage{
					PromptTokens:     150,
					CompletionTokens: 200,
					TotalTokens:      350,
				},
			}, nil
		},
	}

	svc := NewAIService(mock)
	req := WriteRequest{
		Title:    "测试文章标题",
		Keywords: "测试,AI,验证",
		Style:    "轻松",
		MaxWords: 800,
	}

	resp, err := svc.Write(context.Background(), req)
	require.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, mockContent, resp.Content)
	assert.Equal(t, 150, resp.Usage.PromptTokens)
	assert.Equal(t, 200, resp.Usage.CompletionTokens)
	assert.Equal(t, 350, resp.Usage.TotalTokens)
}

func TestWrite_DefaultStyle(t *testing.T) {
	mock := &MockLLMProvider{
		ChatFunc: func(ctx context.Context, messages []Message, opts ChatOpts) (*ChatResult, error) {
			// When no style is specified, should use "正式专业"
			assert.Contains(t, messages[1].Content, "正式专业")
			return &ChatResult{
				Content: "content",
				Usage:   Usage{TotalTokens: 100},
			}, nil
		},
	}

	svc := NewAIService(mock)
	resp, err := svc.Write(context.Background(), WriteRequest{
		Title: "测试",
	})
	require.NoError(t, err)
	assert.NotNil(t, resp)
}

func TestWrite_NoKeywords(t *testing.T) {
	mock := &MockLLMProvider{
		ChatFunc: func(ctx context.Context, messages []Message, opts ChatOpts) (*ChatResult, error) {
			// When no keywords, should not include keywords section in prompt
			assert.NotContains(t, messages[1].Content, "关键词")
			return &ChatResult{
				Content: "content",
				Usage:   Usage{TotalTokens: 100},
			}, nil
		},
	}

	svc := NewAIService(mock)
	resp, err := svc.Write(context.Background(), WriteRequest{
		Title: "测试",
		Style: "活泼",
	})
	require.NoError(t, err)
	assert.NotNil(t, resp)
}

func TestWrite_NoMaxWords(t *testing.T) {
	mock := &MockLLMProvider{
		ChatFunc: func(ctx context.Context, messages []Message, opts ChatOpts) (*ChatResult, error) {
			// When no max_words, should not include 字数限制 in prompt
			assert.NotContains(t, messages[1].Content, "字数控制")
			return &ChatResult{
				Content: "content",
				Usage:   Usage{TotalTokens: 100},
			}, nil
		},
	}

	svc := NewAIService(mock)
	resp, err := svc.Write(context.Background(), WriteRequest{
		Title: "测试",
	})
	require.NoError(t, err)
	assert.NotNil(t, resp)
}

func TestWrite_ProviderError(t *testing.T) {
	mock := &MockLLMProvider{
		ChatFunc: func(ctx context.Context, messages []Message, opts ChatOpts) (*ChatResult, error) {
			return nil, assert.AnError
		},
	}

	svc := NewAIService(mock)
	resp, err := svc.Write(context.Background(), WriteRequest{
		Title: "测试",
	})
	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.Contains(t, err.Error(), "AI帮写失败")
}

// ========== Layout Tests ==========

func TestLayout_Success(t *testing.T) {
	inputContent := json.RawMessage(`{"sections":[{"type":"text","content":"段落1"},{"type":"image","src":"url"}]}`)
	expectedOutput := json.RawMessage(`{"sections":[{"type":"image","src":"url"},{"type":"text","content":"段落1"}]}`)

	mock := &MockLLMProvider{
		ChatFunc: func(ctx context.Context, messages []Message, opts ChatOpts) (*ChatResult, error) {
			assert.Equal(t, "system", messages[0].Role)
			assert.Contains(t, messages[0].Content, "排版")
			assert.Equal(t, "user", messages[1].Role)
			assert.Contains(t, messages[1].Content, string(inputContent))
			assert.Equal(t, 0.5, opts.Temperature)
			return &ChatResult{
				Content: string(expectedOutput),
				Usage: Usage{
					PromptTokens:     100,
					CompletionTokens: 150,
					TotalTokens:      250,
				},
			}, nil
		},
	}

	svc := NewAIService(mock)
	resp, err := svc.Layout(context.Background(), LayoutRequest{
		Content: inputContent,
	})
	require.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, expectedOutput, resp.Content)
	assert.Equal(t, 250, resp.Usage.TotalTokens)
}

func TestLayout_WithMarkdownCodeBlock(t *testing.T) {
	expectedJSON := `{"sections":[{"type":"text","content":"优化后内容"}]}`

	mock := &MockLLMProvider{
		ChatFunc: func(ctx context.Context, messages []Message, opts ChatOpts) (*ChatResult, error) {
			// Return JSON wrapped in markdown code block
			return &ChatResult{
				Content: "```json\n" + expectedJSON + "\n```",
				Usage:   Usage{TotalTokens: 100},
			}, nil
		},
	}

	svc := NewAIService(mock)
	resp, err := svc.Layout(context.Background(), LayoutRequest{
		Content: json.RawMessage(`{"sections":[]}`),
	})
	require.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, json.RawMessage(expectedJSON), resp.Content)
}

func TestLayout_ProviderError(t *testing.T) {
	mock := &MockLLMProvider{
		ChatFunc: func(ctx context.Context, messages []Message, opts ChatOpts) (*ChatResult, error) {
			return nil, assert.AnError
		},
	}

	svc := NewAIService(mock)
	resp, err := svc.Layout(context.Background(), LayoutRequest{
		Content: json.RawMessage(`{}`),
	})
	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.Contains(t, err.Error(), "AI排版失败")
}

// ========== Proofread Tests ==========

func TestProofread_Success(t *testing.T) {
	correctionsJSON := `{"corrections":[{"position":5,"length":2,"type":"typo","original":"措别","suggestion":"错别","explanation":"汉字错误"}]}`
	expectedText := "这是一段有措别字的文字"

	mock := &MockLLMProvider{
		ChatFunc: func(ctx context.Context, messages []Message, opts ChatOpts) (*ChatResult, error) {
			assert.Equal(t, "system", messages[0].Role)
			assert.Contains(t, messages[0].Content, "校对")
			assert.Equal(t, "user", messages[1].Role)
			// user prompt wraps text with proofreading instr prefix; content assertion not needed
			assert.Equal(t, 0.3, opts.Temperature)
			return &ChatResult{
				Content: correctionsJSON,
				Usage: Usage{
					PromptTokens:     50,
					CompletionTokens: 80,
					TotalTokens:      130,
				},
			}, nil
		},
	}

	svc := NewAIService(mock)
	resp, err := svc.Proofread(context.Background(), ProofreadRequest{
		Text: expectedText,
	})
	require.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Len(t, resp.Corrections, 1)
	assert.Equal(t, 5, resp.Corrections[0].Position)
	assert.Equal(t, 2, resp.Corrections[0].Length)
	assert.Equal(t, "typo", resp.Corrections[0].Type)
	assert.Equal(t, "措别", resp.Corrections[0].Original)
	assert.Equal(t, "错别", resp.Corrections[0].Suggestion)
	assert.Equal(t, "汉字错误", resp.Corrections[0].Explanation)
	assert.Equal(t, 130, resp.Usage.TotalTokens)
}

func TestProofread_NoErrors(t *testing.T) {
	mock := &MockLLMProvider{
		ChatFunc: func(ctx context.Context, messages []Message, opts ChatOpts) (*ChatResult, error) {
			return &ChatResult{
				Content: `{"corrections":[]}`,
				Usage:   Usage{TotalTokens: 50},
			}, nil
		},
	}

	svc := NewAIService(mock)
	resp, err := svc.Proofread(context.Background(), ProofreadRequest{
		Text: "这是一段完全正确的文字。",
	})
	require.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Empty(t, resp.Corrections)
}

func TestProofread_InvalidJSONResponse(t *testing.T) {
	mock := &MockLLMProvider{
		ChatFunc: func(ctx context.Context, messages []Message, opts ChatOpts) (*ChatResult, error) {
			// Return invalid JSON that can't be parsed
			return &ChatResult{
				Content: "这不是JSON",
				Usage:   Usage{TotalTokens: 20},
			}, nil
		},
	}

	svc := NewAIService(mock)
	resp, err := svc.Proofread(context.Background(), ProofreadRequest{
		Text: "test",
	})
	// Should not error — returns empty corrections on parse failure
	require.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Empty(t, resp.Corrections)
}

func TestProofread_ProviderError(t *testing.T) {
	mock := &MockLLMProvider{
		ChatFunc: func(ctx context.Context, messages []Message, opts ChatOpts) (*ChatResult, error) {
			return nil, assert.AnError
		},
	}

	svc := NewAIService(mock)
	resp, err := svc.Proofread(context.Background(), ProofreadRequest{
		Text: "test",
	})
	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.Contains(t, err.Error(), "AI校对失败")
}

func TestProofread_MarkdownWrappedJSON(t *testing.T) {
	correctionsJSON := `{"corrections":[{"position":0,"length":3,"type":"grammar","original":"我吃饭","suggestion":"我吃了饭","explanation":"缺少了"}]}`

	mock := &MockLLMProvider{
		ChatFunc: func(ctx context.Context, messages []Message, opts ChatOpts) (*ChatResult, error) {
			return &ChatResult{
				Content: "```json\n" + correctionsJSON + "\n```",
				Usage:   Usage{TotalTokens: 60},
			}, nil
		},
	}

	svc := NewAIService(mock)
	resp, err := svc.Proofread(context.Background(), ProofreadRequest{
		Text: "我吃饭",
	})
	require.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Len(t, resp.Corrections, 1)
	assert.Equal(t, "grammar", resp.Corrections[0].Type)
}

// ========== OpenAILike Chat Tests ==========

func TestOpenAILike_ChatSuccess(t *testing.T) {
	// Create a test server that simulates OpenAI-compatible API
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "/v1/chat/completions", r.URL.Path)
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
		assert.Equal(t, "Bearer test-api-key", r.Header.Get("Authorization"))

		// Return mock response
		resp := openAIResponse{
			ID:      "chatcmpl-test123",
			Object:  "chat.completion",
			Created: time.Now().Unix(),
			Model:   "test-model",
			Choices: []openAIChoice{
				{
					Index: 0,
					Message: openAIMessage{
						Role:    "assistant",
						Content: "你好！我是AI助手。",
					},
					FinishReason: "stop",
				},
			},
			Usage: struct {
				PromptTokens     int `json:"prompt_tokens"`
				CompletionTokens int `json:"completion_tokens"`
				TotalTokens      int `json:"total_tokens"`
			}{
				PromptTokens:     10,
				CompletionTokens: 15,
				TotalTokens:      25,
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewOpenAILike(server.URL, "test-api-key", "test-model", 30*time.Second)
	result, err := client.Chat(context.Background(), []Message{
		{Role: "user", Content: "你好"},
	}, ChatOpts{Temperature: 0.7, MaxTokens: 100})

	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "你好！我是AI助手。", result.Content)
	assert.Equal(t, 10, result.Usage.PromptTokens)
	assert.Equal(t, 15, result.Usage.CompletionTokens)
	assert.Equal(t, 25, result.Usage.TotalTokens)
}

func TestOpenAILike_ChatErrorResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"error":{"message":"Internal server error","type":"server_error","code":"500"}}`))
	}))
	defer server.Close()

	client := NewOpenAILike(server.URL, "test-api-key", "test-model", 30*time.Second)
	result, err := client.Chat(context.Background(), []Message{
		{Role: "user", Content: "你好"},
	}, ChatOpts{})

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "500")
}

func TestOpenAILike_ChatHTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
		w.Write([]byte("rate limited"))
	}))
	defer server.Close()

	client := NewOpenAILike(server.URL, "test-api-key", "test-model", 30*time.Second)
	result, err := client.Chat(context.Background(), []Message{
		{Role: "user", Content: "你好"},
	}, ChatOpts{})

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "429")
}

func TestOpenAILike_ChatNetworkError(t *testing.T) {
	// Use an invalid URL to simulate network error
	client := NewOpenAILike("http://127.0.0.1:1", "test-api-key", "test-model", 1*time.Second)
	result, err := client.Chat(context.Background(), []Message{
		{Role: "user", Content: "你好"},
	}, ChatOpts{})

	assert.Error(t, err)
	assert.Nil(t, result)
}

func TestOpenAILike_ZeroTimeoutDefaults(t *testing.T) {
	client := NewOpenAILike("http://example.com", "key", "model", 0)
	assert.NotNil(t, client.httpClient)
	assert.Equal(t, 60*time.Second, client.httpClient.Timeout)
}

// ========== MockLLMProvider Tests ==========

func TestMockLLMProvider_NilChatFunc(t *testing.T) {
	mock := &MockLLMProvider{}
	result, err := mock.Chat(context.Background(), nil, ChatOpts{})
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Empty(t, result.Content)
	assert.Equal(t, Usage{}, result.Usage)
}

// ========== extractJSON Tests ==========

func TestExtractJSON_PlainJSON(t *testing.T) {
	input := `{"key": "value"}`
	result := extractJSON(input)
	assert.Equal(t, input, result)
}

func TestExtractJSON_WithMarkdownBlock(t *testing.T) {
	input := "```json\n{\"key\": \"value\"}\n```"
	result := extractJSON(input)
	assert.Equal(t, `{"key": "value"}`, result)
}

func TestExtractJSON_WithGenericMarkdownBlock(t *testing.T) {
	input := "```\n{\"key\": \"value\"}\n```"
	result := extractJSON(input)
	assert.Equal(t, `{"key": "value"}`, result)
}

func TestExtractJSON_WithWhitespace(t *testing.T) {
	input := "  \n```json\n{\"key\": \"value\"}\n```\n  "
	result := extractJSON(input)
	assert.Equal(t, `{"key": "value"}`, result)
}

// ========== LLMProvider Interface Test ==========

func TestLLMProviderInterface(t *testing.T) {
	// Verify OpenAILike implements LLMProvider
	var _ LLMProvider = (*OpenAILike)(nil)
	// Verify MockLLMProvider implements LLMProvider
	var _ LLMProvider = (*MockLLMProvider)(nil)
}

// ========== NewAIService Tests ==========

func TestNewAIService(t *testing.T) {
	mock := &MockLLMProvider{}
	svc := NewAIService(mock)
	assert.NotNil(t, svc)
	assert.NotNil(t, svc.provider)
}
