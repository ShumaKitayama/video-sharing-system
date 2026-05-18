package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type VideoRepository struct {
	pool *pgxpool.Pool
}

func NewVideoRepository(pool *pgxpool.Pool) *VideoRepository {
	return &VideoRepository{pool: pool}
}

type VideoRow struct {
	ID               int64
	PublicID         uuid.UUID
	UploaderID       int64
	Title            string
	Description      string
	Status           string
	StorageKey       string
	OriginalFilename string
	MimeType         string
	FileSizeBytes    int64
	DurationSeconds  *int32
	ViewCount        int64
	LikeCount        int64
	CommentCount     int64
	CreatedAt        time.Time
	UpdatedAt        time.Time

	UploaderPublicID   uuid.UUID
	UploaderDisplayName string
}

type VideoStorageInfo struct {
	ID             int64
	PublicID       uuid.UUID
	UploaderID     int64
	Status         string
	StorageKey     string
	MimeType       string
	FileSizeBytes  int64
	UpdatedAt      time.Time
}

func orderPublished(sort string) string {
	switch sort {
	case "oldest":
		return "v.created_at ASC"
	case "most_viewed":
		return "v.view_count DESC, v.created_at DESC"
	case "most_liked":
		return "v.like_count DESC, v.created_at DESC"
	default:
		return "v.created_at DESC"
	}
}

func (r *VideoRepository) Create(ctx context.Context, uploaderID int64, title, description, storageKey, originalFilename, mime string, size int64, duration *int32) (*VideoRow, error) {
	const q = `
INSERT INTO videos (
  uploader_id,
  title,
  description,
  storage_key,
  original_filename,
  mime_type,
  file_size_bytes,
  duration_seconds
) VALUES (
  $1, $2, $3, $4, $5, $6, $7, $8
)
RETURNING
  id,
  public_id,
  uploader_id,
  title,
  description,
  status,
  storage_key,
  original_filename,
  mime_type,
  file_size_bytes,
  duration_seconds,
  view_count,
  like_count,
  comment_count,
  created_at,
  updated_at;
`
	row := r.pool.QueryRow(ctx, q, uploaderID, title, description, storageKey, originalFilename, mime, size, duration)
	var v VideoRow
	if err := row.Scan(
		&v.ID,
		&v.PublicID,
		&v.UploaderID,
		&v.Title,
		&v.Description,
		&v.Status,
		&v.StorageKey,
		&v.OriginalFilename,
		&v.MimeType,
		&v.FileSizeBytes,
		&v.DurationSeconds,
		&v.ViewCount,
		&v.LikeCount,
		&v.CommentCount,
		&v.CreatedAt,
		&v.UpdatedAt,
	); err != nil {
		return nil, err
	}
	return &v, nil
}

