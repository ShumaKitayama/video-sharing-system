package middleware

import (
	"net/http"

	"video-sharing-system/server/internal/apperror"

	"github.com/gin-gonic/gin"
)

func writeAPIError(c *gin.Context, status int, code apperror.Code, message string) {
	c.JSON(status, gin.H{
		"error": gin.H{
			"code":       code,
			"message":    message,
			"request_id": c.Writer.Header().Get("X-Request-ID"),
		},
	})
}

// RequireAuth rejects requests without a valid session.
func RequireAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		if _, ok := userIDFromCtx(c); !ok {
			writeAPIError(c, http.StatusUnauthorized, apperror.Unauthenticated, "ログインが必要です")
			c.Abort()
			return
		}
		c.Next()
	}
}

// RequireTeacher ensures role teacher.
func RequireTeacher() gin.HandlerFunc {
	return func(c *gin.Context) {
		if _, ok := userIDFromCtx(c); !ok {
			writeAPIError(c, http.StatusUnauthorized, apperror.Unauthenticated, "ログインが必要です")
			c.Abort()
			return
		}
		role, _ := c.Get(ContextUserRoleKey)
		if role != "teacher" {
			writeAPIError(c, http.StatusForbidden, apperror.Forbidden, "権限がありません")
			c.Abort()
			return
		}
		c.Next()
	}
}
