package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/weiyeston/weiyeston-v2/internal/config"
)

// CORS 跨域资源共享中间件
// 处理 OPTIONS 预检请求和实际请求的 CORS 响应头
func CORS(corsConfig config.CORSConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")

		// 无 Origin header 的请求直接放行
		if origin == "" {
			c.Next()
			return
		}

		// 检查 Origin 是否在允许列表中
		allowed := false
		for _, o := range corsConfig.AllowedOrigins {
			if o == "*" || o == origin {
				allowed = true
				break
			}
		}

		if allowed {
			c.Header("Access-Control-Allow-Origin", origin)
			c.Header("Access-Control-Allow-Credentials", "true")
		}

		// 设置允许的方法和头
		if len(corsConfig.AllowedMethods) > 0 {
			c.Header("Access-Control-Allow-Methods", strings.Join(corsConfig.AllowedMethods, ", "))
		}
		if len(corsConfig.AllowedHeaders) > 0 {
			c.Header("Access-Control-Allow-Headers", strings.Join(corsConfig.AllowedHeaders, ", "))
		}

		// 暴露的响应头
		c.Header("Access-Control-Expose-Headers", "Content-Length, Content-Type")

		// OPTIONS 预检请求处理
		if c.Request.Method == http.MethodOptions {
			c.Header("Access-Control-Max-Age", "86400") // 24 小时
			if !allowed {
				c.AbortWithStatus(http.StatusNoContent)
				return
			}
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}
