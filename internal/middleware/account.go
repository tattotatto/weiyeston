// Package middleware Gin 中间件集合
package middleware

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/weiyeston/weiyeston-v2/internal/repository/account"
)

// CheckAccountOwnership 公众号归属校验中间件
// 从路由参数 :id 获取公众号 ID，从 gin.Context 获取 tenant_id
// 查询数据库验证该公众号是否属于当前租户
// 不通过时 AbortWithStatusJSON 并阻止后续 handler 执行
func CheckAccountOwnership(accountRepo *account.Repo) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 提取路由参数 :id
		accountID, err := strconv.ParseInt(c.Param("id"), 10, 64)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
				"code": 40001,
				"msg":  "无效的公众号 ID",
				"data": nil,
			})
			return
		}

		// 获取当前租户 ID
		tenantIDVal, exists := c.Get("tenant_id")
		if !exists {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"code": 401,
				"msg":  "未授权访问",
				"data": nil,
			})
			return
		}

		var tenantID int64
		switch v := tenantIDVal.(type) {
		case int64:
			tenantID = v
		case float64:
			tenantID = int64(v)
		default:
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"code": 401,
				"msg":  "未授权访问",
				"data": nil,
			})
			return
		}

		// 查询公众号
		acc, err := accountRepo.GetByID(c.Request.Context(), accountID)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
				"code": 50001,
				"msg":  "服务器内部错误",
				"data": nil,
			})
			return
		}
		if acc == nil {
			c.AbortWithStatusJSON(http.StatusNotFound, gin.H{
				"code": 40401,
				"msg":  "公众号不存在",
				"data": nil,
			})
			return
		}

		// 校验归属
		if acc.TenantID != tenantID {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"code": 40301,
				"msg":  "无权限操作该公众号",
				"data": nil,
			})
			return
		}

		c.Next()
	}
}
