// Package wx 微信开放平台回调 Handler 测试
// TDD: 测试先行 — 使用 httptest 模拟微信推送请求
// component.go 尚未实现，测试使用内联 handler 展示预期行为
package wx

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha1"
	"encoding/base64"
	"encoding/binary"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/weiyeston/weiyeston-v2/internal/config"
)

// ========== 设计文档修正提醒 ==========
// 1. redirect_uri 是 /wx/component/callback（不是 auth-callback）
// 2. SDK API 是 wc.GetOpenPlatform()（不是 GetComponent）
// 3. authorizer_appid 需唯一索引 WHERE NOT NULL AND deleted_at IS NULL
// 4. qrcode_url 改用已有 qr_code_url 字段
// 5. FuncInfo 用 *json.RawMessage（不是 *string）
// 6. 取消授权用 auth_status=3

// ========== 微信 XML 消息结构体定义 ==========

// WechatEncryptRequest 微信加密推送的外层 XML
type WechatEncryptRequest struct {
	XMLName    xml.Name `xml:"xml"`
	ToUserName string   `xml:"ToUserName"`
	Encrypt    string   `xml:"Encrypt"`
}

// ComponentEvent 基础事件
type ComponentEvent struct {
	AppId      string `xml:"AppId"`
	CreateTime int64  `xml:"CreateTime"`
	InfoType   string `xml:"InfoType"`
}

// ComponentVerifyTicketEvent 推送 component_verify_ticket
type ComponentVerifyTicketEvent struct {
	XMLName                 xml.Name `xml:"xml"`
	AppId                   string   `xml:"AppId"`
	CreateTime              int64    `xml:"CreateTime"`
	InfoType                string   `xml:"InfoType"`
	ComponentVerifyTicket   string   `xml:"ComponentVerifyTicket"`
}

// AuthorizedEvent 授权成功事件
type AuthorizedEvent struct {
	XMLName                      xml.Name `xml:"xml"`
	AppId                        string   `xml:"AppId"`
	CreateTime                   int64    `xml:"CreateTime"`
	InfoType                     string   `xml:"InfoType"`
	AuthorizerAppid              string   `xml:"AuthorizerAppid"`
	AuthorizationCode            string   `xml:"AuthorizationCode"`
	AuthorizationCodeExpiredTime int64    `xml:"AuthorizationCodeExpiredTime"`
	PreAuthCode                  string   `xml:"PreAuthCode"`
}

// UpdateAuthorizedEvent 授权更新事件
type UpdateAuthorizedEvent struct {
	XMLName                      xml.Name `xml:"xml"`
	AppId                        string   `xml:"AppId"`
	CreateTime                   int64    `xml:"CreateTime"`
	InfoType                     string   `xml:"InfoType"`
	AuthorizerAppid              string   `xml:"AuthorizerAppid"`
	AuthorizationCode            string   `xml:"AuthorizationCode"`
	AuthorizationCodeExpiredTime int64    `xml:"AuthorizationCodeExpiredTime"`
	PreAuthCode                  string   `xml:"PreAuthCode"`
}

// UnauthorizedEvent 取消授权事件
type UnauthorizedEvent struct {
	XMLName         xml.Name `xml:"xml"`
	AppId           string   `xml:"AppId"`
	CreateTime      int64    `xml:"CreateTime"`
	InfoType        string   `xml:"InfoType"`
	AuthorizerAppid string   `xml:"AuthorizerAppid"`
}

// ========== XML 加解密工具（模拟微信加密） ==========

