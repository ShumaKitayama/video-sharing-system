package v1

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"video-sharing-system/server/internal/api"
	"video-sharing-system/server/internal/apperror"
	"video-sharing-system/server/internal/handlerutil"
	"video-sharing-system/server/internal/middleware"
	"video-sharing-system/server/internal/service"
	"video-sharing-system/server/internal/validation"
)

func handleVideosList(c *gin.Context, deps api.Deps) {
	page := handlerutil.ParsePositiveInt(c.Query("page"), 1, 1_000_000)
	perPage := handlerutil.ParsePositiveInt(c.Query("per_page"), 20, 50)
	sort := validation.ParseVideoSort(c.Query("sort"))

	var qPtr *string
	if q := strings.TrimSpace(c.Query("q")); q != "" {
		qPtr = &q
	}

	var uploaderPtr *uuid.UUID
	if raw := strings.TrimSpace(c.Query("uploader_id")); raw != "" {
		id, err := uuid.Parse(raw)
		if err != nil {
			handlerutil.Error(c, http.StatusBadRequest, apperror.ValidationError, "入力内容を確認してください", []apperror.FieldDetail{
				{Field: "uploader_id", Message: "UUID形式で指定してください"},
			})
			return
		}
		uploaderPtr = &id
	}

	statusRaw := strings.TrimSpace(c.Query("status"))
	role, roleOK := middleware.UserRole(c)
	_, logged := middleware.UserID(c)
	isTeacher := logged && roleOK && role == "teacher"
	if statusRaw != "" && !isTeacher {
		handlerutil.Error(c, http.StatusForbidden, apperror.Forbidden, "権限がありません", nil)
		return
	}

	filterPtr, fieldErrs := validation.ParseVideoListStatusFilter(statusRaw, isTeacher)
	if len(fieldErrs) > 0 {
		handlerutil.Error(c, http.StatusBadRequest, apperror.ValidationError, "入力内容を確認してください", fieldErrs)
		return
	}

	var teacherStatus *string
	if isTeacher {
		if filterPtr != nil && *filterPtr != "all" {
			teacherStatus = filterPtr
		}
	}

	items, total, err := deps.Videos.ListVideos(c.Request.Context(), role, teacherStatus, qPtr, uploaderPtr, sort, page, perPage)
	if err != nil {
		handlerutil.Error(c, http.StatusInternalServerError, apperror.InternalError, "サーバーで問題が発生しました", nil)
		return
	}

	handlerutil.List(c, items, handlerutil.Meta{
		Page:       page,
		PerPage:    perPage,
		Total:      total,
		TotalPages: handlerutil.TotalPages(total, perPage),
	})
}

func handleVideosCreate(c *gin.Context, deps api.Deps) {
	uid, ok := middleware.UserID(c)
	if !ok {
		handlerutil.Error(c, http.StatusUnauthorized, apperror.Unauthenticated, "ログインが必要です", nil)
		return
	}

	if !deps.UploadRL.Allow(fmt.Sprintf("upload:%d", uid)) {
		handlerutil.Error(c, http.StatusTooManyRequests, apperror.RateLimited, "アップロードはしばらく待ってから試してください", nil)
		return
	}

	title, desc, mime, filename, rc, err := parseVideoUpload(c)
	if rc != nil {
		defer rc.Close()
	}
	if err != nil {
		handlerutil.Error(c, http.StatusBadRequest, apperror.ValidationError, "入力内容を確認してください", []apperror.FieldDetail{
			{Field: "file", Message: "multipart/form-data で file/title/description を送信してください"},
		})
		return
	}

	var details []apperror.FieldDetail
	if d := validation.Title(title); d != nil {
		details = append(details, *d)
	}
	if d := validation.Description(desc); d != nil {
		details = append(details, *d)
	}
	if !validation.IsAllowedVideoMIME(mime) || !validation.FilenameMatchesMIME(filename, mime) {
		details = append(details, apperror.FieldDetail{Field: "file", Message: "MP4またはWebMを指定してください"})
	}
	if len(details) > 0 {
		handlerutil.Error(c, http.StatusBadRequest, apperror.ValidationError, "入力内容を確認してください", details)
		return
	}

	video, err := deps.Videos.UploadVideo(c.Request.Context(), uid, title, desc, mime, filename, rc)
	switch {
	case errors.Is(err, context.Canceled), errors.Is(err, context.DeadlineExceeded):
		handlerutil.Error(c, http.StatusRequestTimeout, apperror.InternalError, "アップロードが中断されました", nil)
		return
	case errors.Is(err, service.ErrUnsupportedMedia):
		handlerutil.Error(c, http.StatusUnsupportedMediaType, apperror.UnsupportedMediaType, "この動画形式には対応していません", nil)
		return
	case errors.Is(err, service.ErrPayloadTooLarge):
		handlerutil.Error(c, http.StatusRequestEntityTooLarge, apperror.PayloadTooLarge, "ファイルサイズが大きすぎます", nil)
		return
	case err != nil:
		handlerutil.Error(c, http.StatusInternalServerError, apperror.InternalError, "サーバーで問題が発生しました", nil)
		return
	}

	handlerutil.Data(c, http.StatusCreated, gin.H{
		"id":     video.ID,
		"title":  video.Title,
		"status": video.Status,
	})
}

