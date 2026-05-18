package v1

import (
	"github.com/gin-gonic/gin"

	"video-sharing-system/server/internal/api"
	"video-sharing-system/server/internal/middleware"
)

func registerCommentAndLikeRoutes(g *gin.RouterGroup, deps api.Deps) {
	g.GET("/videos/:id/comments", func(c *gin.Context) { handleCommentsListForVideo(c, deps) })
	g.POST("/videos/:id/comments", middleware.RequireAuth(), func(c *gin.Context) { handleCommentsCreateForVideo(c, deps) })
	g.DELETE("/comments/:id", middleware.RequireAuth(), func(c *gin.Context) { handleCommentsDelete(c, deps) })
	g.GET("/videos/:id/like", middleware.RequireAuth(), func(c *gin.Context) { handleLikesGet(c, deps) })
	g.PUT("/videos/:id/like", middleware.RequireAuth(), func(c *gin.Context) { handleLikesPut(c, deps) })
	g.DELETE("/videos/:id/like", middleware.RequireAuth(), func(c *gin.Context) { handleLikesDelete(c, deps) })
}