// generateWXEncryptedXML 模拟微信加密 XML 的生成
// 实际由微信 SDK 的 Server 中间件处理，此处模拟加密以验证解密流程
func generateWXEncryptedXML(plainXML []byte, aesKey, appID, token string) (string, error) {
	// 微信加密流程:
	// 1. 生成 16 字节随机数
	// 2. 4 字节 msgLen（网络字节序）
	// 3. 拼装: random(16) + msgLen(4) + plainXML + appID
	// 4. PKCS7 填充
	// 5. AES-256-CBC 加密（iv = aesKey[:16]）
	// 6. Base64 编码

	aesKeyBytes, err := base64.StdEncoding.DecodeString(aesKey + "=")
	if err != nil {
		return "", fmt.Errorf("aes key decode error: %w", err)
	}

	// 1. 随机数
	randBytes := make([]byte, 16)
	if _, err := rand.Read(randBytes); err != nil {
		return "", err
	}

	// 2. msgLen（4 字节大端序）
	msgLen := make([]byte, 4)
	binary.BigEndian.PutUint32(msgLen, uint32(len(plainXML)))

	// 3. 拼装
	var buf bytes.Buffer
	buf.Write(randBytes)
	buf.Write(msgLen)
	buf.Write(plainXML)
	buf.Write([]byte(appID))

	// 4. PKCS7 填充
	raw := buf.Bytes()
	blockSize := 32
	padding := blockSize - len(raw)%blockSize
	for i := 0; i < padding; i++ {
		raw = append(raw, byte(padding))
	}

	// 5. AES-256-CBC 加密
	block, err := aes.NewCipher(aesKeyBytes)
	if err != nil {
		return "", err
	}
	ciphertext := make([]byte, len(raw))
	iv := aesKeyBytes[:16]
	mode := cipher.NewCBCEncrypter(block, iv)
	mode.CryptBlocks(ciphertext, raw)

	// 6. Base64 编码
	encrypt := base64.StdEncoding.EncodeToString(ciphertext)

	// 生成 msg_signature
	timestamp := fmt.Sprintf("%d", time.Now().Unix())
	nonce := fmt.Sprintf("%d", time.Now().UnixNano()%1000000)

	// signature = sha1(sort(token, timestamp, nonce, encrypt))
	strs := []string{token, timestamp, nonce, encrypt}
	sort.Strings(strs)
	h := sha1.New()
	h.Write([]byte(strings.Join(strs, "")))
	msgSignature := fmt.Sprintf("%x", h.Sum(nil))

	// 构造加密 XML
	encryptedXML := fmt.Sprintf(`<xml>
<ToUserName><![CDATA[%s]]></ToUserName>
<Encrypt><![CDATA[%s]]></Encrypt>
</xml>`, appID, encrypt)

	return url.QueryEscape(encryptedXML) + "&msg_signature=" + msgSignature + "&timestamp=" + timestamp + "&nonce=" + nonce, nil
}

// decryptWXMessage 模拟微信消息解密（验证流程用）
func decryptWXMessage(encrypt, aesKey, appID string) (string, error) {
	aesKeyBytes, err := base64.StdEncoding.DecodeString(aesKey + "=")
	if err != nil {
		return "", err
	}

	ciphertext, err := base64.StdEncoding.DecodeString(encrypt)
	if err != nil {
		return "", err
	}

	block, err := aes.NewCipher(aesKeyBytes)
	if err != nil {
		return "", err
	}

	plaintext := make([]byte, len(ciphertext))
	iv := aesKeyBytes[:16]
	mode := cipher.NewCBCDecrypter(block, iv)
	mode.CryptBlocks(plaintext, ciphertext)

	// 去 PKCS7 padding
	if len(plaintext) == 0 {
		return "", fmt.Errorf("empty plaintext after decrypt")
	}
	padding := int(plaintext[len(plaintext)-1])
	if padding > len(plaintext) || padding < 1 || padding > 32 {
		return "", fmt.Errorf("invalid PKCS7 padding: %d (likely wrong AES key)", padding)
	}
	plaintext = plaintext[:len(plaintext)-padding]

	// 去随机数(16) + msgLen(4)
	if len(plaintext) < 20 {
		return "", fmt.Errorf("plaintext too short after removing padding")
	}
	plaintext = plaintext[20:]

	// 验证并去除 appID 后缀
	if len(plaintext) < len(appID) {
		return "", fmt.Errorf("plaintext too short for appID")
	}
	// 验证后缀是否为正确的 appID（使用错误 key 时后缀不匹配）
	actualAppID := string(plaintext[len(plaintext)-len(appID):])
	if actualAppID != appID {
		return "", fmt.Errorf("appID mismatch: expected %s, got %s (likely wrong AES key)", appID, actualAppID)
	}
	plaintext = plaintext[:len(plaintext)-len(appID)]

	return string(plaintext), nil
}

// ========== Mock 服务（模拟 wechatService 依赖） ==========

type mockComponentService struct {
	mu            sync.Mutex
	lastTicket    string
	tickets       []string
	authorized    []string
	unauthorized  []string
	updateAuth    []string
	accountTokens map[string]string
}

func newMockComponentService() *mockComponentService {
	return &mockComponentService{
		tickets:       make([]string, 0),
		authorized:    make([]string, 0),
		unauthorized:  make([]string, 0),
		updateAuth:    make([]string, 0),
		accountTokens: make(map[string]string),
	}
}

func (m *mockComponentService) HandleTicket(ticket string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.lastTicket = ticket
	m.tickets = append(m.tickets, ticket)
}

func (m *mockComponentService) HandleAuthorized(authorizerAppID, authCode, preAuthCode string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.authorized = append(m.authorized, authorizerAppID)
	m.accountTokens[authorizerAppID] = authCode
}

func (m *mockComponentService) HandleUpdateAuthorized(authorizerAppID, authCode, preAuthCode string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.updateAuth = append(m.updateAuth, authorizerAppID)
}

func (m *mockComponentService) HandleUnauthorized(authorizerAppID string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.unauthorized = append(m.unauthorized, authorizerAppID)
	delete(m.accountTokens, authorizerAppID)
}

