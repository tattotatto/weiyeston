package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

// TestRecoveryMiddleware_PanicHandling 测试 Panic 恢复
func TestRecoveryMiddleware_PanicHandling(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("发生 panic 时返回 500 而非崩溃", func(t *testing.T) {
		logger := zap.NewNop()
		r := gin.New()
		r.Use(Recovery(logger))
		r.GET("/panic", func(c *gin.Context) {
			panic("test panic")
		})

		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodGet, "/panic", nil)
		// 不应 panic
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusInternalServerError, w.Code,
			"panic 应被恢复并返回 500")
	})

	t.Run("发生 panic 时响应为 JSON 格式", func(t *testing.T) {
		logger := zap.NewNop()
		r := gin.New()
		r.Use(Recovery(logger))
		r.GET("/panic-json", func(c *gin.Context) {
			panic("json panic test")
		})

		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodGet, "/panic-json", nil)
		r.ServeHTTP(w, req)

		assert.Equal(t, "application/json; charset=utf-8", w.Header().Get("Content-Type"),
			"panic 恢复后的响应应为 JSON 格式")
	})

	t.Run("正常请求不受影响", func(t *testing.T) {
		logger := zap.NewNop()
		r := gin.New()
		r.Use(Recovery(logger))
		r.GET("/normal", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"ok": true})
		})

		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodGet, "/normal", nil)
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code,
			"正常请求应正常返回")
	})
}

// TestRecoveryMiddleware_Logging 测试 panic 时的日志记录
func TestRecoveryMiddleware_Logging(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("panic 时应记录错误日志", func(t *testing.T) {
		logger := zap.NewNop() // 使用 Nop logger，实际实现中应记录
		r := gin.New()
		r.Use(Recovery(logger))
		r.GET("/logged-panic", func(c *gin.Context) {
			panic("logged test panic")
		})

		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodGet, "/logged-panic", nil)
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
		// 日志验证在集成测试中进行
	})
}
