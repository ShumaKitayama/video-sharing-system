package service

import (
	"context"
	"errors"
	"io"
	"net/url"
	"path/filepath"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"

	"video-sharing-system/server/internal/repository"
	"video-sharing-system/server/internal/validation"
)

const (
	blobUploadMaxBytes    = 500 * 1024 * 1024
	blobClientTokenExpiry = time.Hour
)

var blobAllowedContentTypes = []string{"video/mp4", "video/webm"}

// IssueBlobUploadToken returns a browser upload token for a direct-to-Blob upload.
func (s *VideoService) IssueBlobUploadToken(pathname string) (string, error) {
	return s.store.IssueBlobClientToken(pathname, blobAllowedContentTypes, blobUploadMaxBytes, blobClientTokenExpiry)
}

// BlobEnabled reports whether direct Blob uploads are available.
func (s *VideoService) BlobEnabled() bool {
	return s.store.BlobEnabled()
}

// RegisterBlobVideo creates a video record for a file the browser already
// uploaded directly to Vercel Blob. No file bytes flow through the API here.
func (s *VideoService) RegisterBlobVideo(ctx context.Context, uploaderID int64, title, description, blobURL, declaredMIME string, duration *int32) (VideoDTO, error) {
	parsed, err := url.Parse(strings.TrimSpace(blobURL))
	if err != nil || parsed.Scheme != "https" || !strings.HasSuffix(strings.ToLower(parsed.Hostname()), ".blob.vercel-storage.com") {
		return VideoDTO{}, ErrInvalidBlobURL
	}

	// HEAD the object to confirm it exists and to read authoritative metadata.
	size, headContentType, err := s.store.StatBlob(ctx, blobURL)
	if err != nil {
		return VideoDTO{}, ErrInvalidBlobURL
	}
	if size <= 0 {
		return VideoDTO{}, ErrInvalidBlobURL
	}
	if size > blobUploadMaxBytes {
		return VideoDTO{}, ErrPayloadTooLarge
	}

	name := strings.TrimSpace(filepath.Base(parsed.Path))
	if name == "" {
		name = "video.mp4"
	}
	if utf8.RuneCountInString(name) > 255 {
		name = string([]rune(name)[:255])
	}

	// Prefer the blob's stored content type; fall back to the client hint.
	mime := strings.TrimSpace(strings.ToLower(headContentType))
	if !validation.IsAllowedVideoMIME(mime) {
		mime = strings.TrimSpace(strings.ToLower(declaredMIME))
	}
	if !validation.IsAllowedVideoMIME(mime) || !validation.FilenameMatchesMIME(name, mime) {
		return VideoDTO{}, ErrUnsupportedMedia
	}

	row, err := s.repo.Create(ctx, uploaderID, strings.TrimSpace(title), description, blobURL, name, mime, size, duration)
	if err != nil {
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

func (s *VideoService) UploadVideo(ctx context.Context, uploaderID int64, title, description, declaredMIME, originalFilename string, duration *int32, src io.Reader) (VideoDTO, error) {
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
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return VideoDTO{}, err
		}
		if strings.Contains(err.Error(), "payload too large") {
			return VideoDTO{}, ErrPayloadTooLarge
		}
		if strings.Contains(err.Error(), "unsupported media") {
			return VideoDTO{}, ErrUnsupportedMedia
		}
		return VideoDTO{}, err
	}

	row, err := s.repo.Create(ctx, uploaderID, strings.TrimSpace(title), description, key, name, mime, size, duration)
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
