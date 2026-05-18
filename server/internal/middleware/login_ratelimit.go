package middleware

import (
	"net/http"

	"video-sharing-system/server/internal/apperror"
	"video-sharing-system/server/internal/ratelimit"

	"github.com/gin-gonic/gin"
)

// LoginRateLimit limits POST /auth/login per client IP.
func LoginRateLimit(l *ratelimit.WindowLimiter) gin.HandlerFunc {
	return func(c *gin.Context) {
		ip := c.ClientIP()
		if !l.Allow("login:"+ip) {
			writeAPIError(c, http.StatusTooManyRequests, apperror.RateLimited, "しばらく待ってから再度お試しください")
			c.Abort()
			return
		}
		c.Next()
	}
}
