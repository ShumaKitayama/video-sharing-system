package v1

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"video-sharing-system/server/internal/api"
	"video-sharing-system/server/internal/apperror"
	"video-sharing-system/server/internal/handlerutil"
	"video-sharing-system/server/internal/middleware"
	"video-sharing-system/server/internal/service"
	"video-sharing-system/server/internal/validation"
)

func registerUserRoutes(g *gin.RouterGroup, deps api.Deps) {
	g.GET("/users/:id", middleware.RequireAuth(), func(c *gin.Context) {
		id, err := uuid.Parse(c.Param("id"))
		if err != nil {
			handlerutil.Error(c, http.StatusBadRequest, apperror.ValidationError, "入力内容を確認してください", []apperror.FieldDetail{
				{Field: "id", Message: "UUID形式で指定してください"},
			})
			return
		}
		role, _ := middleware.UserRole(c)
		pidStr, _ := middleware.UserPublicID(c)
		actorPublic, _ := uuid.Parse(pidStr)

		u, err := deps.Users.GetUser(c.Request.Context(), id, role, actorPublic)
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
		handlerutil.Data(c, http.StatusOK, u)
	})

	g.PATCH("/users/:id", middleware.RequireAuth(), func(c *gin.Context) {
		id, err := uuid.Parse(c.Param("id"))
		if err != nil {
			handlerutil.Error(c, http.StatusBadRequest, apperror.ValidationError, "入力内容を確認してください", []apperror.FieldDetail{
				{Field: "id", Message: "UUID形式で指定してください"},
			})
			return
		}

		var req struct {
			DisplayName *string `json:"display_name"`
			Password    *string `json:"password"`
			Role        *string `json:"role"`
			IsActive    *bool   `json:"is_active"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			handlerutil.Error(c, http.StatusBadRequest, apperror.ValidationError, "入力内容を確認してください", []apperror.FieldDetail{
				{Field: "body", Message: "JSON形式で送信してください"},
			})
			return
		}

		var details []apperror.FieldDetail
		if req.DisplayName != nil {
			if d := validation.DisplayName(*req.DisplayName); d != nil {
				details = append(details, *d)
			}
		}
		if req.Password != nil {
			if d := validation.Password(*req.Password); d != nil {
				details = append(details, *d)
			}
		}
		if req.Role != nil {
			if _, d := validation.ParseUserRole(*req.Role); d != nil {
				details = append(details, *d)
			}
		}
		if len(details) > 0 {
			handlerutil.Error(c, http.StatusBadRequest, apperror.ValidationError, "入力内容を確認してください", details)
			return
		}

		role, _ := middleware.UserRole(c)
		pidStr, _ := middleware.UserPublicID(c)
		actorPublic, _ := uuid.Parse(pidStr)

		var rolePtr *string
		if req.Role != nil {
			rl, _ := validation.ParseUserRole(*req.Role)
			rolePtr = &rl
		}

		out, err := deps.Users.PatchUser(c.Request.Context(), id, role, actorPublic, service.PatchUserInput{
			DisplayName: req.DisplayName,
			Password:    req.Password,
			Role:        rolePtr,
			IsActive:    req.IsActive,
		})
		if errors.Is(err, service.ErrForbidden) {
			handlerutil.Error(c, http.StatusForbidden, apperror.Forbidden, "権限がありません", nil)
			return
		}
		if errors.Is(err, service.ErrNotFound) {
			handlerutil.Error(c, http.StatusNotFound, apperror.NotFound, "対象が見つかりません", nil)
			return
		}
		if errors.Is(err, service.ErrConflict) {
			handlerutil.Error(c, http.StatusConflict, apperror.Conflict, "状態が競合しました", nil)
			return
		}
		if err != nil {
			handlerutil.Error(c, http.StatusInternalServerError, apperror.InternalError, "サーバーで問題が発生しました", nil)
			return
		}

		handlerutil.Data(c, http.StatusOK, out)
	})
}
