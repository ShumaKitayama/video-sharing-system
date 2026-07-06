package v1

import (
	"github.com/gin-gonic/gin"

	"video-sharing-system/server/internal/api"
	"video-sharing-system/server/internal/config"
)

// Register mounts all handlers for /api/v1.
// The caller must create r.Group("/api/v1") and attach middleware.Auth before calling Register.
func Register(g *gin.RouterGroup, cfg config.Config, deps api.Deps) {
	registerAuthRoutes(g, cfg, deps)
	registerTeacherUserRoutes(g, deps)
	registerUserRoutes(g, deps)
	registerVideoRoutes(g, cfg, deps)
	registerCommentAndLikeRoutes(g, deps)
}
