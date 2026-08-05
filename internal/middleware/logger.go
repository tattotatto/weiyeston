package middleware

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

// Logger 请求日志中间件
// 记录每个 HTTP 请求的方法、路径、状态码、耗时、客户端 IP 等信息
func Logger(logger *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 生成 request_id 并注入上下文
		requestID := c.GetHeader("X-Request-ID")
		if requestID == "" {
			requestID = uuid.New().String()
		}
		c.Set("request_id", requestID)
		c.Header("X-Request-ID", requestID)

		start := time.Now()
		path := c.Request.URL.Path
		method := c.Request.Method

		// 处理请求
		c.Next()

		// 计算耗时
		latency := time.Since(start)
		status := c.Writer.Status()

		// 提取客户端 IP
		clientIP := c.ClientIP()

		// 提取 User-Agent
		userAgent := c.Request.UserAgent()

		// 构建日志字段
		fields := []zap.Field{
			zap.String("method", method),
			zap.String("path", path),
			zap.Int("status", status),
			zap.Float64("latency", float64(latency.Microseconds())/1000.0),
			zap.String("client_ip", clientIP),
			zap.String("user_agent", userAgent),
			zap.String("request_id", requestID),
		}

		// 根据状态码选择日志级别
		switch {
		case status >= 500:
			logger.Error("请求处理完成", fields...)
		case status >= 400:
			logger.Warn("请求处理完成", fields...)
		default:
			logger.Info("请求处理完成", fields...)
		}
	}
}
