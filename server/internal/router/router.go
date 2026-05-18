package router

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"

	"video-sharing-system/server/internal/api"
	apiv1 "video-sharing-system/server/internal/api/v1"
	"video-sharing-system/server/internal/apperror"
	"video-sharing-system/server/internal/config"
	"video-sharing-system/server/internal/handlerutil"
	"video-sharing-system/server/internal/middleware"
)

// New builds the Gin engine: global middleware, /health, /ready, and API versions under /api.
func New(cfg config.Config, deps api.Deps) *gin.Engine {
	r := gin.New()
	r.Use(gin.Logger())
	r.Use(gin.Recovery())
	r.Use(middleware.RequestID())

	r.Use(cors.New(cors.Config{
		AllowOrigins:     cfg.CORSOrigins,
		AllowMethods:     []string{"GET", "POST", "PATCH", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "X-Request-ID"},
		ExposeHeaders:    []string{"X-Request-ID"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	r.GET("/health", func(c *gin.Context) {
		handlerutil.Data(c, http.StatusOK, gin.H{"status": "ok"})
	})

	r.GET("/ready", func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
		defer cancel()

		var details []apperror.FieldDetail
		dbStatus := "ok"
		if err := deps.Pool.Ping(ctx); err != nil {
			dbStatus = "error"
			details = append(details, apperror.FieldDetail{Field: "database", Message: err.Error()})
		}

		stStatus := "ok"
		if err := deps.Store.PrepareWritableDir(ctx); err != nil {
			stStatus = "error"
			details = append(details, apperror.FieldDetail{Field: "storage", Message: err.Error()})
		}

		if dbStatus != "ok" || stStatus != "ok" {
			handlerutil.Error(c, http.StatusServiceUnavailable, apperror.ServiceUnavailable, "依存サービスの準備ができていません", details)
			return
		}

		handlerutil.Data(c, http.StatusOK, gin.H{
			"status":   "ready",
			"database": dbStatus,
			"storage":  stStatus,
		})
	})

	v1Group := r.Group("/api/v1")
	v1Group.Use(middleware.Auth(deps.Auth))
	apiv1.Register(v1Group, cfg, deps)

	return r
}
