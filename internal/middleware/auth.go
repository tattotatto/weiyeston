// Package middleware Gin 中间件集合
package middleware

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"

	"github.com/weiyeston/weiyeston-v2/internal/config"
)

// Auth JWT 认证中间件
// 从 Authorization header 提取 Bearer token，验证有效性
// 验证通过后将 user_id、tenant_id 注入 gin.Context
func Auth(jwtConfig config.JWTConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 提取 Authorization header
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"code": 401,
				"msg":  "未授权访问",
			})
			return
		}

		tokenString := strings.TrimPrefix(authHeader, "Bearer ")
		if tokenString == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"code": 401,
				"msg":  "未授权访问",
			})
			return
		}

		// 解析 token（仅允许 HS256 签名算法）
		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			// 验证签名算法必须为 HS256
			if token.Method.Alg() != jwt.SigningMethodHS256.Alg() {
				return nil, jwt.ErrSignatureInvalid
			}
			return []byte(jwtConfig.Secret), nil
		})

		if err != nil {
			msg := "token 无效或已过期"
			if errors.Is(err, jwt.ErrTokenExpired) {
				msg = "token 已过期"
			}
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"code": 401,
				"msg":  msg,
			})
			return
		}

		if !token.Valid {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"code": 401,
				"msg":  "token 无效或已过期",
			})
			return
		}

		// 提取 claims
		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"code": 401,
				"msg":  "token 无效或已过期",
			})
			return
		}

		// 提取 user_id（从 sub 字段）
		sub, exists := claims["sub"]
		if !exists {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"code": 401,
				"msg":  "token 无效或已过期",
			})
			return
		}

		// sub 在 JSON 反序列化后可能是 float64，需要转换为 int64
		var userID int64
		switch v := sub.(type) {
		case float64:
			userID = int64(v)
		case int64:
			userID = v
		default:
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"code": 401,
				"msg":  "token 无效或已过期",
			})
			return
		}

		// 注入 user_id 到上下文
		c.Set("user_id", userID)

		// 提取 tenant_id（从 claims 的 tenant_id 字段，容错：不存在时设为 0）
		if tid, ok := claims["tenant_id"]; ok {
			switch v := tid.(type) {
			case float64:
				c.Set("tenant_id", int64(v))
			case int64:
				c.Set("tenant_id", v)
			default:
				c.Set("tenant_id", int64(0))
			}
		} else {
			// 容错：token 无 tenant_id 时设为 0
			c.Set("tenant_id", int64(0))
		}

		// 提取 role（T2 新增，容错：不存在时默认 "user"）
		if role, ok := claims["role"]; ok {
			if roleStr, ok := role.(string); ok {
				c.Set("role", roleStr)
			} else {
				c.Set("role", "user")
			}
		} else {
			c.Set("role", "user")
		}

		c.Next()
	}
}
