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
	mr, err := c.Request.MultipartReader()
	if err != nil {
		return "", "", "", "", nil, err
	}

	var fileRC io.ReadCloser

	for {
		part, err := mr.NextPart()
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", "", "", "", nil, err
		}

		switch part.FormName() {
		case "title":
			b, rerr := io.ReadAll(io.LimitReader(part, 256))
			_ = part.Close()
			if rerr != nil {
				return "", "", "", "", nil, rerr
			}
			title = strings.TrimSpace(string(b))
		case "description":
			b, rerr := io.ReadAll(io.LimitReader(part, 4096))
			_ = part.Close()
			if rerr != nil {
				return "", "", "", "", nil, rerr
			}
			description = string(b)
		case "file":
			mime = part.Header.Get("Content-Type")
			filename = part.FileName()
			fileRC = part
		default:
			_, _ = io.Copy(io.Discard, part)
			_ = part.Close()
		}
	}

	if fileRC == nil {
		return "", "", "", "", nil, fmt.Errorf("missing file")
	}

	return title, description, mime, filename, fileRC, nil
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