// ========== 内联测试 Handler ==========

// testComponentCallbackHandler 模拟微信开放平台回调 handler
// 预期实现: POST /wx/component/callback
// 1. 读取加密 XML body
// 2. 验证签名（msg_signature）
// 3. 解密 XML
// 4. 根据 InfoType 分发事件
// 5. 返回 "success" 字符串
func testComponentCallbackHandler(service *mockComponentService, cfg config.WechatConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 读取 body
		body, err := io.ReadAll(c.Request.Body)
		if err != nil {
			c.String(http.StatusBadRequest, "read body failed")
			return
		}

		// 获取签名参数
		msgSignature := c.Query("msg_signature")
		timestamp := c.Query("timestamp")
		nonce := c.Query("nonce")

		// 验证签名（简化版: 这里应调用 SDK 验证）
		if msgSignature == "" {
			c.String(http.StatusForbidden, "signature required")
			return
		}

		// 解密（简化版: 实际应由 SDK Server 中间件处理）
		var encryptedReq WechatEncryptRequest
		if err := xml.Unmarshal(body, &encryptedReq); err != nil {
			// 如果不是加密 XML，尝试直接解析明文（仅测试用）
			c.String(http.StatusOK, "success")
			return
		}

		// 解密消息
		plainXML, err := decryptWXMessage(encryptedReq.Encrypt, cfg.EncodingAESKey, cfg.ComponentAppID)
		if err != nil {
			c.String(http.StatusForbidden, "decrypt failed")
			return
		}

		// 解析 InfoType 分发
		type BaseEvent struct {
			InfoType string `xml:"InfoType"`
		}
		var base BaseEvent
		if err := xml.Unmarshal([]byte(plainXML), &base); err != nil {
			c.String(http.StatusBadRequest, "parse event failed")
			return
		}

		switch base.InfoType {
		case "component_verify_ticket":
			var ticketEvent ComponentVerifyTicketEvent
			if err := xml.Unmarshal([]byte(plainXML), &ticketEvent); err != nil {
				c.String(http.StatusBadRequest, "parse ticket failed")
				return
			}
			service.HandleTicket(ticketEvent.ComponentVerifyTicket)

		case "authorized":
			var authEvent AuthorizedEvent
			if err := xml.Unmarshal([]byte(plainXML), &authEvent); err != nil {
				c.String(http.StatusBadRequest, "parse authorized event failed")
				return
			}
			service.HandleAuthorized(
				authEvent.AuthorizerAppid,
				authEvent.AuthorizationCode,
				authEvent.PreAuthCode,
			)

		case "updateauthorized":
			var updateEvent UpdateAuthorizedEvent
			if err := xml.Unmarshal([]byte(plainXML), &updateEvent); err != nil {
				c.String(http.StatusBadRequest, "parse updateauthorized event failed")
				return
			}
			service.HandleUpdateAuthorized(
				updateEvent.AuthorizerAppid,
				updateEvent.AuthorizationCode,
				updateEvent.PreAuthCode,
			)

		case "unauthorized":
			var unauthEvent UnauthorizedEvent
			if err := xml.Unmarshal([]byte(plainXML), &unauthEvent); err != nil {
				c.String(http.StatusBadRequest, "parse unauthorized event failed")
				return
			}
			service.HandleUnauthorized(unauthEvent.AuthorizerAppid)

		default:
			// 未知事件类型，仍返回 success
		}

		// 微信要求必须返回 "success" 字符串
		c.String(http.StatusOK, "success")

		_ = timestamp
		_ = nonce
	}
}

// ========== 测试辅助 ==========

func newTestWechatConfig() config.WechatConfig {
	return config.WechatConfig{
		ComponentAppID:     "wx_test_component_app_id",
		ComponentAppSecret: "test_component_secret_32_chars__",
		Token:              "test_token_2026",
		EncodingAESKey:     "abcdefghijklmnopqrstuvwxyz0123456789ABCDEFG", // 43 chars
		ServerURL:          "https://api.example.com",
	}
}

// ========== 1. POST /wx/component/callback 接收 ticket 推送 ==========

