package middleware

import (
	"video-sharing-system/server/internal/service"

	"github.com/gin-gonic/gin"
)

const (
	ContextUserIDKey       = "auth_user_id"
	ContextUserPublicIDKey = "auth_user_public_id"
	ContextUserRoleKey     = "auth_user_role"
	ContextSessionIDKey    = "auth_session_db_id"
)

// SessionCookieName matches api-design.md.
const SessionCookieName = "session_id"

// Auth loads session from cookie and attaches user context when valid.
func Auth(authSvc *service.AuthService) gin.HandlerFunc {
	return func(c *gin.Context) {
		token, err := c.Cookie(SessionCookieName)
		if err != nil || token == "" {
			c.Next()
			return
		}
		sess, err := authSvc.ResolveSession(c.Request.Context(), token)
		if err != nil || sess == nil {
			c.Next()
			return
		}
		c.Set(ContextUserIDKey, sess.UserID)
		c.Set(ContextUserPublicIDKey, sess.PublicID.String())
		c.Set(ContextUserRoleKey, string(sess.Role))
		c.Set(ContextSessionIDKey, sess.SessionID)
		c.Next()
	}
}

func userIDFromCtx(c *gin.Context) (int64, bool) {
	v, ok := c.Get(ContextUserIDKey)
	if !ok {
		return 0, false
	}
	id, ok := v.(int64)
	return id, ok
}

// UserID returns authenticated internal user id when present.
func UserID(c *gin.Context) (int64, bool) {
	return userIDFromCtx(c)
}

// UserPublicID returns authenticated user's public UUID string when present.
func UserPublicID(c *gin.Context) (string, bool) {
	v, ok := c.Get(ContextUserPublicIDKey)
	if !ok {
		return "", false
	}
	s, ok := v.(string)
	return s, ok
}

// UserRole returns authenticated role string when present.
func UserRole(c *gin.Context) (string, bool) {
	v, ok := c.Get(ContextUserRoleKey)
	if !ok {
		return "", false
	}
	s, ok := v.(string)
	return s, ok
}
