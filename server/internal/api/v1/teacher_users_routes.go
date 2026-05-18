package v1

import (
	"github.com/gin-gonic/gin"

	"video-sharing-system/server/internal/api"
	"video-sharing-system/server/internal/middleware"
)

func registerTeacherUserRoutes(g *gin.RouterGroup, deps api.Deps) {
	teacher := g.Group("")
	teacher.Use(middleware.RequireTeacher())
	{
		teacher.GET("/users", func(c *gin.Context) { handleTeacherUsersList(c, deps) })
		teacher.POST("/users", func(c *gin.Context) { handleTeacherUsersCreate(c, deps) })
		teacher.DELETE("/users/:id", func(c *gin.Context) { handleTeacherUsersDelete(c, deps) })
	}
}
