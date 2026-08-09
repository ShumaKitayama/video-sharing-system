package v1

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"video-sharing-system/server/internal/api"
	"video-sharing-system/server/internal/apperror"
	"video-sharing-system/server/internal/config"
	"video-sharing-system/server/internal/handlerutil"
	"video-sharing-system/server/internal/middleware"
	"video-sharing-system/server/internal/validation"
)

// handleUploadTokenIssue returns a short-lived token used to authorize the
// browser's follow-up calls (direct Blob upload + DB registration).
func handleUploadTokenIssue(c *gin.Context, cfg config.Config, deps api.Deps) {
	uid, ok := middleware.UserID(c)
	if !ok {
		handlerutil.Error(c, http.StatusUnauthorized, apperror.Unauthenticated, "ログインが必要です", nil)
		return
	}
	role, ok := middleware.UserRole(c)
	if !ok {
		handlerutil.Error(c, http.StatusUnauthorized, apperror.Unauthenticated, "ログインが必要です", nil)
		return
	}

	token, err := deps.Auth.IssueUploadToken(uid, role, cfg.UploadTokenSecret)
	if err != nil {
		handlerutil.Error(c, http.StatusInternalServerError, apperror.InternalError, "サーバーで問題が発生しました", nil)
		return
	}

	handlerutil.Data(c, http.StatusOK, gin.H{
		"token": token,
	})
}

// handleDirectUpload hands the browser a single pre-authorized URL it can PUT
// the video file to. The file goes straight to object storage, which keeps it
// clear of the roughly 4.5MB request-body limit this API runs under on Vercel.
// No storage credentials are ever exposed: the URL works once, for one object
// key, for one content type, and only until it expires.
func handleDirectUpload(c *gin.Context, deps api.Deps) {
	var req struct {
		ContentType string `json:"content_type"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		handlerutil.Error(c, http.StatusBadRequest, apperror.ValidationError, "入力内容を確認してください", []apperror.FieldDetail{
			{Field: "body", Message: "JSON形式で送信してください"},
		})
		return
	}

	if !validation.IsAllowedVideoMIME(req.ContentType) {
		handlerutil.Error(c, http.StatusUnsupportedMediaType, apperror.UnsupportedMediaType, "MP4またはWebM形式の動画を選んでください", nil)
		return
	}
	if !deps.Videos.RemoteStorageEnabled() {
		handlerutil.Error(c, http.StatusServiceUnavailable, apperror.InternalError, "動画の保存先が設定されていません", nil)
		return
	}

	slot, err := deps.Videos.IssueDirectUpload(req.ContentType)
	if err != nil {
		handlerutil.Error(c, http.StatusInternalServerError, apperror.InternalError, "サーバーで問題が発生しました", nil)
		return
	}

	handlerutil.Data(c, http.StatusOK, gin.H{
		"upload_url":   slot.UploadURL,
		"public_url":   slot.PublicURL,
		"content_type": slot.ContentType,
		"expires_in":   slot.ExpiresInSeconds,
	})
}
