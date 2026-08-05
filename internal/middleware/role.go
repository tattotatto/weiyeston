// Package middleware Gin 中间件集合
package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// RequireRole 角色权限校验中间件
// 使用方式: v1.GET("/admin/tenants", handler, middleware.RequireRole("admin"))
func RequireRole(roles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		role, exists := c.Get("role")
		if !exists {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"code": 403,
				"msg":  "权限不足",
			})
			return
		}

		roleStr, ok := role.(string)
		if !ok {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"code": 403,
				"msg":  "权限不足",
			})
			return
		}

		for _, allowed := range roles {
			if roleStr == allowed {
				c.Next()
				return
			}
		}

		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
			"code": 403,
			"msg":  "权限不足",
		})
	}
}
