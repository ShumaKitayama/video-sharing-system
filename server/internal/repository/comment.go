package repository

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type CommentRepository struct {
	pool *pgxpool.Pool
}

func NewCommentRepository(pool *pgxpool.Pool) *CommentRepository {
	return &CommentRepository{pool: pool}
}

type CommentRow struct {
	ID        int64
	PublicID  uuid.UUID
	VideoID   int64
	UserID    int64
	Body      string
	CreatedAt time.Time
	UpdatedAt time.Time

	AuthorPublicID   uuid.UUID
	AuthorDisplayName string
}

type CommentInternal struct {
	ID        int64
	PublicID  uuid.UUID
	VideoID   int64
	UserID    int64
	Body      string
	DeletedAt *time.Time
}

func (r *CommentRepository) Create(ctx context.Context, videoID, userID int64, body string) (*CommentRow, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	const ins = `
INSERT INTO comments (
  video_id,
  user_id,
  body
) VALUES (
  $1, $2, $3
)
RETURNING id, public_id, video_id, user_id, body, created_at, updated_at;
`
	var c CommentRow
	if err := tx.QueryRow(ctx, ins, videoID, userID, body).Scan(&c.ID, &c.PublicID, &c.VideoID, &c.UserID, &c.Body, &c.CreatedAt, &c.UpdatedAt); err != nil {
		return nil, err
	}

	const inc = `
UPDATE videos
SET comment_count = comment_count + 1
WHERE id = $1
  AND deleted_at IS NULL;
`
	if _, err := tx.Exec(ctx, inc, videoID); err != nil {
		return nil, err
	}

	const author = `
SELECT public_id, display_name
FROM users
WHERE id = $1 AND deleted_at IS NULL;
`
	if err := tx.QueryRow(ctx, author, userID).Scan(&c.AuthorPublicID, &c.AuthorDisplayName); err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return &c, nil
}

func (r *CommentRepository) ListByVideo(ctx context.Context, videoID int64, limit, offset int32) ([]CommentRow, error) {
	const q = `
SELECT
  c.public_id,
  c.body,
  c.created_at,
  c.updated_at,
  u.public_id AS author_public_id,
  u.display_name AS author_display_name
FROM comments c
JOIN users u ON u.id = c.user_id
WHERE c.video_id = $1
  AND c.deleted_at IS NULL
ORDER BY c.created_at DESC
LIMIT $2
OFFSET $3;
`
	rows, err := r.pool.Query(ctx, q, videoID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []CommentRow
	for rows.Next() {
		var c CommentRow
		c.VideoID = videoID
		if err := rows.Scan(&c.PublicID, &c.Body, &c.CreatedAt, &c.UpdatedAt, &c.AuthorPublicID, &c.AuthorDisplayName); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

func (r *CommentRepository) CountByVideo(ctx context.Context, videoID int64) (int64, error) {
	const q = `
SELECT COUNT(*)
FROM comments
WHERE video_id = $1
  AND deleted_at IS NULL;
`
	var total int64
	if err := r.pool.QueryRow(ctx, q, videoID).Scan(&total); err != nil {
		return 0, err
	}
	return total, nil
}

func (r *CommentRepository) FindByPublicID(ctx context.Context, id uuid.UUID) (*CommentInternal, error) {
	const q = `
SELECT
  id,
  public_id,
  video_id,
  user_id,
  body,
  deleted_at
FROM comments
WHERE public_id = $1;
`
	var c CommentInternal
	err := r.pool.QueryRow(ctx, q, id).Scan(&c.ID, &c.PublicID, &c.VideoID, &c.UserID, &c.Body, &c.DeletedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &c, nil
}

func (r *CommentRepository) SoftDeleteWithCounter(ctx context.Context, publicID uuid.UUID) (bool, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return false, err
	}
	defer tx.Rollback(ctx)

	const q = `
WITH deleted_comment AS (
  UPDATE comments
  SET
    deleted_at = NOW(),
    updated_at = NOW()
  WHERE public_id = $1
    AND deleted_at IS NULL
  RETURNING video_id
)
UPDATE videos
SET comment_count = GREATEST(comment_count - 1, 0)
WHERE id IN (SELECT video_id FROM deleted_comment);
`
	tag, err := tx.Exec(ctx, q, publicID)
	if err != nil {
		return false, err
	}
	if err := tx.Commit(ctx); err != nil {
		return false, err
	}
	return tag.RowsAffected() > 0, nil
}
