// Package ai AI 服务 — LLM 提供方抽象 + 文章帮写/智能排版/校对
package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// ========== 基础类型 ==========

// Message represents a chat message (OpenAI-compatible format)
type Message struct {
	Role    string `json:"role"`    // system / user / assistant
	Content string `json:"content"`
}

// ChatOpts options for chat completion
type ChatOpts struct {
	Temperature float64 `json:"temperature,omitempty"`
	MaxTokens   int     `json:"max_tokens,omitempty"`
}

// ChatResult represents a chat completion result
type ChatResult struct {
	Content string `json:"content"`
	Usage   Usage  `json:"usage"`
}

// Usage token usage statistics
type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// ========== LLMProvider 接口 ==========

// LLMProvider interface — OpenAI-compatible chat protocol
type LLMProvider interface {
	Chat(ctx context.Context, messages []Message, opts ChatOpts) (*ChatResult, error)
}

// ========== OpenAI-compatible 实现 ==========

// OpenAILike implements LLMProvider for OpenAI-compatible APIs (DeepSeek, Qwen, etc.)
type OpenAILike struct {
	baseURL    string
	apiKey     string
	model      string
	httpClient *http.Client
}

// NewOpenAILike creates a new OpenAI-compatible LLM provider
func NewOpenAILike(baseURL, apiKey, model string, timeout time.Duration) *OpenAILike {
	if timeout <= 0 {
		timeout = 60 * time.Second
	}
	return &OpenAILike{
		baseURL: strings.TrimRight(baseURL, "/"),
		apiKey:  apiKey,
		model:   model,
		httpClient: &http.Client{
			Timeout: timeout,
		},
	}
}

// openAIRequest OpenAI-compatible chat completion request body
type openAIRequest struct {
	Model       string    `json:"model"`
	Messages    []Message `json:"messages"`
	Temperature float64   `json:"temperature,omitempty"`
	MaxTokens   int       `json:"max_tokens,omitempty"`
}

// openAIChoice represents a single completion choice
type openAIChoice struct {
	Index        int              `json:"index"`
	Message      openAIMessage    `json:"message"`
	FinishReason string           `json:"finish_reason"`
}

type openAIMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// openAIResponse OpenAI-compatible chat completion response body
type openAIResponse struct {
	ID      string         `json:"id"`
	Object  string         `json:"object"`
	Created int64          `json:"created"`
	Model   string         `json:"model"`
	Choices []openAIChoice `json:"choices"`
	Usage   struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage"`
	Error *struct {
		Message string `json:"message"`
		Type    string `json:"type"`
		Code    string `json:"code"`
	} `json:"error,omitempty"`
}

// Chat sends a chat completion request to an OpenAI-compatible API
func (o *OpenAILike) Chat(ctx context.Context, messages []Message, opts ChatOpts) (*ChatResult, error) {
	reqBody := openAIRequest{
		Model:       o.model,
		Messages:    messages,
		Temperature: opts.Temperature,
		MaxTokens:   opts.MaxTokens,
	}

	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("序列化请求失败: %w", err)
	}

	url := o.baseURL + "/v1/chat/completions"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+o.apiKey)

	resp, err := o.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("请求失败: %w", err)
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应失败: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API 返回错误状态 %d: %s", resp.StatusCode, string(respBytes))
	}

	var result openAIResponse
	if err := json.Unmarshal(respBytes, &result); err != nil {
		return nil, fmt.Errorf("解析响应失败: %w", err)
	}

	if result.Error != nil {
		return nil, fmt.Errorf("API 错误: %s (%s)", result.Error.Message, result.Error.Code)
	}

	if len(result.Choices) == 0 {
		return nil, fmt.Errorf("API 未返回任何内容")
	}

	return &ChatResult{
		Content: result.Choices[0].Message.Content,
		Usage: Usage{
			PromptTokens:     result.Usage.PromptTokens,
			CompletionTokens: result.Usage.CompletionTokens,
			TotalTokens:      result.Usage.TotalTokens,
		},
	}, nil
}

// ========== AIService ==========

// AIService provides AI capabilities (article writing, layout, proofreading)
type AIService struct {
	provider LLMProvider
}

// NewAIService creates a new AI service
func NewAIService(provider LLMProvider) *AIService {
	return &AIService{provider: provider}
}