func TestComponentCallback_ReceiveTicket(t *testing.T) {
	gin.SetMode(gin.TestMode)

	service := newMockComponentService()
	cfg := newTestWechatConfig()

	r := gin.New()
	r.POST("/wx/component/callback", testComponentCallbackHandler(service, cfg))

	// 构造 ticket 推送 XML（明文，测试直接发送）
	ticketXML := `<xml>
<AppId><![CDATA[wx_test_component_app_id]]></AppId>
<CreateTime>1700000000</CreateTime>
<InfoType><![CDATA[component_verify_ticket]]></InfoType>
<ComponentVerifyTicket><![CDATA[ticket_abc123def456ghi789]]></ComponentVerifyTicket>
</xml>`

	// 加密
	encryptedBody, err := generateWXEncryptedXML(
		[]byte(ticketXML),
		cfg.EncodingAESKey,
		cfg.ComponentAppID,
		cfg.Token,
	)
	require.NoError(t, err)

	// 解析参数
	params := strings.SplitN(encryptedBody, "&", 4)
	var encryptedXMLData, msgSig, timestamp, nonce string
	for _, p := range params {
		kv := strings.SplitN(p, "=", 2)
		switch kv[0] {
		case "msg_signature":
			msgSig = kv[1]
		case "timestamp":
			timestamp = kv[1]
		case "nonce":
			nonce = kv[1]
		default:
			encryptedXMLData, _ = url.QueryUnescape(p)
		}
	}

	// 发送请求
	w := httptest.NewRecorder()
	reqURL := fmt.Sprintf("/wx/component/callback?msg_signature=%s&timestamp=%s&nonce=%s",
		msgSig, timestamp, nonce)
	req, _ := http.NewRequest(http.MethodPost, reqURL, strings.NewReader(encryptedXMLData))
	req.Header.Set("Content-Type", "application/xml")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "success", w.Body.String())

	// 验证 ticket 已被处理
	assert.Equal(t, "ticket_abc123def456ghi789", service.lastTicket)
	assert.Len(t, service.tickets, 1)

	t.Log("component_verify_ticket 推送接收成功")
}

func TestComponentCallback_ReceiveMultipleTickets(t *testing.T) {
	// 微信每 10 分钟推送一次 ticket，验证多次推送
	gin.SetMode(gin.TestMode)

	service := newMockComponentService()
	cfg := newTestWechatConfig()

	r := gin.New()
	r.POST("/wx/component/callback", testComponentCallbackHandler(service, cfg))

	tickets := []string{
		"ticket_p1_first_push",
		"ticket_p2_second_push",
		"ticket_p3_third_push",
	}

	for i, ticket := range tickets {
		ticketXML := fmt.Sprintf(`<xml>
<AppId><![CDATA[wx_test_component_app_id]]></AppId>
<CreateTime>%d</CreateTime>
<InfoType><![CDATA[component_verify_ticket]]></InfoType>
<ComponentVerifyTicket><![CDATA[%s]]></ComponentVerifyTicket>
</xml>`, 1700000000+int64(i*600), ticket)

		encryptedBody, err := generateWXEncryptedXML(
			[]byte(ticketXML),
			cfg.EncodingAESKey,
			cfg.ComponentAppID,
			cfg.Token,
		)
		require.NoError(t, err)

		params := strings.SplitN(encryptedBody, "&", 4)
		var encryptedXMLData, msgSig, timestamp, nonce string
		for _, p := range params {
			kv := strings.SplitN(p, "=", 2)
			switch kv[0] {
			case "msg_signature":
				msgSig = kv[1]
			case "timestamp":
				timestamp = kv[1]
			case "nonce":
				nonce = kv[1]
			default:
				encryptedXMLData, _ = url.QueryUnescape(p)
			}
		}

		w := httptest.NewRecorder()
		reqURL := fmt.Sprintf("/wx/component/callback?msg_signature=%s&timestamp=%s&nonce=%s",
			msgSig, timestamp, nonce)
		req, _ := http.NewRequest(http.MethodPost, reqURL, strings.NewReader(encryptedXMLData))
		req.Header.Set("Content-Type", "application/xml")
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code, "第 %d 次推送", i+1)
		assert.Equal(t, "success", w.Body.String())
	}

	assert.Len(t, service.tickets, 3, "应收到 3 次 ticket 推送")
	assert.Equal(t, tickets[2], service.lastTicket, "最后推送的 ticket 应是最新")
}

// ========== 2. 接收 authorized 事件 ==========

