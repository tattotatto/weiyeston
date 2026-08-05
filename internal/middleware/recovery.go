package middleware

import (
	"fmt"
	"net/http"
	"runtime/debug"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// Recovery Panic 恢复中间件
// 捕获 handler 中的 panic，记录日志并返回 500 响应
func Recovery(logger *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				// 记录堆栈信息到日志
				stack := debug.Stack()
				logger.Error("请求发生 panic",
					zap.Any("panic", err),
					zap.String("stack", string(stack)),
					zap.String("method", c.Request.Method),
					zap.String("path", c.Request.URL.Path),
				)

				// 检查连接是否已中断
				if c.Writer.Written() {
					c.Abort()
					return
				}

				// 返回 500 JSON 响应
				c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
					"code": 500,
					"msg":  fmt.Sprintf("服务器内部错误: %v", err),
				})
			}
		}()

		c.Next()
	}
}
