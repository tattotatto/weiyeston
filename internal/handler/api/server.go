// Package api 服务器信息 Handler
package api

import (
	"io"
	"net"
	"net/http"
	"os"
	"runtime"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// ServerHandler 服务器信息处理器
type ServerHandler struct {
	startTime time.Time
}

// NewServerHandler 创建服务器信息 Handler 实例
func NewServerHandler() *ServerHandler {
	return &ServerHandler{
		startTime: time.Now(),
	}
}

// ServerInfoResponse 服务器信息响应
type ServerInfoResponse struct {
	PublicIP     string `json:"public_ip"`
	LocalIP      string `json:"local_ip"`
	Version      string `json:"version"`
	GoVersion    string `json:"go_version"`
	OS           string `json:"os"`
	Arch         string `json:"arch"`
	Hostname     string `json:"hostname"`
	Uptime       string `json:"uptime"`
	NumCPU       int    `json:"num_cpu"`
	NumGoroutine int    `json:"num_goroutine"`
}

// GetInfo GET /api/v1/server/info — 获取服务器基本信息
func (h *ServerHandler) GetInfo(c *gin.Context) {
	hostname, _ := os.Hostname()
	publicIP := getPublicIP()
	localIP := getLocalIP()
	uptime := time.Since(h.startTime).Truncate(time.Second).String()

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"msg":  "ok",
		"data": ServerInfoResponse{
			PublicIP:     publicIP,
			LocalIP:      localIP,
			Version:      "2.0.0",
			GoVersion:    runtime.Version(),
			OS:           runtime.GOOS,
			Arch:         runtime.GOARCH,
			Hostname:     hostname,
			Uptime:       uptime,
			NumCPU:       runtime.NumCPU(),
			NumGoroutine: runtime.NumGoroutine(),
		},
	})
}

// getPublicIP tries to detect the public IP by calling external services
func getPublicIP() string {
	services := []string{
		"https://api.ipify.org",
		"https://ifconfig.me/ip",
		"https://icanhazip.com",
	}
	client := &http.Client{Timeout: 3 * time.Second}
	for _, url := range services {
		resp, err := client.Get(url)
		if err != nil {
			continue
		}
		body, err := io.ReadAll(io.LimitReader(resp.Body, 64))
		resp.Body.Close()
		if err != nil {
			continue
		}
		ip := strings.TrimSpace(string(body))
		if ip != "" {
			return ip
		}
	}
	return "unknown"
}

// getLocalIP returns the first non-loopback local IPv4 address
func getLocalIP() string {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return "unknown"
	}
	for _, addr := range addrs {
		if ipNet, ok := addr.(*net.IPNet); ok && !ipNet.IP.IsLoopback() {
			if ipNet.IP.To4() != nil {
				return ipNet.IP.String()
			}
		}
	}
	return "unknown"
}
