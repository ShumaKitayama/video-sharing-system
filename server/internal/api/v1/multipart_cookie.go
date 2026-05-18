package v1

import (
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"video-sharing-system/server/internal/config"
	"video-sharing-system/server/internal/middleware"
)

func parseVideoUpload(c *gin.Context) (title string, description string, mime string, filename string, file io.ReadCloser, err error) {
	// Parse multipart via net/http so field order does not discard the file body.
	// (Calling MultipartReader.NextPart() after selecting "file" drains the file before it can be read.)
	const maxMemory = 32 << 20
	if err := c.Request.ParseMultipartForm(maxMemory); err != nil {
		return "", "", "", "", nil, err
	}
	title = strings.TrimSpace(c.PostForm("title"))
	description = c.PostForm("description")
	fh, err := c.FormFile("file")
	if err != nil {
		return "", "", "", "", nil, fmt.Errorf("missing file")
	}
	rc, err := fh.Open()
	if err != nil {
		return "", "", "", "", nil, err
	}
	mime = fh.Header.Get("Content-Type")
	filename = fh.Filename
	return title, description, mime, filename, rc, nil
}

func setSessionCookie(c *gin.Context, cfg config.Config, token string) {
	ss := http.SameSiteLaxMode
	switch strings.ToLower(cfg.CookieSameSite) {
	case "strict":
		ss = http.SameSiteStrictMode
	case "none":
		ss = http.SameSiteNoneMode
	}

	c.SetSameSite(ss)
	c.SetCookie(
		middleware.SessionCookieName,
		token,
		int((24 * time.Hour).Seconds()),
		"/",
		"",
		cfg.CookieSecure,
		true,
	)
}

func clearSessionCookie(c *gin.Context, cfg config.Config) {
	ss := http.SameSiteLaxMode
	switch strings.ToLower(cfg.CookieSameSite) {
	case "strict":
		ss = http.SameSiteStrictMode
	case "none":
		ss = http.SameSiteNoneMode
	}
	c.SetSameSite(ss)
	c.SetCookie(
		middleware.SessionCookieName,
		"",
		-1,
		"/",
		"",
		cfg.CookieSecure,
		true,
	)
}
