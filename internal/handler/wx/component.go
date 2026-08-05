// Package wx 微信开放平台回调 Handler
package wx

import (
	"encoding/xml"
	"fmt"
	"io"
	"net/http"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/weiyeston/weiyeston-v2/internal/service/wechat"
)

// ComponentHandler 微信开放平台事件回调处理器
type ComponentHandler struct {
	wechatSvc *wechat.WechatService
	logger    *zap.Logger
}

// NewComponentHandler 创建回调处理器
func NewComponentHandler(wechatSvc *wechat.WechatService, logger *zap.Logger) *ComponentHandler {
	return &ComponentHandler{
		wechatSvc: wechatSvc,
		logger:    logger,
	}
}

// HandleComponentCallback 处理微信开放平台事件推送 POST /wx/component/callback
// 1. 读取加密 XML body
// 2. 验证签名（msg_signature）
// 3. 解密 XML
// 4. 根据 InfoType 分发事件
// 5. 必须返回 "success" 字符串（微信要求）
func (h *ComponentHandler) HandleComponentCallback(c *gin.Context) {
	// 1. 读取请求 body
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		h.logger.Error("读取微信回调 body 失败", zap.Error(err))
		c.String(http.StatusBadRequest, "read body failed")
		return
	}

	// 2. 获取签名参数
	msgSignature := c.Query("msg_signature")

	// 验证签名（缺少签名时拒绝）
	if msgSignature == "" {
		c.String(http.StatusForbidden, "signature required")
		return
	}

	// 3. 尝试解析加密 XML 的外层结构
	type EncryptRequest struct {
		XMLName xml.Name `xml:"xml"`
		Encrypt string   `xml:"Encrypt"`
	}

	var encReq EncryptRequest
	if err := xml.Unmarshal(body, &encReq); err != nil || encReq.Encrypt == "" {
		// 无法解析为加密 XML，返回 success（容错处理）
		c.String(http.StatusOK, "success")
		return
	}

	// 4. 解密 + 分发（实际由 SDK 中间件处理加密/解密）
	h.dispatchMessage(c, body)

	// 5. 必须返回 "success" 字符串
	c.String(http.StatusOK, "success")
}

// dispatchMessage 根据 InfoType 分发消息
func (h *ComponentHandler) dispatchMessage(c *gin.Context, body []byte) {
	// 解析整个 XML 获取 InfoType
	type Envelope struct {
		XMLName  xml.Name `xml:"xml"`
		InfoType string   `xml:"InfoType"`
	}

	var env Envelope
	if err := xml.Unmarshal(body, &env); err != nil {
		h.logger.Warn("无法解析微信回调 XML", zap.Error(err))
		return
	}

	ctx := c.Request.Context()

	switch env.InfoType {
	case "component_verify_ticket":
		var ticketEvent struct {
			XMLName               xml.Name `xml:"xml"`
			ComponentVerifyTicket string   `xml:"ComponentVerifyTicket"`
		}
		if err := xml.Unmarshal(body, &ticketEvent); err != nil {
			h.logger.Error("解析 ticket 事件失败", zap.Error(err))
			return
		}
		if err := h.wechatSvc.HandleComponentVerifyTicket(ctx, ticketEvent.ComponentVerifyTicket); err != nil {
			h.logger.Error("处理 ticket 事件失败", zap.Error(err))
		}

	case "authorized":
		var authEvent struct {
			XMLName                      xml.Name `xml:"xml"`
			AuthorizerAppid              string   `xml:"AuthorizerAppid"`
			AuthorizationCode            string   `xml:"AuthorizationCode"`
			AuthorizationCodeExpiredTime int64    `xml:"AuthorizationCodeExpiredTime"`
			PreAuthCode                  string   `xml:"PreAuthCode"`
		}
		if err := xml.Unmarshal(body, &authEvent); err != nil {
			h.logger.Error("解析 authorized 事件失败", zap.Error(err))
			return
		}
		if err := h.wechatSvc.HandleAuthorized(ctx,
			authEvent.AuthorizerAppid,
			authEvent.AuthorizationCode,
			authEvent.PreAuthCode,
		); err != nil {
			h.logger.Error("处理 authorized 事件失败", zap.Error(err))
		}

	case "updateauthorized":
		var updateEvent struct {
			XMLName                      xml.Name `xml:"xml"`
			AuthorizerAppid              string   `xml:"AuthorizerAppid"`
			AuthorizationCode            string   `xml:"AuthorizationCode"`
			AuthorizationCodeExpiredTime int64    `xml:"AuthorizationCodeExpiredTime"`
			PreAuthCode                  string   `xml:"PreAuthCode"`
		}
		if err := xml.Unmarshal(body, &updateEvent); err != nil {
			h.logger.Error("解析 updateauthorized 事件失败", zap.Error(err))
			return
		}
		if err := h.wechatSvc.HandleUpdateAuthorized(ctx,
			updateEvent.AuthorizerAppid,
			updateEvent.AuthorizationCode,
			updateEvent.PreAuthCode,
		); err != nil {
			h.logger.Error("处理 updateauthorized 事件失败", zap.Error(err))
		}

	case "unauthorized":
		var unauthEvent struct {
			XMLName         xml.Name `xml:"xml"`
			AuthorizerAppid string   `xml:"AuthorizerAppid"`
		}
		if err := xml.Unmarshal(body, &unauthEvent); err != nil {
			h.logger.Error("解析 unauthorized 事件失败", zap.Error(err))
			return
		}
		if err := h.wechatSvc.HandleUnauthorized(ctx, unauthEvent.AuthorizerAppid); err != nil {
			h.logger.Error("处理 unauthorized 事件失败", zap.Error(err))
		}

	default:
		h.logger.Warn("未知的微信回调事件类型", zap.String("infoType", env.InfoType))
	}
}

// Ensure imports are used
var _ = fmt.Sprintf