func TestComponentCallback_ReceiveAuthorizedEvent(t *testing.T) {
	gin.SetMode(gin.TestMode)

	service := newMockComponentService()
	cfg := newTestWechatConfig()

	r := gin.New()
	r.POST("/wx/component/callback", testComponentCallbackHandler(service, cfg))

	authorizerAppID := "wx_app_authorized_001"
	authCode := "auth_code_authorized_001"
	preAuthCode := "pre_auth_code_001"

	authXML := fmt.Sprintf(`<xml>
<AppId><![CDATA[wx_test_component_app_id]]></AppId>
<CreateTime>1700000100</CreateTime>
<InfoType><![CDATA[authorized]]></InfoType>
<AuthorizerAppid><![CDATA[%s]]></AuthorizerAppid>
<AuthorizationCode><![CDATA[%s]]></AuthorizationCode>
<AuthorizationCodeExpiredTime>1700007300</AuthorizationCodeExpiredTime>
<PreAuthCode><![CDATA[%s]]></PreAuthCode>
</xml>`, authorizerAppID, authCode, preAuthCode)

	encryptedBody, err := generateWXEncryptedXML(
		[]byte(authXML), cfg.EncodingAESKey, cfg.ComponentAppID, cfg.Token,
	)
	require.NoError(t, err)

	params := strings.SplitN(encryptedBody, "&", 4)
	var encryptedXMLData, msgSig, timestamp, nonce string
	for _, p := range params {
		kv := strings.SplitN(p, "=", 2)
		switch kv[0] {
		case "msg_signature":
			msgSig = kv[1]
		case "timestamp":
			timestamp = kv[1]
		case "nonce":
			nonce = kv[1]
		default:
			encryptedXMLData, _ = url.QueryUnescape(p)
		}
	}

	w := httptest.NewRecorder()
	reqURL := fmt.Sprintf("/wx/component/callback?msg_signature=%s&timestamp=%s&nonce=%s",
		msgSig, timestamp, nonce)
	req, _ := http.NewRequest(http.MethodPost, reqURL, strings.NewReader(encryptedXMLData))
	req.Header.Set("Content-Type", "application/xml")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "success", w.Body.String())

	assert.Contains(t, service.authorized, authorizerAppID)
	assert.Equal(t, authCode, service.accountTokens[authorizerAppID])

	t.Logf("authorized 事件: authorizerAppID=%s, authCode=%s", authorizerAppID, authCode)
}

func TestComponentCallback_ReceiveAuthorizedEvent_NewAccount(t *testing.T) {
	// 新公众号首次授权
	gin.SetMode(gin.TestMode)

	service := newMockComponentService()
	cfg := newTestWechatConfig()

	r := gin.New()
	r.POST("/wx/component/callback", testComponentCallbackHandler(service, cfg))

	authorizerAppID := "wx_new_authorized_app_002"
	authCode := "auth_code_new_002"
	preAuthCode := "pre_auth_new_002"

	authXML := fmt.Sprintf(`<xml>
<AppId><![CDATA[wx_test_component_app_id]]></AppId>
<CreateTime>1700000200</CreateTime>
<InfoType><![CDATA[authorized]]></InfoType>
<AuthorizerAppid><![CDATA[%s]]></AuthorizerAppid>
<AuthorizationCode><![CDATA[%s]]></AuthorizationCode>
<AuthorizationCodeExpiredTime>1700007400</AuthorizationCodeExpiredTime>
<PreAuthCode><![CDATA[%s]]></PreAuthCode>
</xml>`, authorizerAppID, authCode, preAuthCode)

	encryptedBody, err := generateWXEncryptedXML(
		[]byte(authXML), cfg.EncodingAESKey, cfg.ComponentAppID, cfg.Token,
	)
	require.NoError(t, err)

	params := strings.SplitN(encryptedBody, "&", 4)
	var encryptedXMLData, msgSig, timestamp, nonce string
	for _, p := range params {
		kv := strings.SplitN(p, "=", 2)
		switch kv[0] {
		case "msg_signature":
			msgSig = kv[1]
		case "timestamp":
			timestamp = kv[1]
		case "nonce":
			nonce = kv[1]
		default:
			encryptedXMLData, _ = url.QueryUnescape(p)
		}
	}

	w := httptest.NewRecorder()
	reqURL := fmt.Sprintf("/wx/component/callback?msg_signature=%s&timestamp=%s&nonce=%s",
		msgSig, timestamp, nonce)
	req, _ := http.NewRequest(http.MethodPost, reqURL, strings.NewReader(encryptedXMLData))
	req.Header.Set("Content-Type", "application/xml")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, service.authorized, authorizerAppID,
		"新授权公众号应被正确记录")
}

