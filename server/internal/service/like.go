package service

import (
	"context"

	"github.com/google/uuid"

	"video-sharing-system/server/internal/repository"
)

type LikeService struct {
	likes  *repository.LikeRepository
	videos *VideoService
}

func NewLikeService(likes *repository.LikeRepository, videos *VideoService) *LikeService {
	return &LikeService{likes: likes, videos: videos}
}

func (s *LikeService) Get(ctx context.Context, videoPublic uuid.UUID, userID int64, role string) (bool, error) {
	vrow, err := s.videos.RequireAccessibleVideo(ctx, videoPublic, userID, role)
	if err != nil {
		return false, err
	}
	return s.likes.Exists(ctx, vrow.ID, userID)
}

func (s *LikeService) Put(ctx context.Context, videoPublic uuid.UUID, userID int64, role string) (bool, int64, error) {
	vrow, err := s.videos.RequireAccessibleVideo(ctx, videoPublic, userID, role)
	if err != nil {
		return false, 0, err
	}
	inserted, count, err := s.likes.Add(ctx, vrow.ID, userID)
	return inserted, count, err
}

func (s *LikeService) Delete(ctx context.Context, videoPublic uuid.UUID, userID int64, role string) (int64, error) {
	vrow, err := s.videos.RequireAccessibleVideo(ctx, videoPublic, userID, role)
	if err != nil {
		return 0, err
	}
	_, count, err := s.likes.Remove(ctx, vrow.ID, userID)
	return count, err
}
