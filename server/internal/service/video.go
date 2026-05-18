package service

import (
	"context"
	"io"
	"path/filepath"
	"strings"
	"unicode/utf8"

	"github.com/google/uuid"

	"video-sharing-system/server/internal/repository"
	"video-sharing-system/server/internal/storage"
	"video-sharing-system/server/internal/validation"
)

type VideoService struct {
	repo  *repository.VideoRepository
	store *storage.VideoStorage
}

func NewVideoService(repo *repository.VideoRepository, store *storage.VideoStorage) *VideoService {
	return &VideoService{repo: repo, store: store}
}

type UploaderDTO struct {
	ID          uuid.UUID `json:"id"`
	DisplayName string    `json:"display_name"`
}

type VideoDTO struct {
	ID               uuid.UUID    `json:"id"`
	Title            string       `json:"title"`
	Description      string       `json:"description"`
	Status           string       `json:"status"`
	Uploader         UploaderDTO  `json:"uploader"`
	MimeType         string       `json:"mime_type"`
	FileSizeBytes    int64        `json:"file_size_bytes"`
	DurationSeconds  *int         `json:"duration_seconds,omitempty"`
	ViewCount        int64        `json:"view_count"`
	LikeCount        int64        `json:"like_count"`
	CommentCount     int64        `json:"comment_count"`
	CreatedAt        string       `json:"created_at"`
	UpdatedAt        string       `json:"updated_at"`
}

func dtoFromVideoRow(v repository.VideoRow) VideoDTO {
	var dur *int
	if v.DurationSeconds != nil {
		x := int(*v.DurationSeconds)
		dur = &x
	}
	return VideoDTO{
		ID:              v.PublicID,
		Title:           v.Title,
		Description:     v.Description,
		Status:          v.Status,
		Uploader:        UploaderDTO{ID: v.UploaderPublicID, DisplayName: v.UploaderDisplayName},
		MimeType:        v.MimeType,
		FileSizeBytes:   v.FileSizeBytes,
		DurationSeconds: dur,
		ViewCount:       v.ViewCount,
		LikeCount:       v.LikeCount,
		CommentCount:    v.CommentCount,
		CreatedAt:       formatRFC3339(v.CreatedAt),
		UpdatedAt:       formatRFC3339(v.UpdatedAt),
	}
}

func CanAccessVideo(status string, uploaderID int64, viewerUserID int64, viewerRole string) bool {
	if status == "published" {
		return true
	}
	if viewerRole == "teacher" {
		return true
	}
	if viewerUserID != 0 && viewerUserID == uploaderID {
		return true
	}
	return false
}

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

func (s *VideoService) UploadVideo(ctx context.Context, uploaderID int64, title, description, declaredMIME, originalFilename string, src io.Reader) (VideoDTO, error) {
	mime := strings.TrimSpace(strings.ToLower(declaredMIME))
	if !validation.IsAllowedVideoMIME(mime) {
		return VideoDTO{}, ErrUnsupportedMedia
	}
	if !validation.FilenameMatchesMIME(originalFilename, mime) {
		return VideoDTO{}, ErrUnsupportedMedia
	}

	name := strings.TrimSpace(filepath.Base(originalFilename))
	if name == "" {
		name = "video.bin"
	}
	if utf8.RuneCountInString(name) > 255 {
		name = string([]rune(name)[:255])
	}

	key, size, err := s.store.SaveUploadedVideo(ctx, src, mime, name)
	if err != nil {
		if strings.Contains(err.Error(), "payload too large") {
			return VideoDTO{}, ErrPayloadTooLarge
		}
		if strings.Contains(err.Error(), "unsupported media") {
			return VideoDTO{}, ErrUnsupportedMedia
		}
		return VideoDTO{}, err
	}

	row, err := s.repo.Create(ctx, uploaderID, strings.TrimSpace(title), description, key, name, mime, size, nil)
	if err != nil {
		_ = s.store.Delete(key)
		return VideoDTO{}, err
	}

	full, err := s.repo.FindByPublicID(ctx, row.PublicID)
	if err != nil {
		return VideoDTO{}, err
	}
	if full == nil {
		return VideoDTO{}, ErrNotFound
	}
	return dtoFromVideoRow(*full), nil
}

type PatchVideoInput struct {
	Title       *string
	Description *string
	Status      *string // teacher-only in handler/service enforcement
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

func (s *VideoService) PatchVideo(ctx context.Context, id uuid.UUID, actorUserID int64, actorRole string, in PatchVideoInput) (VideoDTO, error) {
	v, err := s.repo.FindByPublicID(ctx, id)
	if err != nil {
		return VideoDTO{}, err
	}
	if v == nil {
		return VideoDTO{}, ErrNotFound
	}

	canEditMeta := actorRole == "teacher" || v.UploaderID == actorUserID

	if in.Status != nil {
		if actorRole != "teacher" {
			return VideoDTO{}, ErrForbidden
		}
		updated, err := s.repo.UpdateStatus(ctx, id, *in.Status)
		if err != nil {
			return VideoDTO{}, err
		}
		if updated == nil {
			return VideoDTO{}, ErrNotFound
		}
	}

	if in.Title != nil || in.Description != nil {
		if !canEditMeta {
			return VideoDTO{}, ErrForbidden
		}
		patch := repository.VideoMetadataPatch{Title: in.Title, Description: in.Description}
		updated, err := s.repo.UpdateMetadata(ctx, id, patch)
		if err != nil {
			return VideoDTO{}, err
		}
		if updated == nil {
			return VideoDTO{}, ErrNotFound
		}
	}

	out, err := s.repo.FindByPublicID(ctx, id)
	if err != nil {
		return VideoDTO{}, err
	}
	if out == nil {
		return VideoDTO{}, ErrNotFound
	}
	return dtoFromVideoRow(*out), nil
}

func (s *VideoService) DeleteVideo(ctx context.Context, id uuid.UUID, actorUserID int64, actorRole string) error {
	v, err := s.repo.FindByPublicID(ctx, id)
	if err != nil {
		return err
	}
	if v == nil {
		return ErrNotFound
	}
	if actorRole != "teacher" && v.UploaderID != actorUserID {
		return ErrForbidden
	}

	key, ok, err := s.repo.SoftDelete(ctx, id)
	if err != nil {
		return err
	}
	if !ok {
		return ErrNotFound
	}

	if err := s.store.Delete(key); err != nil {
		// Best-effort cleanup per query-catalog 11.3 — already logically deleted.
		return nil
	}
	return nil
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