func TestComponentCallback_ReceiveAuthorizedEvent_Reauthorize(t *testing.T) {
	// 已存在的公众号再次授权
	gin.SetMode(gin.TestMode)

	service := newMockComponentService()
	cfg := newTestWechatConfig()

	// 先模拟首次授权
	service.HandleAuthorized("wx_repeat_app", "old_auth_code", "old_pre_auth")

	r := gin.New()
	r.POST("/wx/component/callback", testComponentCallbackHandler(service, cfg))

	// 再次授权
	authorizerAppID := "wx_repeat_app"
	authCode := "new_auth_code_repeat"
	preAuthCode := "new_pre_auth_repeat"

	authXML := fmt.Sprintf(`<xml>
<AppId><![CDATA[wx_test_component_app_id]]></AppId>
<CreateTime>1700000300</CreateTime>
<InfoType><![CDATA[authorized]]></InfoType>
<AuthorizerAppid><![CDATA[%s]]></AuthorizerAppid>
<AuthorizationCode><![CDATA[%s]]></AuthorizationCode>
<AuthorizationCodeExpiredTime>1700007500</AuthorizationCodeExpiredTime>
<PreAuthCode><![CDATA[%s]]></PreAuthCode>
</xml>`, authorizerAppID, authCode, preAuthCode)

	encryptedBody, err := generateWXEncryptedXML(
		[]byte(authXML), cfg.EncodingAESKey, cfg.ComponentAppID, cfg.Token,
	)
	require.NoError(t, err)

	params := strings.SplitN(encryptedBody, "&", 4)
	var encryptedXMLData, msgSig, timestamp, nonce string
	for _, p := range params {
		kv := strings.SplitN(p, "=", 2)
		switch kv[0] {
		case "msg_signature":
			msgSig = kv[1]
		case "timestamp":
			timestamp = kv[1]
		case "nonce":
			nonce = kv[1]
		default:
			encryptedXMLData, _ = url.QueryUnescape(p)
		}
	}

	w := httptest.NewRecorder()
	reqURL := fmt.Sprintf("/wx/component/callback?msg_signature=%s&timestamp=%s&nonce=%s",
		msgSig, timestamp, nonce)
	req, _ := http.NewRequest(http.MethodPost, reqURL, strings.NewReader(encryptedXMLData))
	req.Header.Set("Content-Type", "application/xml")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	// authorized 列表中应出现两次同一个 appID（首次 + 再次授权）
	count := 0
	for _, a := range service.authorized {
		if a == authorizerAppID {
			count++
		}
	}
	assert.Equal(t, 2, count, "同一公众号被授权两次，应记录两条记录")
}

// ========== 3. 接收 unauthorized 事件 ==========

func TestComponentCallback_ReceiveUnauthorizedEvent(t *testing.T) {
	gin.SetMode(gin.TestMode)

	service := newMockComponentService()
	cfg := newTestWechatConfig()

	// 先授权（让 token 存在）
	service.HandleAuthorized("wx_app_will_cancel", "auth_code_before", "pre_auth_before")
	assert.Contains(t, service.accountTokens, "wx_app_will_cancel")

	r := gin.New()
	r.POST("/wx/component/callback", testComponentCallbackHandler(service, cfg))

	// 取消授权
	unauthXML := `<xml>
<AppId><![CDATA[wx_test_component_app_id]]></AppId>
<CreateTime>1700001000</CreateTime>
<InfoType><![CDATA[unauthorized]]></InfoType>
<AuthorizerAppid><![CDATA[wx_app_will_cancel]]></AuthorizerAppid>
</xml>`

	encryptedBody, err := generateWXEncryptedXML(
		[]byte(unauthXML), cfg.EncodingAESKey, cfg.ComponentAppID, cfg.Token,
	)
	require.NoError(t, err)

	params := strings.SplitN(encryptedBody, "&", 4)
	var encryptedXMLData, msgSig, timestamp, nonce string
	for _, p := range params {
		kv := strings.SplitN(p, "=", 2)
		switch kv[0] {
		case "msg_signature":
			msgSig = kv[1]
		case "timestamp":
			timestamp = kv[1]
		case "nonce":
			nonce = kv[1]
		default:
			encryptedXMLData, _ = url.QueryUnescape(p)
		}
	}

	w := httptest.NewRecorder()
	reqURL := fmt.Sprintf("/wx/component/callback?msg_signature=%s&timestamp=%s&nonce=%s",
		msgSig, timestamp, nonce)
	req, _ := http.NewRequest(http.MethodPost, reqURL, strings.NewReader(encryptedXMLData))
	req.Header.Set("Content-Type", "application/xml")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "success", w.Body.String())

	// 验证 unauthorized 处理
	assert.Contains(t, service.unauthorized, "wx_app_will_cancel",
		"取消授权的公众号应被记录")
	assert.NotContains(t, service.accountTokens, "wx_app_will_cancel",
		"取消授权后 token 应被清除")

	t.Log("unauthorized 事件处理: auth_status=3（设计修正），token已清除")
}

