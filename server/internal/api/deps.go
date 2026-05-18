package api

import (
	"github.com/jackc/pgx/v5/pgxpool"

	"video-sharing-system/server/internal/ratelimit"
	"video-sharing-system/server/internal/service"
	"video-sharing-system/server/internal/storage"
)

// Deps bundles runtime collaborators wired into HTTP handlers (v1 and future versions).
type Deps struct {
	Pool     *pgxpool.Pool
	Store    *storage.VideoStorage
	Auth     *service.AuthService
	Users    *service.UserService
	Videos   *service.VideoService
	Stream   *service.StreamService
	Comments *service.CommentService
	Likes    *service.LikeService

	LoginRL   *ratelimit.WindowLimiter
	UploadRL  *ratelimit.WindowLimiter
	CommentRL *ratelimit.WindowLimiter
}