func (r *VideoRepository) FindByPublicID(ctx context.Context, id uuid.UUID) (*VideoRow, error) {
	const q = `
SELECT
  v.id,
  v.public_id,
  v.uploader_id,
  v.title,
  v.description,
  v.status,
  v.storage_key,
  v.original_filename,
  v.mime_type,
  v.file_size_bytes,
  v.duration_seconds,
  v.view_count,
  v.like_count,
  v.comment_count,
  v.created_at,
  v.updated_at,
  u.public_id AS uploader_public_id,
  u.display_name AS uploader_display_name
FROM videos v
JOIN users u ON u.id = v.uploader_id
WHERE v.public_id = $1
  AND v.deleted_at IS NULL;
`
	var v VideoRow
	err := r.pool.QueryRow(ctx, q, id).Scan(
		&v.ID,
		&v.PublicID,
		&v.UploaderID,
		&v.Title,
		&v.Description,
		&v.Status,
		&v.StorageKey,
		&v.OriginalFilename,
		&v.MimeType,
		&v.FileSizeBytes,
		&v.DurationSeconds,
		&v.ViewCount,
		&v.LikeCount,
		&v.CommentCount,
		&v.CreatedAt,
		&v.UpdatedAt,
		&v.UploaderPublicID,
		&v.UploaderDisplayName,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &v, nil
}

func (r *VideoRepository) FindStorageInfo(ctx context.Context, id uuid.UUID) (*VideoStorageInfo, error) {
	const q = `
SELECT
  id,
  public_id,
  uploader_id,
  status,
  storage_key,
  mime_type,
  file_size_bytes,
  updated_at
FROM videos
WHERE public_id = $1
  AND deleted_at IS NULL;
`
	var v VideoStorageInfo
	err := r.pool.QueryRow(ctx, q, id).Scan(&v.ID, &v.PublicID, &v.UploaderID, &v.Status, &v.StorageKey, &v.MimeType, &v.FileSizeBytes, &v.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &v, nil
}

func (r *VideoRepository) ListPublished(ctx context.Context, qtext *string, uploaderPublicID *uuid.UUID, sort string, limit, offset int32) ([]VideoRow, error) {
	order := orderPublished(sort)
	sqlText := fmt.Sprintf(`
SELECT
  v.id,
  v.public_id,
  v.uploader_id,
  v.title,
  v.description,
  v.status,
  v.storage_key,
  v.original_filename,
  v.mime_type,
  v.file_size_bytes,
  v.duration_seconds,
  v.view_count,
  v.like_count,
  v.comment_count,
  v.created_at,
  v.updated_at,
  u.public_id AS uploader_public_id,
  u.display_name AS uploader_display_name
FROM videos v
JOIN users u ON u.id = v.uploader_id
WHERE v.deleted_at IS NULL
  AND v.status = 'published'
  AND (
    $1::TEXT IS NULL
    OR v.title ILIKE '%%' || $1 || '%%'
    OR v.description ILIKE '%%' || $1 || '%%'
    OR u.display_name ILIKE '%%' || $1 || '%%'
  )
  AND ($2::UUID IS NULL OR u.public_id = $2)
ORDER BY %s
LIMIT $3
OFFSET $4;
`, order)

	rows, err := r.pool.Query(ctx, sqlText, qtext, uploaderPublicID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanVideoRows(rows)
}

func (r *VideoRepository) CountPublished(ctx context.Context, qtext *string, uploaderPublicID *uuid.UUID) (int64, error) {
	const q = `
SELECT COUNT(*)
FROM videos v
JOIN users u ON u.id = v.uploader_id
WHERE v.deleted_at IS NULL
  AND v.status = 'published'
  AND (
    $1::TEXT IS NULL
    OR v.title ILIKE '%' || $1 || '%'
    OR v.description ILIKE '%' || $1 || '%'
    OR u.display_name ILIKE '%' || $1 || '%'
  )
  AND ($2::UUID IS NULL OR u.public_id = $2);
`
	var total int64
	if err := r.pool.QueryRow(ctx, q, qtext, uploaderPublicID).Scan(&total); err != nil {
		return 0, err
	}
	return total, nil
}

func (r *VideoRepository) ListForTeacher(ctx context.Context, status *string, qtext *string, sort string, limit, offset int32) ([]VideoRow, error) {
	order := orderPublished(sort)

	var statusEnum any
	if status == nil {
		statusEnum = nil
	} else {
		statusEnum = *status
	}

	sqlText := fmt.Sprintf(`
SELECT
  v.id,
  v.public_id,
  v.uploader_id,
  v.title,
  v.description,
  v.status,
  v.storage_key,
  v.original_filename,
  v.mime_type,
  v.file_size_bytes,
  v.duration_seconds,
  v.view_count,
  v.like_count,
  v.comment_count,
  v.created_at,
  v.updated_at,
  u.public_id AS uploader_public_id,
  u.display_name AS uploader_display_name
FROM videos v
JOIN users u ON u.id = v.uploader_id
WHERE v.deleted_at IS NULL
  AND ($1::video_status IS NULL OR v.status = $1)
  AND (
    $2::TEXT IS NULL
    OR v.title ILIKE '%%' || $2 || '%%'
    OR v.description ILIKE '%%' || $2 || '%%'
    OR u.display_name ILIKE '%%' || $2 || '%%'
  )
ORDER BY %s
LIMIT $3
OFFSET $4;
`, order)

	rows, err := r.pool.Query(ctx, sqlText, statusEnum, qtext, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanVideoRows(rows)
}

func (r *VideoRepository) CountForTeacher(ctx context.Context, status *string, qtext *string) (int64, error) {
	var statusEnum any
	if status == nil {
		statusEnum = nil
	} else {
		statusEnum = *status
	}

	const q = `
SELECT COUNT(*)
FROM videos v
JOIN users u ON u.id = v.uploader_id
WHERE v.deleted_at IS NULL
  AND ($1::video_status IS NULL OR v.status = $1)
  AND (
    $2::TEXT IS NULL
    OR v.title ILIKE '%' || $2 || '%'
    OR v.description ILIKE '%' || $2 || '%'
    OR u.display_name ILIKE '%' || $2 || '%'
  );
`
	var total int64
	if err := r.pool.QueryRow(ctx, q, statusEnum, qtext).Scan(&total); err != nil {
		return 0, err
	}
	return total, nil
}

func (r *VideoRepository) ListByUploader(ctx context.Context, uploaderID int64, limit, offset int32) ([]VideoRow, error) {
	const q = `
SELECT
  v.id,
  v.public_id,
  v.uploader_id,
  v.title,
  v.description,
  v.status,
  v.storage_key,
  v.original_filename,
  v.mime_type,
  v.file_size_bytes,
  v.duration_seconds,
  v.view_count,
  v.like_count,
  v.comment_count,
  v.created_at,
  v.updated_at,
  u.public_id AS uploader_public_id,
  u.display_name AS uploader_display_name
FROM videos v
JOIN users u ON u.id = v.uploader_id
WHERE v.uploader_id = $1
  AND v.deleted_at IS NULL
ORDER BY v.created_at DESC
LIMIT $2
OFFSET $3;
`
	rows, err := r.pool.Query(ctx, q, uploaderID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanVideoRows(rows)
}

func (r *VideoRepository) CountByUploader(ctx context.Context, uploaderID int64) (int64, error) {
	const q = `
SELECT COUNT(*)
FROM videos
WHERE uploader_id = $1
  AND deleted_at IS NULL;
`
	var total int64
	if err := r.pool.QueryRow(ctx, q, uploaderID).Scan(&total); err != nil {
		return 0, err
	}
	return total, nil
}

type VideoMetadataPatch struct {
	Title       *string
	Description *string
}

func (r *VideoRepository) UpdateMetadata(ctx context.Context, publicID uuid.UUID, patch VideoMetadataPatch) (*VideoRow, error) {
	const q = `
UPDATE videos
SET
  title = COALESCE($2, title),
  description = COALESCE($3, description),
  updated_at = NOW()
WHERE public_id = $1
  AND deleted_at IS NULL
RETURNING
  id,
  public_id,
  uploader_id,
  title,
  description,
  status,
  storage_key,
  original_filename,
  mime_type,
  file_size_bytes,
  duration_seconds,
  view_count,
  like_count,
  comment_count,
  created_at,
  updated_at;
`
	row := r.pool.QueryRow(ctx, q, publicID, patch.Title, patch.Description)
	var v VideoRow
	err := row.Scan(
		&v.ID,
		&v.PublicID,
		&v.UploaderID,
		&v.Title,
		&v.Description,
		&v.Status,
		&v.StorageKey,
		&v.OriginalFilename,
		&v.MimeType,
		&v.FileSizeBytes,
		&v.DurationSeconds,
		&v.ViewCount,
		&v.LikeCount,
		&v.CommentCount,
		&v.CreatedAt,
		&v.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	if err := r.pool.QueryRow(ctx, `
SELECT u.public_id, u.display_name
FROM videos v JOIN users u ON u.id = v.uploader_id
WHERE v.public_id = $1 AND v.deleted_at IS NULL
`, publicID).Scan(&v.UploaderPublicID, &v.UploaderDisplayName); err != nil {
		return nil, err
	}

	return &v, nil
}

func (r *VideoRepository) UpdateStatus(ctx context.Context, publicID uuid.UUID, status string) (*VideoRow, error) {
	const q = `
UPDATE videos
SET
  status = $2::video_status,
  updated_at = NOW()
WHERE public_id = $1
  AND deleted_at IS NULL
RETURNING
  id,
  public_id,
  uploader_id,
  title,
  description,
  status,
  storage_key,
  original_filename,
  mime_type,
  file_size_bytes,
  duration_seconds,
  view_count,
  like_count,
  comment_count,
  created_at,
  updated_at;
`
	row := r.pool.QueryRow(ctx, q, publicID, status)
	var v VideoRow
	err := row.Scan(
		&v.ID,
		&v.PublicID,
		&v.UploaderID,
		&v.Title,
		&v.Description,
		&v.Status,
		&v.StorageKey,
		&v.OriginalFilename,
		&v.MimeType,
		&v.FileSizeBytes,
		&v.DurationSeconds,
		&v.ViewCount,
		&v.LikeCount,
		&v.CommentCount,
		&v.CreatedAt,
		&v.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	if err := r.pool.QueryRow(ctx, `
SELECT u.public_id, u.display_name
FROM videos v JOIN users u ON u.id = v.uploader_id
WHERE v.public_id = $1 AND v.deleted_at IS NULL
`, publicID).Scan(&v.UploaderPublicID, &v.UploaderDisplayName); err != nil {
		return nil, err
	}

	return &v, nil
}

func (r *VideoRepository) SoftDelete(ctx context.Context, publicID uuid.UUID) (string, bool, error) {
	const q = `
UPDATE videos
SET
  deleted_at = NOW(),
  updated_at = NOW()
WHERE public_id = $1
  AND deleted_at IS NULL
RETURNING storage_key;
`
	var key string
	err := r.pool.QueryRow(ctx, q, publicID).Scan(&key)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", false, nil
	}
	if err != nil {
		return "", false, err
	}
	return key, true, nil
}

func (r *VideoRepository) IncrementViewCount(ctx context.Context, id int64) error {
	const q = `
UPDATE videos
SET view_count = view_count + 1
WHERE id = $1
  AND deleted_at IS NULL;
`
	_, err := r.pool.Exec(ctx, q, id)
	return err
}

func scanVideoRows(rows pgx.Rows) ([]VideoRow, error) {
	var out []VideoRow
	for rows.Next() {
		var v VideoRow
		if err := rows.Scan(
			&v.ID,
			&v.PublicID,
			&v.UploaderID,
			&v.Title,
			&v.Description,
			&v.Status,
			&v.StorageKey,
			&v.OriginalFilename,
			&v.MimeType,
			&v.FileSizeBytes,
			&v.DurationSeconds,
			&v.ViewCount,
			&v.LikeCount,
			&v.CommentCount,
			&v.CreatedAt,
			&v.UpdatedAt,
			&v.UploaderPublicID,
			&v.UploaderDisplayName,
		); err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, rows.Err()
}
