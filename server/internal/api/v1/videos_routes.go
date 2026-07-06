package v1

import (
	"github.com/gin-gonic/gin"

	"video-sharing-system/server/internal/api"
	"video-sharing-system/server/internal/config"
	"video-sharing-system/server/internal/middleware"
)

func registerVideoRoutes(g *gin.RouterGroup, cfg config.Config, deps api.Deps) {
	g.GET("/videos", func(c *gin.Context) { handleVideosList(c, deps) })
	g.POST("/uploads/token", middleware.RequireAuth(), func(c *gin.Context) { handleUploadTokenIssue(c, cfg, deps) })
	g.POST("/uploads/blob", func(c *gin.Context) { handleBlobUpload(c, cfg, deps) })
	g.POST("/videos", middleware.RequireAuthOrUploadToken(cfg.UploadTokenSecret, deps.Auth), func(c *gin.Context) { handleVideosCreate(c, deps) })
	g.GET("/videos/:id", func(c *gin.Context) { handleVideosGet(c, deps) })
	g.GET("/videos/:id/stream", func(c *gin.Context) { handleVideosStream(c, deps) })
	g.PATCH("/videos/:id", middleware.RequireAuth(), func(c *gin.Context) { handleVideosPatch(c, deps) })
	g.DELETE("/videos/:id", middleware.RequireAuth(), func(c *gin.Context) { handleVideosDelete(c, deps) })
	g.GET("/me/videos", middleware.RequireAuth(), func(c *gin.Context) { handleVideosMeList(c, deps) })
}
