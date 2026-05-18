package handlerutil

import (
	"net/http"
	"net/url"

	"video-sharing-system/server/internal/apperror"

	"github.com/gin-gonic/gin"
)

type errorEnvelope struct {
	Error apperror.APIError `json:"error"`
}

type dataEnvelope struct {
	Data any `json:"data"`
}

type listEnvelope struct {
	Data any      `json:"data"`
	Meta Meta     `json:"meta"`
}

// Meta matches api-design pagination meta.
type Meta struct {
	Page       int   `json:"page"`
	PerPage    int   `json:"per_page"`
	Total      int64 `json:"total"`
	TotalPages int   `json:"total_pages"`
}

func requestID(c *gin.Context) string {
	return c.Writer.Header().Get("X-Request-ID")
}

// Error writes the standard API error JSON body.
func Error(c *gin.Context, status int, code apperror.Code, message string, details []apperror.FieldDetail) {
	c.JSON(status, errorEnvelope{
		Error: apperror.APIError{
			Code:      code,
			Message:   message,
			Details:   details,
			RequestID: requestID(c),
		},
	})
}

// ErrorFromAPIError writes error using a fully-built APIError (status separate).
func ErrorFromAPIError(c *gin.Context, status int, apiErr apperror.APIError) {
	apiErr.RequestID = requestID(c)
	c.JSON(status, errorEnvelope{Error: apiErr})
}

// Data writes {"data": payload}.
func Data(c *gin.Context, status int, payload any) {
	c.JSON(status, dataEnvelope{Data: payload})
}

// List writes paginated list response.
func List(c *gin.Context, payload any, meta Meta) {
	c.JSON(http.StatusOK, listEnvelope{Data: payload, Meta: meta})
}

// RedirectTemporary sends 307 with Location header (unused but kept for completeness).
func RedirectTemporary(c *gin.Context, loc string) {
	c.Redirect(http.StatusTemporaryRedirect, loc)
}

// ParsePositiveInt parses query param with default.
func ParsePositiveInt(raw string, def int, max int) int {
	if raw == "" {
		return def
	}
	v, err := url.QueryUnescape(raw)
	if err != nil {
		return def
	}
	n := 0
	for _, ch := range v {
		if ch < '0' || ch > '9' {
			return def
		}
		n = n*10 + int(ch-'0')
		if n > max {
			return max
		}
	}
	if n <= 0 {
		return def
	}
	return n
}

// TotalPages computes pages for pagination meta.
func TotalPages(total int64, perPage int) int {
	if total == 0 {
		return 0
	}
	return int((total + int64(perPage) - 1) / int64(perPage))
}
