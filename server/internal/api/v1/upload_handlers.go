package v1

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"video-sharing-system/server/internal/api"
	"video-sharing-system/server/internal/apperror"
	"video-sharing-system/server/internal/config"
	"video-sharing-system/server/internal/handlerutil"
	"video-sharing-system/server/internal/middleware"
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

// blobUploadEvent is the request body sent by @vercel/blob's browser `upload()`
// helper to its `handleUploadUrl` (this endpoint).
type blobUploadEvent struct {
	Type    string `json:"type"`
	Payload struct {
		Pathname      string `json:"pathname"`
		ClientPayload string `json:"clientPayload"`
		Multipart     bool   `json:"multipart"`
	} `json:"payload"`
}

// handleBlobUpload speaks the @vercel/blob client-upload protocol so the browser
// can upload video files straight to Vercel Blob (avoiding Vercel's request-body
// size limit on our API). It only mints upload tokens; the file never passes
// through this handler.
func handleBlobUpload(c *gin.Context, cfg config.Config, deps api.Deps) {
	var body blobUploadEvent
	if err := c.ShouldBindJSON(&body); err != nil {
		handlerutil.Error(c, http.StatusBadRequest, apperror.ValidationError, "入力内容を確認してください", nil)
		return
	}

	switch body.Type {
	case "blob.generate-client-token":
		if !authorizeBlobUpload(c, cfg, deps, body.Payload.ClientPayload) {
			handlerutil.Error(c, http.StatusUnauthorized, apperror.Unauthenticated, "ログインが必要です", nil)
			return
		}
		if !deps.Videos.BlobEnabled() {
			handlerutil.Error(c, http.StatusServiceUnavailable, apperror.InternalError, "ストレージが未設定です", nil)
			return
		}

		token, err := deps.Videos.IssueBlobUploadToken(body.Payload.Pathname)
		if err != nil {
			handlerutil.Error(c, http.StatusInternalServerError, apperror.InternalError, "サーバーで問題が発生しました", nil)
			return
		}
		// The Blob client expects this exact (unwrapped) shape.
		c.JSON(http.StatusOK, gin.H{
			"type":        body.Type,
			"clientToken": token,
		})

	case "blob.upload-completed":
		// We register uploads from the browser instead of relying on this
		// server-to-server callback, so simply acknowledge it.
		c.JSON(http.StatusOK, gin.H{"type": body.Type, "response": "ok"})

	default:
		handlerutil.Error(c, http.StatusBadRequest, apperror.ValidationError, "不明なイベントです", nil)
	}
}

// authorizeBlobUpload accepts either a valid session cookie (populated by the
// global Auth middleware) or a valid upload token passed as the clientPayload.
func authorizeBlobUpload(c *gin.Context, cfg config.Config, deps api.Deps, clientPayload string) bool {
	if _, ok := middleware.UserID(c); ok {
		return true
	}
	if clientPayload == "" {
		return false
	}
	_, _, err := deps.Auth.ResolveUploadToken(clientPayload, cfg.UploadTokenSecret)
	return err == nil
}
