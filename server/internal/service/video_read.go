package service

import (
	"context"

	"github.com/google/uuid"

	"video-sharing-system/server/internal/repository"
)

func (s *VideoService) GetVideo(ctx context.Context, id uuid.UUID, viewerUserID int64, viewerRole string) (VideoDTO, error) {
	v, err := s.repo.FindByPublicID(ctx, id)
	if err != nil {
		return VideoDTO{}, err
	}
	if v == nil {
		return VideoDTO{}, ErrNotFound
	}
	if !CanAccessVideo(v.Status, v.UploaderID, viewerUserID, viewerRole) {
		return VideoDTO{}, ErrNotFound
	}
	return dtoFromVideoRow(*v), nil
}

func (s *VideoService) RequireAccessibleVideo(ctx context.Context, id uuid.UUID, viewerUserID int64, viewerRole string) (*repository.VideoRow, error) {
	v, err := s.repo.FindByPublicID(ctx, id)
	if err != nil {
		return nil, err
	}
	if v == nil {
		return nil, ErrNotFound
	}
	if !CanAccessVideo(v.Status, v.UploaderID, viewerUserID, viewerRole) {
		return nil, ErrNotFound
	}
	return v, nil
}

func (s *VideoService) GetVideoInternalID(ctx context.Context, public uuid.UUID) (int64, error) {
	v, err := s.repo.FindByPublicID(ctx, public)
	if err != nil {
		return 0, err
	}
	if v == nil {
		return 0, ErrNotFound
	}
	return v.ID, nil
}