func handleVideosGet(c *gin.Context, deps api.Deps) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		handlerutil.Error(c, http.StatusBadRequest, apperror.ValidationError, "入力内容を確認してください", []apperror.FieldDetail{
			{Field: "id", Message: "UUID形式で指定してください"},
		})
		return
	}

	var viewerID int64
	var viewerRole string
	if uid, ok := middleware.UserID(c); ok {
		viewerID = uid
		viewerRole, _ = middleware.UserRole(c)
	}

	video, err := deps.Videos.GetVideo(c.Request.Context(), id, viewerID, viewerRole)
	if errors.Is(err, service.ErrNotFound) {
		handlerutil.Error(c, http.StatusNotFound, apperror.NotFound, "対象が見つかりません", nil)
		return
	}
	if err != nil {
		handlerutil.Error(c, http.StatusInternalServerError, apperror.InternalError, "サーバーで問題が発生しました", nil)
		return
	}

	handlerutil.Data(c, http.StatusOK, video)
}

func handleVideosStream(c *gin.Context, deps api.Deps) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		handlerutil.Error(c, http.StatusBadRequest, apperror.ValidationError, "入力内容を確認してください", []apperror.FieldDetail{
			{Field: "id", Message: "UUID形式で指定してください"},
		})
		return
	}

	var viewerID int64
	var viewerRole string
	if uid, ok := middleware.UserID(c); ok {
		viewerID = uid
		viewerRole, _ = middleware.UserRole(c)
	}

	if err := deps.Stream.ServeHTTP(c.Request.Context(), id, viewerID, viewerRole, c.ClientIP(), c.Writer, c.Request); err != nil {
		if errors.Is(err, service.ErrNotFound) {
			handlerutil.Error(c, http.StatusNotFound, apperror.NotFound, "対象が見つかりません", nil)
			return
		}
		handlerutil.Error(c, http.StatusInternalServerError, apperror.InternalError, "サーバーで問題が発生しました", nil)
		return
	}
}

func handleVideosPatch(c *gin.Context, deps api.Deps) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		handlerutil.Error(c, http.StatusBadRequest, apperror.ValidationError, "入力内容を確認してください", []apperror.FieldDetail{
			{Field: "id", Message: "UUID形式で指定してください"},
		})
		return
	}

	var req struct {
		Title       *string `json:"title"`
		Description *string `json:"description"`
		Status      *string `json:"status"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		handlerutil.Error(c, http.StatusBadRequest, apperror.ValidationError, "入力内容を確認してください", []apperror.FieldDetail{
			{Field: "body", Message: "JSON形式で送信してください"},
		})
		return
	}

	var details []apperror.FieldDetail
	if req.Title != nil {
		if d := validation.Title(*req.Title); d != nil {
			details = append(details, *d)
		}
	}
	if req.Description != nil {
		if d := validation.Description(*req.Description); d != nil {
			details = append(details, *d)
		}
	}
	var statusPtr *string
	if req.Status != nil {
		st, d := validation.ParseVideoStatus(*req.Status)
		if d != nil {
			details = append(details, *d)
		} else {
			statusPtr = &st
		}
	}
	if len(details) > 0 {
		handlerutil.Error(c, http.StatusBadRequest, apperror.ValidationError, "入力内容を確認してください", details)
		return
	}

	uid, _ := middleware.UserID(c)
	role, _ := middleware.UserRole(c)

	out, err := deps.Videos.PatchVideo(c.Request.Context(), id, uid, role, service.PatchVideoInput{
		Title:       req.Title,
		Description: req.Description,
		Status:      statusPtr,
	})
	if errors.Is(err, service.ErrForbidden) {
		handlerutil.Error(c, http.StatusForbidden, apperror.Forbidden, "権限がありません", nil)
		return
	}
	if errors.Is(err, service.ErrNotFound) {
		handlerutil.Error(c, http.StatusNotFound, apperror.NotFound, "対象が見つかりません", nil)
		return
	}
	if err != nil {
		handlerutil.Error(c, http.StatusInternalServerError, apperror.InternalError, "サーバーで問題が発生しました", nil)
		return
	}

	handlerutil.Data(c, http.StatusOK, out)
}

func handleVideosDelete(c *gin.Context, deps api.Deps) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		handlerutil.Error(c, http.StatusBadRequest, apperror.ValidationError, "入力内容を確認してください", []apperror.FieldDetail{
			{Field: "id", Message: "UUID形式で指定してください"},
		})
		return
	}
	uid, _ := middleware.UserID(c)
	role, _ := middleware.UserRole(c)

	if err := deps.Videos.DeleteVideo(c.Request.Context(), id, uid, role); err != nil {
		if errors.Is(err, service.ErrForbidden) {
			handlerutil.Error(c, http.StatusForbidden, apperror.Forbidden, "権限がありません", nil)
			return
		}
		if errors.Is(err, service.ErrNotFound) {
			handlerutil.Error(c, http.StatusNotFound, apperror.NotFound, "対象が見つかりません", nil)
			return
		}
		handlerutil.Error(c, http.StatusInternalServerError, apperror.InternalError, "サーバーで問題が発生しました", nil)
		return
	}
	c.Status(http.StatusNoContent)
}

func handleVideosMeList(c *gin.Context, deps api.Deps) {
	uid, ok := middleware.UserID(c)
	if !ok {
		handlerutil.Error(c, http.StatusUnauthorized, apperror.Unauthenticated, "ログインが必要です", nil)
		return
	}
	page := handlerutil.ParsePositiveInt(c.Query("page"), 1, 1_000_000)
	perPage := handlerutil.ParsePositiveInt(c.Query("per_page"), 20, 50)

	items, total, err := deps.Videos.ListMyVideos(c.Request.Context(), uid, page, perPage)
	if err != nil {
		handlerutil.Error(c, http.StatusInternalServerError, apperror.InternalError, "サーバーで問題が発生しました", nil)
		return
	}

	handlerutil.List(c, items, handlerutil.Meta{
		Page:       page,
		PerPage:    perPage,
		Total:      total,
		TotalPages: handlerutil.TotalPages(total, perPage),
	})
}