// ========== AI帮写文章 ==========

// WriteRequest AI帮写文章请求
type WriteRequest struct {
	Title    string `json:"title" binding:"required"`
	Keywords string `json:"keywords"`
	Style    string `json:"style"`     // 正式/轻松/活泼
	MaxWords int    `json:"max_words"` // 字数限制
}

// WriteResponse AI帮写文章响应
type WriteResponse struct {
	Content string `json:"content"` // markdown content
	Usage   Usage  `json:"usage"`
}

// Write generates an article using AI
func (s *AIService) Write(ctx context.Context, req WriteRequest) (*WriteResponse, error) {
	styleDesc := "正式专业"
	switch req.Style {
	case "轻松":
		styleDesc = "轻松自然，通俗易懂"
	case "活泼":
		styleDesc = "活泼有趣，充满活力"
	}

	wordLimit := ""
	if req.MaxWords > 0 {
		wordLimit = fmt.Sprintf("，字数控制在%d字左右", req.MaxWords)
	}

	keywordsLine := ""
	if req.Keywords != "" {
		keywordsLine = fmt.Sprintf("关键词：%s\n", req.Keywords)
	}

	systemPrompt := "你是一位资深的微信公众号小编，擅长撰写吸引人的公众号文章。你的文章结构清晰、语言生动、排版美观，使用 Markdown 格式。"

	userPrompt := fmt.Sprintf(`请帮我写一篇微信公众号文章。

标题：%s
%s风格：%s%s

要求：
1. 使用 Markdown 格式
2. 包含吸引人的开头
3. 内容结构清晰，有小标题分段
4. 语言符合公众号风格
5. 结尾有互动引导`, req.Title, keywordsLine, styleDesc, wordLimit)

	messages := []Message{
		{Role: "system", Content: systemPrompt},
		{Role: "user", Content: userPrompt},
	}

	result, err := s.provider.Chat(ctx, messages, ChatOpts{Temperature: 0.8, MaxTokens: 4096})
	if err != nil {
		return nil, fmt.Errorf("AI帮写失败: %w", err)
	}

	return &WriteResponse{
		Content: result.Content,
		Usage:   result.Usage,
	}, nil
}

// ========== AI智能排版 ==========

// LayoutRequest AI智能排版请求
type LayoutRequest struct {
	Content json.RawMessage `json:"content" binding:"required"` // current JSON content
}

// LayoutResponse AI智能排版响应
type LayoutResponse struct {
	Content json.RawMessage `json:"content"` // optimized JSON structure
	Usage   Usage           `json:"usage"`
}

// Layout optimizes page layout using AI
func (s *AIService) Layout(ctx context.Context, req LayoutRequest) (*LayoutResponse, error) {
	systemPrompt := `你是一位专业的网页排版设计师。用户会给你一个 JSON 格式的页面内容结构，你需要分析其内容并在保持整体信息完整的前提下优化排版。

优化原则：
1. 分析段落逻辑关系，调整顺序使其更有层次感
2. 将大段文字拆分为更易读的段落
3. 为重要的内容建议添加合适的组件类型标记
4. 保持原始 JSON 结构，仅调整内容位置和分组

返回格式：仅返回优化后的 JSON，不要包含任何其他文字。`

	userPrompt := fmt.Sprintf("请优化以下页面内容的排版结构：\n\n```json\n%s\n```", string(req.Content))

	messages := []Message{
		{Role: "system", Content: systemPrompt},
		{Role: "user", Content: userPrompt},
	}

	result, err := s.provider.Chat(ctx, messages, ChatOpts{Temperature: 0.5, MaxTokens: 4096})
	if err != nil {
		return nil, fmt.Errorf("AI排版失败: %w", err)
	}

	// Try to extract JSON from the response (may be inside markdown code blocks)
	content := extractJSON(result.Content)

	return &LayoutResponse{
		Content: json.RawMessage(content),
		Usage:   result.Usage,
	}, nil
}

// ========== AI校对 ==========

// ProofreadRequest AI校对请求
type ProofreadRequest struct {
	Text string `json:"text" binding:"required"`
}

