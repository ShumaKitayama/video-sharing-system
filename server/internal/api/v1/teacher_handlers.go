package v1

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"video-sharing-system/server/internal/api"
	"video-sharing-system/server/internal/apperror"
	"video-sharing-system/server/internal/handlerutil"
	"video-sharing-system/server/internal/service"
	"video-sharing-system/server/internal/validation"
)

func handleTeacherUsersList(c *gin.Context, deps api.Deps) {
	page := handlerutil.ParsePositiveInt(c.Query("page"), 1, 1_000_000)
	perPage := handlerutil.ParsePositiveInt(c.Query("per_page"), 20, 50)

	var rolePtr *string
	if raw := strings.TrimSpace(c.Query("role")); raw != "" {
		rl, d := validation.ParseUserRole(raw)
		if d != nil {
			handlerutil.Error(c, http.StatusBadRequest, apperror.ValidationError, "入力内容を確認してください", []apperror.FieldDetail{*d})
			return
		}
		rolePtr = &rl
	}

	activePtr, d := validation.ParseBoolQueryParam(c.Query("is_active"))
	if d != nil {
		handlerutil.Error(c, http.StatusBadRequest, apperror.ValidationError, "入力内容を確認してください", []apperror.FieldDetail{*d})
		return
	}

	var qPtr *string
	if q := strings.TrimSpace(c.Query("q")); q != "" {
		qPtr = &q
	}

	items, total, err := deps.Users.ListUsers(c.Request.Context(), rolePtr, activePtr, qPtr, page, perPage)
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

func handleTeacherUsersCreate(c *gin.Context, deps api.Deps) {
	var req struct {
		Username    string `json:"username"`
		DisplayName string `json:"display_name"`
		Password    string `json:"password"`
		Role        string `json:"role"`
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
	if d := validation.DisplayName(req.DisplayName); d != nil {
		details = append(details, *d)
	}
	if d := validation.Password(req.Password); d != nil {
		details = append(details, *d)
	}
	role, d := validation.ParseUserRole(req.Role)
	if d != nil {
		details = append(details, *d)
	}
	if len(details) > 0 {
		handlerutil.Error(c, http.StatusBadRequest, apperror.ValidationError, "入力内容を確認してください", details)
		return
	}

	u, err := deps.Users.CreateTeacherUser(c.Request.Context(), strings.TrimSpace(req.Username), strings.TrimSpace(req.DisplayName), req.Password, role)
	if errors.Is(err, service.ErrConflict) {
		handlerutil.Error(c, http.StatusConflict, apperror.Conflict, "すでに存在するユーザーです", nil)
		return
	}
	if err != nil {
		handlerutil.Error(c, http.StatusInternalServerError, apperror.InternalError, "サーバーで問題が発生しました", nil)
		return
	}
	handlerutil.Data(c, http.StatusCreated, u)
}

func handleTeacherUsersDelete(c *gin.Context, deps api.Deps) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		handlerutil.Error(c, http.StatusBadRequest, apperror.ValidationError, "入力内容を確認してください", []apperror.FieldDetail{
			{Field: "id", Message: "UUID形式で指定してください"},
		})
		return
	}
	if err := deps.Users.SoftDeleteUser(c.Request.Context(), id, "teacher"); err != nil {
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
