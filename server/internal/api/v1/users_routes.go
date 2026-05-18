package v1

import (
	"github.com/gin-gonic/gin"

	"video-sharing-system/server/internal/api"
	"video-sharing-system/server/internal/middleware"
)

func registerUserRoutes(g *gin.RouterGroup, deps api.Deps) {
	g.GET("/users/:id", middleware.RequireAuth(), func(c *gin.Context) { handleUsersGet(c, deps) })
	g.PATCH("/users/:id", middleware.RequireAuth(), func(c *gin.Context) { handleUsersPatch(c, deps) })
}
