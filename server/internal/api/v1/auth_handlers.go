package v1

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"video-sharing-system/server/internal/api"
	"video-sharing-system/server/internal/apperror"
	"video-sharing-system/server/internal/config"
	"video-sharing-system/server/internal/handlerutil"
	"video-sharing-system/server/internal/middleware"
	"video-sharing-system/server/internal/service"
	"video-sharing-system/server/internal/validation"
)

func handleAuthLogin(c *gin.Context, cfg config.Config, deps api.Deps) {
	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		handlerutil.Error(c, http.StatusBadRequest, apperror.ValidationError, "入力内容を確認してください", []apperror.FieldDetail{
			{Field: "body", Message: "JSON形式で送信してください"},
		})
		return
	}

	var details []apperror.FieldDetail
	if d := validation.Username(strings.TrimSpace(req.Username)); d != nil {
		details = append(details, *d)
	}
	if d := validation.Password(req.Password); d != nil {
		details = append(details, *d)
	}
	if len(details) > 0 {
		handlerutil.Error(c, http.StatusBadRequest, apperror.ValidationError, "入力内容を確認してください", details)
		return
	}

	token, user, err := deps.Auth.Login(c.Request.Context(), strings.TrimSpace(req.Username), req.Password)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrInvalidCredentials):
			handlerutil.Error(c, http.StatusUnauthorized, apperror.Unauthenticated, "ユーザー名またはパスワードが正しくありません", nil)
			return
		case errors.Is(err, service.ErrInactiveUser):
			handlerutil.Error(c, http.StatusForbidden, apperror.Forbidden, "このアカウントは利用できません", nil)
			return
		default:
			handlerutil.Error(c, http.StatusInternalServerError, apperror.InternalError, "サーバーで問題が発生しました", nil)
			return
		}
	}

	setSessionCookie(c, cfg, token)
	handlerutil.Data(c, http.StatusOK, gin.H{"user": user})
}

func handleAuthLogout(c *gin.Context, cfg config.Config, deps api.Deps) {
	token, err := c.Cookie(middleware.SessionCookieName)
	if err != nil {
		token = ""
	}
	_ = deps.Auth.Logout(c.Request.Context(), token)
	clearSessionCookie(c, cfg)
	c.Status(http.StatusNoContent)
}

func handleAuthMe(c *gin.Context, deps api.Deps) {
	pidStr, ok := middleware.UserPublicID(c)
	if !ok {
		handlerutil.Error(c, http.StatusUnauthorized, apperror.Unauthenticated, "ログインが必要です", nil)
		return
	}
	id, err := uuid.Parse(pidStr)
	if err != nil {
		handlerutil.Error(c, http.StatusUnauthorized, apperror.Unauthenticated, "ログインが必要です", nil)
		return
	}
	u, err := deps.Auth.GetAuthUser(c.Request.Context(), id)
	if err != nil {
		handlerutil.Error(c, http.StatusInternalServerError, apperror.InternalError, "サーバーで問題が発生しました", nil)
		return
	}
	if u == nil {
		handlerutil.Error(c, http.StatusUnauthorized, apperror.Unauthenticated, "ログインが必要です", nil)
		return
	}
	handlerutil.Data(c, http.StatusOK, u)
}
