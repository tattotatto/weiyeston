// Package model 微信开放平台推送事件数据结构
package model

import "encoding/xml"

// ComponentEvent 基础事件（公共字段）
type ComponentEvent struct {
	AppId      string `xml:"AppId"      json:"app_id"`
	CreateTime int64  `xml:"CreateTime" json:"create_time"`
	InfoType   string `xml:"InfoType"   json:"info_type"`
}

// ComponentVerifyTicketEvent 推送 component_verify_ticket
type ComponentVerifyTicketEvent struct {
	XMLName                xml.Name `xml:"xml"`
	AppId                  string   `xml:"AppId"`
	CreateTime             int64    `xml:"CreateTime"`
	InfoType               string   `xml:"InfoType"`
	ComponentVerifyTicket  string   `xml:"ComponentVerifyTicket"`
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