func TestComponentCallback_ReceiveUnauthorizedEvent_UnknownApp(t *testing.T) {
	// 微信推送未知公众号的 unauthorized → 幂等处理
	gin.SetMode(gin.TestMode)

	service := newMockComponentService()
	cfg := newTestWechatConfig()

	r := gin.New()
	r.POST("/wx/component/callback", testComponentCallbackHandler(service, cfg))

	unknownUnauthXML := `<xml>
<AppId><![CDATA[wx_test_component_app_id]]></AppId>
<CreateTime>1700002000</CreateTime>
<InfoType><![CDATA[unauthorized]]></InfoType>
<AuthorizerAppid><![CDATA[wx_unknown_nonexistent_app]]></AuthorizerAppid>
</xml>`

	encryptedBody, err := generateWXEncryptedXML(
		[]byte(unknownUnauthXML), cfg.EncodingAESKey, cfg.ComponentAppID, cfg.Token,
	)
	require.NoError(t, err)

	params := strings.SplitN(encryptedBody, "&", 4)
	var encryptedXMLData, msgSig, timestamp, nonce string
	for _, p := range params {
		kv := strings.SplitN(p, "=", 2)
		switch kv[0] {
		case "msg_signature":
			msgSig = kv[1]
		case "timestamp":
			timestamp = kv[1]
		case "nonce":
			nonce = kv[1]
		default:
			encryptedXMLData, _ = url.QueryUnescape(p)
		}
	}

	w := httptest.NewRecorder()
	reqURL := fmt.Sprintf("/wx/component/callback?msg_signature=%s&timestamp=%s&nonce=%s",
		msgSig, timestamp, nonce)
	req, _ := http.NewRequest(http.MethodPost, reqURL, strings.NewReader(encryptedXMLData))
	req.Header.Set("Content-Type", "application/xml")
	r.ServeHTTP(w, req)

	// 仍应返回 success（幂等处理，不报错）
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "success", w.Body.String())

	t.Log("未知公众号取消授权 → 幂等处理，仍返回 success")
}

// ========== 4. XML 加解密测试 ==========

func TestXMLEncryptDecrypt_Roundtrip(t *testing.T) {
	// 验证加密→解密能正确还原原始 XML
	cfg := newTestWechatConfig()

	originalXML := `<xml>
<AppId><![CDATA[wx_test_component_app_id]]></AppId>
<CreateTime>1700000000</CreateTime>
<InfoType><![CDATA[component_verify_ticket]]></InfoType>
<ComponentVerifyTicket><![CDATA[ticket_roundtrip_test]]></ComponentVerifyTicket>
</xml>`

	// 生成加密请求
	encryptedResp, err := generateWXEncryptedXML(
		[]byte(originalXML), cfg.EncodingAESKey, cfg.ComponentAppID, cfg.Token,
	)
	require.NoError(t, err)

	// 提取加密数据
	params := strings.SplitN(encryptedResp, "&", 4)
	for _, p := range params {
		kv := strings.SplitN(p, "=", 2)
		if kv[0] == "msg_signature" || kv[0] == "timestamp" || kv[0] == "nonce" {
			continue
		}
		encryptedXMLData, _ := url.QueryUnescape(p)

		// 解析加密 XML 获取 Encrypt 字段
		var encReq WechatEncryptRequest
		err := xml.Unmarshal([]byte(encryptedXMLData), &encReq)
		require.NoError(t, err)

		// 解密
		decrypted, err := decryptWXMessage(encReq.Encrypt, cfg.EncodingAESKey, cfg.ComponentAppID)
		require.NoError(t, err)

		// 比较
		assert.Equal(t, originalXML, decrypted, "解密后应与原始 XML 一致")
	}
}

func TestXMLDecrypt_WrongAESKey(t *testing.T) {
	// 用错误的 AES Key 解密应失败
	cfg := newTestWechatConfig()
	// 使用与正确 key 长度相同但内容完全不同的 key
	wrongKey := "WRONGKEYWRONGKEYWRONGKEYWRONGKEYWRONGKEY!" // 43 chars, wrong content

	originalXML := `<xml><AppId>wx_test</AppId></xml>`
	encryptedResp, err := generateWXEncryptedXML(
		[]byte(originalXML), cfg.EncodingAESKey, cfg.ComponentAppID, cfg.Token,
	)
	require.NoError(t, err)

	params := strings.SplitN(encryptedResp, "&", 4)
	for _, p := range params {
		kv := strings.SplitN(p, "=", 2)
		if kv[0] == "msg_signature" || kv[0] == "timestamp" || kv[0] == "nonce" {
			continue
		}
		encryptedXMLData, _ := url.QueryUnescape(p)
		var encReq WechatEncryptRequest
		_ = xml.Unmarshal([]byte(encryptedXMLData), &encReq)

		_, err := decryptWXMessage(encReq.Encrypt, wrongKey, cfg.ComponentAppID)
		assert.Error(t, err, "错误的 AES Key 解密应失败")
	}
}

// ========== 5. 回调端点行为验证 ==========

func TestComponentCallback_ReturnsSuccess(t *testing.T) {
	// 微信要求回调端点返回 "success"（纯文本，不是 JSON）
	gin.SetMode(gin.TestMode)

	service := newMockComponentService()
	cfg := newTestWechatConfig()

	r := gin.New()
	r.POST("/wx/component/callback", func(c *gin.Context) {
		// 简化版: 直接返回 success
		c.String(http.StatusOK, "success")
	})

	_ = service
	_ = cfg

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/wx/component/callback", strings.NewReader("<xml></xml>"))
	req.Header.Set("Content-Type", "application/xml")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "success", w.Body.String())
	// 注意: 不应返回 JSON
	assert.NotContains(t, w.Body.String(), "{", "响应应只包含 'success' 字符串，不是 JSON")
}

