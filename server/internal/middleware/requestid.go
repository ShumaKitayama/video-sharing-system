package middleware

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

const RequestIDHeader = "X-Request-ID"
const RequestIDContextKey = "request_id"

// RequestID ensures every request has X-Request-ID (generate if missing).
func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		rid := c.GetHeader(RequestIDHeader)
		if rid == "" {
			rid = "req_" + uuid.NewString()
		}
		c.Writer.Header().Set(RequestIDHeader, rid)
		c.Set(RequestIDContextKey, rid)
		c.Next()
	}
}

func GetRequestID(c *gin.Context) string {
	if v, ok := c.Get(RequestIDContextKey); ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}
