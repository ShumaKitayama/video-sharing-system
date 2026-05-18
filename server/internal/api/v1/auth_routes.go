package v1

import (
	"github.com/gin-gonic/gin"

	"video-sharing-system/server/internal/api"
	"video-sharing-system/server/internal/config"
	"video-sharing-system/server/internal/middleware"
)

func registerAuthRoutes(g *gin.RouterGroup, cfg config.Config, deps api.Deps) {
	g.POST("/auth/login", middleware.LoginRateLimit(deps.LoginRL), func(c *gin.Context) {
		handleAuthLogin(c, cfg, deps)
	})

	authz := g.Group("")
	authz.Use(middleware.RequireAuth())
	{
		authz.POST("/auth/logout", func(c *gin.Context) { handleAuthLogout(c, cfg, deps) })
		authz.GET("/auth/me", func(c *gin.Context) { handleAuthMe(c, deps) })
	}
}