// Correction represents a single correction suggestion
type Correction struct {
	Position    int    `json:"position"`    // character position
	Length      int    `json:"length"`      // length of problematic text
	Type        string `json:"type"`        // typo/grammar/sensitive/style
	Original    string `json:"original"`    // original text
	Suggestion  string `json:"suggestion"`  // suggested fix
	Explanation string `json:"explanation"` // explanation
}

// ProofreadResponse AI校对响应
type ProofreadResponse struct {
	Corrections []Correction `json:"corrections"`
	Usage       Usage        `json:"usage"`
}

// proofreadResult raw AI response structure for proofreading
type proofreadResult struct {
	Corrections []proofreadCorrection `json:"corrections"`
}

type proofreadCorrection struct {
	Position    int    `json:"position"`
	Length      int    `json:"length"`
	Type        string `json:"type"`
	Original    string `json:"original"`
	Suggestion  string `json:"suggestion"`
	Explanation string `json:"explanation"`
}

// Proofread checks text for errors
func (s *AIService) Proofread(ctx context.Context, req ProofreadRequest) (*ProofreadResponse, error) {
	systemPrompt := `你是一位专业的文字校对编辑，擅长发现文本中的各种问题。

请对用户提供的文本进行仔细校对，检查以下方面：
1. 错别字（typo）
2. 语法错误（grammar）
3. 敏感词或不恰当表达（sensitive）
4. 风格问题（style）

请以 JSON 数组格式返回所有发现的问题，每个问题包含以下字段：
- position: 问题文本的起始位置（从0开始计数，char位置）
- length: 问题文本的长度
- type: 问题类型（typo/grammar/sensitive/style）
- original: 原文中的问题文本
- suggestion: 建议修改为
- explanation: 修改说明

如果没有发现任何问题，返回空数组 []。

请严格按照 JSON 格式返回，不要包含任何其他文字。`

	userPrompt := fmt.Sprintf("请校对以下文本：\n\n%s", req.Text)

	messages := []Message{
		{Role: "system", Content: systemPrompt},
		{Role: "user", Content: userPrompt},
	}

	result, err := s.provider.Chat(ctx, messages, ChatOpts{Temperature: 0.3, MaxTokens: 4096})
	if err != nil {
		return nil, fmt.Errorf("AI校对失败: %w", err)
	}

	// Parse the JSON correction list
	content := extractJSON(result.Content)

	var rawResult proofreadResult
	if err := json.Unmarshal([]byte(content), &rawResult); err != nil {
		// If parsing fails, return empty corrections with usage info
		return &ProofreadResponse{
			Corrections: []Correction{},
			Usage:       result.Usage,
		}, nil
	}

	corrections := make([]Correction, 0, len(rawResult.Corrections))
	for _, c := range rawResult.Corrections {
		corrections = append(corrections, Correction{
			Position:    c.Position,
			Length:      c.Length,
			Type:        c.Type,
			Original:    c.Original,
			Suggestion:  c.Suggestion,
			Explanation: c.Explanation,
		})
	}

	if corrections == nil {
		corrections = []Correction{}
	}

	return &ProofreadResponse{
		Corrections: corrections,
		Usage:       result.Usage,
	}, nil
}

// ========== MockLLMProvider ==========

// MockLLMProvider for testing
type MockLLMProvider struct {
	ChatFunc func(ctx context.Context, messages []Message, opts ChatOpts) (*ChatResult, error)
}

// Chat calls the mock function
func (m *MockLLMProvider) Chat(ctx context.Context, messages []Message, opts ChatOpts) (*ChatResult, error) {
	if m.ChatFunc != nil {
		return m.ChatFunc(ctx, messages, opts)
	}
	return &ChatResult{
		Content: "",
		Usage:   Usage{},
	}, nil
}

// ========== 辅助函数 ==========

// extractJSON extracts JSON from a string that may contain markdown code blocks
func extractJSON(s string) string {
	s = strings.TrimSpace(s)

	// Try to extract from ```json ... ``` block
	if strings.HasPrefix(s, "```") {
		// Find the first newline after ```
		idx := strings.Index(s, "\n")
		if idx == -1 {
			return s
		}
		// Remove opening ```json or ```
		rest := s[idx+1:]
		// Remove closing ```
		if lastIdx := strings.LastIndex(rest, "```"); lastIdx != -1 {
			rest = rest[:lastIdx]
		}
		return strings.TrimSpace(rest)
	}

	return s
}
