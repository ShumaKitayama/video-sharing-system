package service

import (
	"video-sharing-system/server/internal/repository"
	"video-sharing-system/server/internal/storage"
)

// VideoService coordinates published-video workflows between Postgres and disk storage.
type VideoService struct {
	repo  *repository.VideoRepository
	store *storage.VideoStorage
}

// NewVideoService constructs VideoService with its collaborators.
func NewVideoService(repo *repository.VideoRepository, store *storage.VideoStorage) *VideoService {
	return &VideoService{repo: repo, store: store}
}
