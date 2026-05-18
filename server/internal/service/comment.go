package service

import (
	"context"

	"github.com/google/uuid"

	"video-sharing-system/server/internal/repository"
)

type CommentService struct {
	comments *repository.CommentRepository
	videos   *VideoService
}

func NewCommentService(comments *repository.CommentRepository, videos *VideoService) *CommentService {
	return &CommentService{comments: comments, videos: videos}
}

type CommentDTO struct {
	ID        uuid.UUID   `json:"id"`
	Body      string      `json:"body"`
	Author    UploaderDTO `json:"author"`
	CreatedAt string      `json:"created_at"`
	UpdatedAt string      `json:"updated_at"`
}

func (s *CommentService) List(ctx context.Context, videoPublic uuid.UUID, viewerUserID int64, viewerRole string, page, perPage int) ([]CommentDTO, int64, error) {
	vrow, err := s.videos.RequireAccessibleVideo(ctx, videoPublic, viewerUserID, viewerRole)
	if err != nil {
		return nil, 0, err
	}

	offset := int32((page - 1) * perPage)
	items, err := s.comments.ListByVideo(ctx, vrow.ID, int32(perPage), offset)
	if err != nil {
		return nil, 0, err
	}
	total, err := s.comments.CountByVideo(ctx, vrow.ID)
	if err != nil {
		return nil, 0, err
	}

	out := make([]CommentDTO, 0, len(items))
	for _, c := range items {
		out = append(out, CommentDTO{
			ID:        c.PublicID,
			Body:      c.Body,
			Author:    UploaderDTO{ID: c.AuthorPublicID, DisplayName: c.AuthorDisplayName},
			CreatedAt: formatRFC3339(c.CreatedAt),
			UpdatedAt: formatRFC3339(c.UpdatedAt),
		})
	}
	return out, total, nil
}

func (s *CommentService) Create(ctx context.Context, videoPublic uuid.UUID, authorUserID int64, viewerRole string, body string) (CommentDTO, error) {
	vrow, err := s.videos.RequireAccessibleVideo(ctx, videoPublic, authorUserID, viewerRole)
	if err != nil {
		return CommentDTO{}, err
	}

	row, err := s.comments.Create(ctx, vrow.ID, authorUserID, body)
	if err != nil {
		return CommentDTO{}, err
	}

	return CommentDTO{
		ID:        row.PublicID,
		Body:      row.Body,
		Author:    UploaderDTO{ID: row.AuthorPublicID, DisplayName: row.AuthorDisplayName},
		CreatedAt: formatRFC3339(row.CreatedAt),
		UpdatedAt: formatRFC3339(row.UpdatedAt),
	}, nil
}

func (s *CommentService) Delete(ctx context.Context, commentPublic uuid.UUID, actorUserID int64, actorRole string) error {
	c, err := s.comments.FindByPublicID(ctx, commentPublic)
	if err != nil {
		return err
	}
	if c == nil || c.DeletedAt != nil {
		return ErrNotFound
	}

	if actorRole != "teacher" && c.UserID != actorUserID {
		return ErrForbidden
	}

	ok, err := s.comments.SoftDeleteWithCounter(ctx, commentPublic)
	if err != nil {
		return err
	}
	if !ok {
		return ErrNotFound
	}
	return nil
}
