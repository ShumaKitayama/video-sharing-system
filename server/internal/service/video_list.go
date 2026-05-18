package service

import (
	"context"

	"github.com/google/uuid"
)

func (s *VideoService) ListVideos(ctx context.Context, viewerRole string, teacherStatus *string, q *string, uploaderPublic *uuid.UUID, sort string, page, perPage int) ([]VideoDTO, int64, error) {
	offset := int32((page - 1) * perPage)

	if viewerRole == "teacher" {
		var statusEnum *string
		if teacherStatus == nil {
			statusEnum = nil
		} else if *teacherStatus == "all" {
			statusEnum = nil
		} else {
			statusEnum = teacherStatus
		}

		items, err := s.repo.ListForTeacher(ctx, statusEnum, q, sort, int32(perPage), offset)
		if err != nil {
			return nil, 0, err
		}
		total, err := s.repo.CountForTeacher(ctx, statusEnum, q)
		if err != nil {
			return nil, 0, err
		}
		out := make([]VideoDTO, 0, len(items))
		for _, row := range items {
			out = append(out, dtoFromVideoRow(row))
		}
		return out, total, nil
	}

	items, err := s.repo.ListPublished(ctx, q, uploaderPublic, sort, int32(perPage), offset)
	if err != nil {
		return nil, 0, err
	}
	total, err := s.repo.CountPublished(ctx, q, uploaderPublic)
	if err != nil {
		return nil, 0, err
	}
	out := make([]VideoDTO, 0, len(items))
	for _, row := range items {
		out = append(out, dtoFromVideoRow(row))
	}
	return out, total, nil
}

func (s *VideoService) ListMyVideos(ctx context.Context, uploaderID int64, page, perPage int) ([]VideoDTO, int64, error) {
	offset := int32((page - 1) * perPage)
	items, err := s.repo.ListByUploader(ctx, uploaderID, int32(perPage), offset)
	if err != nil {
		return nil, 0, err
	}
	total, err := s.repo.CountByUploader(ctx, uploaderID)
	if err != nil {
		return nil, 0, err
	}
	out := make([]VideoDTO, 0, len(items))
	for _, row := range items {
		out = append(out, dtoFromVideoRow(row))
	}
	return out, total, nil
}
