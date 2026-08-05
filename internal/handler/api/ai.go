// Package api AI 集成 Handler (T11)
package api

import (
	"net/http"

	"github.com/gin-gonic/gin"

	aiservice "github.com/weiyeston/weiyeston-v2/internal/service/ai"
)

// AIHandler AI 功能 API 处理器
type AIHandler struct {
	service *aiservice.AIService
}

// NewAIHandler 创建 AI Handler
func NewAIHandler(service *aiservice.AIService) *AIHandler {
	return &AIHandler{service: service}
}

// Write POST /api/v1/ai/write — AI帮写文章
func (h *AIHandler) Write(c *gin.Context) {
	var req aiservice.WriteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code": 40001,
			"msg":  "请求参数不合法: " + err.Error(),
			"data": nil,
		})
		return
	}

	resp, err := h.service.Write(c.Request.Context(), req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code": 50001,
			"msg":  err.Error(),
			"data": nil,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"msg":  "ok",
		"data": resp,
	})
}

// Layout POST /api/v1/ai/layout — AI智能排版
func (h *AIHandler) Layout(c *gin.Context) {
	var req aiservice.LayoutRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code": 40001,
			"msg":  "请求参数不合法: " + err.Error(),
			"data": nil,
		})
		return
	}

	resp, err := h.service.Layout(c.Request.Context(), req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code": 50001,
			"msg":  err.Error(),
			"data": nil,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"msg":  "ok",
		"data": resp,
	})
}

// Proofread POST /api/v1/ai/proofread — AI校对
func (h *AIHandler) Proofread(c *gin.Context) {
	var req aiservice.ProofreadRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code": 40001,
			"msg":  "请求参数不合法: " + err.Error(),
			"data": nil,
		})
		return
	}

	resp, err := h.service.Proofread(c.Request.Context(), req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code": 50001,
			"msg":  err.Error(),
			"data": nil,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"msg":  "ok",
		"data": resp,
	})
}