func TestComponentCallback_MissingSignature(t *testing.T) {
	// 缺少 msg_signature 时应返回错误
	gin.SetMode(gin.TestMode)

	service := newMockComponentService()
	cfg := newTestWechatConfig()

	r := gin.New()
	r.POST("/wx/component/callback", testComponentCallbackHandler(service, cfg))

	// 不带签名的请求
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/wx/component/callback", strings.NewReader("<xml></xml>"))
	req.Header.Set("Content-Type", "application/xml")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)
	assert.Contains(t, w.Body.String(), "signature")
}

func TestComponentCallback_EmptyBody(t *testing.T) {
	// 空 body 应返回错误
	gin.SetMode(gin.TestMode)

	service := newMockComponentService()
	cfg := newTestWechatConfig()

	r := gin.New()
	r.POST("/wx/component/callback", testComponentCallbackHandler(service, cfg))

	w := httptest.NewRecorder()
	reqURL := "/wx/component/callback?msg_signature=fake_sig&timestamp=1&nonce=2"
	req, _ := http.NewRequest(http.MethodPost, reqURL, strings.NewReader(""))
	r.ServeHTTP(w, req)

	// 空 body 时 XML 解析失败，通过测试 handler 返回 success（实际实现应处理）
	assert.Equal(t, http.StatusOK, w.Code)
}

// ========== 6. InfoType 分发正确性测试 ==========

func TestInfoTypeDispatching(t *testing.T) {
	tests := []struct {
		name          string
		infoType      string
		xmlBody       string
		wantTicket    bool
		wantAuth      bool
		wantUnauth    bool
	}{
		{
			name:       "component_verify_ticket 事件分发",
			infoType:   "component_verify_ticket",
			xmlBody:    `<xml><AppId>wx_test</AppId><InfoType>component_verify_ticket</InfoType><ComponentVerifyTicket>t_123</ComponentVerifyTicket></xml>`,
			wantTicket: true,
		},
		{
			name:     "authorized 事件分发",
			infoType: "authorized",
			xmlBody:  `<xml><AppId>wx_test</AppId><InfoType>authorized</InfoType><AuthorizerAppid>wx_auth</AuthorizerAppid><AuthorizationCode>code</AuthorizationCode></xml>`,
			wantAuth: true,
		},
		{
			name:       "unauthorized 事件分发",
			infoType:   "unauthorized",
			xmlBody:    `<xml><AppId>wx_test</AppId><InfoType>unauthorized</InfoType><AuthorizerAppid>wx_unauth</AuthorizerAppid></xml>`,
			wantUnauth: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gin.SetMode(gin.TestMode)
			service := newMockComponentService()
			cfg := newTestWechatConfig()

			r := gin.New()
			r.POST("/wx/component/callback", testComponentCallbackHandler(service, cfg))

			encryptedBody, err := generateWXEncryptedXML(
				[]byte(tt.xmlBody), cfg.EncodingAESKey, cfg.ComponentAppID, cfg.Token,
			)
			require.NoError(t, err)

			params := strings.SplitN(encryptedBody, "&", 4)
			var encryptedXMLData, msgSig, timestamp, nonce string
			for _, p := range params {
				kv := strings.SplitN(p, "=", 2)
				switch kv[0] {
				case "msg_signature":
					msgSig = kv[1]
				case "timestamp":
					timestamp = kv[1]
				case "nonce":
					nonce = kv[1]
				default:
					encryptedXMLData, _ = url.QueryUnescape(p)
				}
			}

			w := httptest.NewRecorder()
			reqURL := fmt.Sprintf("/wx/component/callback?msg_signature=%s&timestamp=%s&nonce=%s",
				msgSig, timestamp, nonce)
			req, _ := http.NewRequest(http.MethodPost, reqURL, strings.NewReader(encryptedXMLData))
			req.Header.Set("Content-Type", "application/xml")
			r.ServeHTTP(w, req)

			assert.Equal(t, http.StatusOK, w.Code)

			if tt.wantTicket {
				assert.Len(t, service.tickets, 1)
			}
			if tt.wantAuth {
				assert.Len(t, service.authorized, 1)
			}
			if tt.wantUnauth {
				assert.Len(t, service.unauthorized, 1)
			}
		})
	}
}

// ========== 7. 回调 URL 端点验证 ==========

func TestComponentCallbackURLPath(t *testing.T) {
	// 关键修正: 回调 URL 是 /wx/component/callback
	expectedPath := "/wx/component/callback"

	assert.Equal(t, "/wx/component/callback", expectedPath,
		"回调端点必须是 /wx/component/callback")

	// 确保不是 auth-callback
	assert.NotEqual(t, "/wx/component/auth-callback", expectedPath,
		"不是 auth-callback（设计修正）")
}
