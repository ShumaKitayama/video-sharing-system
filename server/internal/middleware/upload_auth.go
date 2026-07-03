package middleware

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"video-sharing-system/server/internal/apperror"
	"video-sharing-system/server/internal/service"
)

// RequireAuthOrUploadToken accepts a normal session cookie or X-Upload-Token for large uploads.
func RequireAuthOrUploadToken(secret string, auth *service.AuthService) gin.HandlerFunc {
	return func(c *gin.Context) {
		if _, ok := userIDFromCtx(c); ok {
			c.Next()
			return
		}

		token := c.GetHeader("X-Upload-Token")
		if token == "" {
			writeAPIError(c, http.StatusUnauthorized, apperror.Unauthenticated, "ログインが必要です")
			c.Abort()
			return
		}

		uid, role, err := auth.ResolveUploadToken(token, secret)
		if err != nil {
			if errors.Is(err, service.ErrInvalidUploadToken) {
				writeAPIError(c, http.StatusUnauthorized, apperror.Unauthenticated, "アップロードトークンが無効です")
				c.Abort()
				return
			}
			writeAPIError(c, http.StatusInternalServerError, apperror.InternalError, "サーバーで問題が発生しました")
			c.Abort()
			return
		}

		c.Set(ContextUserIDKey, uid)
		c.Set(ContextUserRoleKey, role)
		c.Next()
	}
}
