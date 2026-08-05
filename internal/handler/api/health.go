// Package api HTTP handler — 管理后台 API 接口
package api

import (
	"context"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"
	"github.com/redis/go-redis/v9"
)

// startTime 记录服务启动时间，用于计算 uptime
var startTime = time.Now()

// HealthHandler 健康检查处理器
type HealthHandler struct {
	db    *sqlx.DB
	redis *redis.Client
}

// NewHealthHandler 创建健康检查处理器
func NewHealthHandler(db *sqlx.DB, rdb *redis.Client) *HealthHandler {
	return &HealthHandler{db: db, redis: rdb}
}

// CheckResult 单个检查项的结果
type CheckResult struct {
	Status    string  `json:"status"` // "ok" | "error"
	LatencyMs float64 `json:"latency_ms,omitempty"`
	Message   string  `json:"message,omitempty"`
}

// HealthResponse 健康检查响应体
type HealthResponse struct {
	Status        string                 `json:"status"` // "healthy" | "degraded"
	Version       string                 `json:"version"`
	UptimeSeconds int64                  `json:"uptime_seconds"`
	Checks        map[string]CheckResult `json:"checks"`
	Timestamp     string                 `json:"timestamp"`
}

// Check 健康检查端点 GET /api/v1/health
// 并发检查 DB (Ping) + Redis (Ping)
// 返回 JSON：status, version, uptime_seconds, checks, timestamp
// 任何组件不健康返回 503
func (h *HealthHandler) Check(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 3*time.Second)
	defer cancel()

	var mu sync.Mutex
	checks := make(map[string]CheckResult)
	overall := "healthy"

	var wg sync.WaitGroup
	wg.Add(2)

	// 并发检查数据库
	go func() {
		defer wg.Done()
		result := CheckResult{Status: "ok"}
		start := time.Now()

		if h.db != nil {
			err := h.db.PingContext(ctx)
			latency := float64(time.Since(start).Microseconds()) / 1000.0
			result.LatencyMs = latency
			if err != nil {
				result = CheckResult{Status: "error", Message: err.Error()}
				mu.Lock()
				overall = "degraded"
				mu.Unlock()
			}
		}

		mu.Lock()
		checks["database"] = result
		mu.Unlock()
	}()

	// 并发检查 Redis
	go func() {
		defer wg.Done()
		result := CheckResult{Status: "ok"}
		start := time.Now()

		if h.redis != nil {
			err := h.redis.Ping(ctx).Err()
			latency := float64(time.Since(start).Microseconds()) / 1000.0
			result.LatencyMs = latency
			if err != nil {
				result = CheckResult{Status: "error", Message: err.Error()}
				mu.Lock()
				overall = "degraded"
				mu.Unlock()
			}
		}

		mu.Lock()
		checks["redis"] = result
		mu.Unlock()
	}()

	wg.Wait()

	httpStatus := http.StatusOK
	if overall != "healthy" {
		httpStatus = http.StatusServiceUnavailable
	}

	c.JSON(httpStatus, HealthResponse{
		Status:        overall,
		Version:       "0.1.0",
		UptimeSeconds: int64(time.Since(startTime).Seconds()),
		Checks:        checks,
		Timestamp:     time.Now().UTC().Format(time.RFC3339),
	})
}
