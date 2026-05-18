package v1

import (
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

func registerCommentAndLikeRoutes(g *gin.RouterGroup, deps api.Deps) {
	g.GET("/videos/:id/comments", func(c *gin.Context) {
		id, err := uuid.Parse(c.Param("id"))
		if err != nil {
			handlerutil.Error(c, http.StatusBadRequest, apperror.ValidationError, "入力内容を確認してください", []apperror.FieldDetail{
				{Field: "id", Message: "UUID形式で指定してください"},
			})
			return
		}

		page := handlerutil.ParsePositiveInt(c.Query("page"), 1, 1_000_000)
		perPage := handlerutil.ParsePositiveInt(c.Query("per_page"), 20, 50)

		var viewerID int64
		var viewerRole string
		if uid, ok := middleware.UserID(c); ok {
			viewerID = uid
			viewerRole, _ = middleware.UserRole(c)
		}

		items, total, err := deps.Comments.List(c.Request.Context(), id, viewerID, viewerRole, page, perPage)
		if errors.Is(err, service.ErrNotFound) {
			handlerutil.Error(c, http.StatusNotFound, apperror.NotFound, "対象が見つかりません", nil)
			return
		}
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
	})

	g.POST("/videos/:id/comments", middleware.RequireAuth(), func(c *gin.Context) {
		id, err := uuid.Parse(c.Param("id"))
		if err != nil {
			handlerutil.Error(c, http.StatusBadRequest, apperror.ValidationError, "入力内容を確認してください", []apperror.FieldDetail{
				{Field: "id", Message: "UUID形式で指定してください"},
			})
			return
		}

		uid, ok := middleware.UserID(c)
		if !ok {
			handlerutil.Error(c, http.StatusUnauthorized, apperror.Unauthenticated, "ログインが必要です", nil)
			return
		}
		role, _ := middleware.UserRole(c)

		if !deps.CommentRL.Allow(fmt.Sprintf("comment:%d", uid)) {
			handlerutil.Error(c, http.StatusTooManyRequests, apperror.RateLimited, "コメント投稿はしばらく待ってから試してください", nil)
			return
		}

		var req struct {
			Body string `json:"body"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			handlerutil.Error(c, http.StatusBadRequest, apperror.ValidationError, "入力内容を確認してください", []apperror.FieldDetail{
				{Field: "body", Message: "JSON形式で送信してください"},
			})
			return
		}

		if d := validation.CommentBody(req.Body); d != nil {
			handlerutil.Error(c, http.StatusBadRequest, apperror.ValidationError, "入力内容を確認してください", []apperror.FieldDetail{*d})
			return
		}

		item, err := deps.Comments.Create(c.Request.Context(), id, uid, role, strings.TrimSpace(req.Body))
		if errors.Is(err, service.ErrNotFound) {
			handlerutil.Error(c, http.StatusNotFound, apperror.NotFound, "対象が見つかりません", nil)
			return
		}
		if err != nil {
			handlerutil.Error(c, http.StatusInternalServerError, apperror.InternalError, "サーバーで問題が発生しました", nil)
			return
		}

		handlerutil.Data(c, http.StatusCreated, item)
	})

	g.DELETE("/comments/:id", middleware.RequireAuth(), func(c *gin.Context) {
		id, err := uuid.Parse(c.Param("id"))
		if err != nil {
			handlerutil.Error(c, http.StatusBadRequest, apperror.ValidationError, "入力内容を確認してください", []apperror.FieldDetail{
				{Field: "id", Message: "UUID形式で指定してください"},
			})
			return
		}

		uid, _ := middleware.UserID(c)
		role, _ := middleware.UserRole(c)

		if err := deps.Comments.Delete(c.Request.Context(), id, uid, role); err != nil {
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
	})

	g.GET("/videos/:id/like", middleware.RequireAuth(), func(c *gin.Context) {
		id, err := uuid.Parse(c.Param("id"))
		if err != nil {
			handlerutil.Error(c, http.StatusBadRequest, apperror.ValidationError, "入力内容を確認してください", []apperror.FieldDetail{
				{Field: "id", Message: "UUID形式で指定してください"},
			})
			return
		}
		uid, _ := middleware.UserID(c)
		role, _ := middleware.UserRole(c)

		liked, err := deps.Likes.Get(c.Request.Context(), id, uid, role)
		if errors.Is(err, service.ErrNotFound) {
			handlerutil.Error(c, http.StatusNotFound, apperror.NotFound, "対象が見つかりません", nil)
			return
		}
		if err != nil {
			handlerutil.Error(c, http.StatusInternalServerError, apperror.InternalError, "サーバーで問題が発生しました", nil)
			return
		}

		handlerutil.Data(c, http.StatusOK, gin.H{"liked": liked})
	})

	g.PUT("/videos/:id/like", middleware.RequireAuth(), func(c *gin.Context) {
		id, err := uuid.Parse(c.Param("id"))
		if err != nil {
			handlerutil.Error(c, http.StatusBadRequest, apperror.ValidationError, "入力内容を確認してください", []apperror.FieldDetail{
				{Field: "id", Message: "UUID形式で指定してください"},
			})
			return
		}
		uid, _ := middleware.UserID(c)
		role, _ := middleware.UserRole(c)

		_, count, err := deps.Likes.Put(c.Request.Context(), id, uid, role)
		if errors.Is(err, service.ErrNotFound) {
			handlerutil.Error(c, http.StatusNotFound, apperror.NotFound, "対象が見つかりません", nil)
			return
		}
		if err != nil {
			handlerutil.Error(c, http.StatusInternalServerError, apperror.InternalError, "サーバーで問題が発生しました", nil)
			return
		}

		handlerutil.Data(c, http.StatusOK, gin.H{"liked": true, "like_count": count})
	})

	g.DELETE("/videos/:id/like", middleware.RequireAuth(), func(c *gin.Context) {
		id, err := uuid.Parse(c.Param("id"))
		if err != nil {
			handlerutil.Error(c, http.StatusBadRequest, apperror.ValidationError, "入力内容を確認してください", []apperror.FieldDetail{
				{Field: "id", Message: "UUID形式で指定してください"},
			})
			return
		}
		uid, _ := middleware.UserID(c)
		role, _ := middleware.UserRole(c)

		_, err = deps.Likes.Delete(c.Request.Context(), id, uid, role)
		if errors.Is(err, service.ErrNotFound) {
			handlerutil.Error(c, http.StatusNotFound, apperror.NotFound, "対象が見つかりません", nil)
			return
		}
		if err != nil {
			handlerutil.Error(c, http.StatusInternalServerError, apperror.InternalError, "サーバーで問題が発生しました", nil)
			return
		}

		c.Status(http.StatusNoContent)
	})
}
