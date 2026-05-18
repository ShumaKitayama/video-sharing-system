package service

import (
	"github.com/google/uuid"

	"video-sharing-system/server/internal/repository"
)

type UploaderDTO struct {
	ID          uuid.UUID `json:"id"`
	DisplayName string    `json:"display_name"`
}

type VideoDTO struct {
	ID              uuid.UUID   `json:"id"`
	Title           string      `json:"title"`
	Description     string      `json:"description"`
	Status          string      `json:"status"`
	Uploader        UploaderDTO `json:"uploader"`
	MimeType        string      `json:"mime_type"`
	FileSizeBytes   int64       `json:"file_size_bytes"`
	DurationSeconds *int        `json:"duration_seconds,omitempty"`
	ViewCount       int64       `json:"view_count"`
	LikeCount       int64       `json:"like_count"`
	CommentCount    int64       `json:"comment_count"`
	CreatedAt       string      `json:"created_at"`
	UpdatedAt       string      `json:"updated_at"`
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

// CanAccessVideo enforces visibility rules for a video row (published vs draft/private paths).
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
