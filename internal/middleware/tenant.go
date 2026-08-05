package middleware

import (
	"strconv"

	"github.com/gin-gonic/gin"
)

// Tenant 租户上下文注入中间件
// 从认证信息中解析租户 ID，注入到 gin.Context
// 优先级：X-Tenant-ID header > context 中已有的 tenant_id > user_id
func Tenant() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 1. 检查 X-Tenant-ID header（最高优先级）
		if tenantHeader := c.GetHeader("X-Tenant-ID"); tenantHeader != "" {
			if tid, err := strconv.ParseInt(tenantHeader, 10, 64); err == nil {
				c.Set("tenant_id", tid)
				c.Next()
				return
			}
		}

		// 2. 如果上下文中已有 tenant_id（由 auth 中间件设置），直接使用
		if _, exists := c.Get("tenant_id"); exists {
			c.Next()
			return
		}

		// 3. 从 user_id 推导 tenant_id（默认租户与用户相同）
		if userID, exists := c.Get("user_id"); exists {
			switch v := userID.(type) {
			case int64:
				c.Set("tenant_id", v)
			case float64:
				c.Set("tenant_id", int64(v))
			default:
				c.Set("tenant_id", int64(0))
			}
		} else {
			// 无 user_id 时设为 0，不阻塞请求
			c.Set("tenant_id", int64(0))
		}

		c.Next()
	}
}
