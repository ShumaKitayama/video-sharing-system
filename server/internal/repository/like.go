package repository

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type LikeRepository struct {
	pool *pgxpool.Pool
}

func NewLikeRepository(pool *pgxpool.Pool) *LikeRepository {
	return &LikeRepository{pool: pool}
}

func (r *LikeRepository) Exists(ctx context.Context, videoID, userID int64) (bool, error) {
	const q = `
SELECT EXISTS (
  SELECT 1
  FROM video_likes
  WHERE video_id = $1
    AND user_id = $2
) AS liked;
`
	var liked bool
	if err := r.pool.QueryRow(ctx, q, videoID, userID).Scan(&liked); err != nil {
		return false, err
	}
	return liked, nil
}

func (r *LikeRepository) Add(ctx context.Context, videoID, userID int64) (inserted bool, likeCount int64, err error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return false, 0, err
	}
	defer tx.Rollback(ctx)

	const ins = `
INSERT INTO video_likes (
  video_id,
  user_id
) VALUES (
  $1, $2
)
ON CONFLICT (video_id, user_id) DO NOTHING
RETURNING video_id;
`
	var dummy int64
	err = tx.QueryRow(ctx, ins, videoID, userID).Scan(&dummy)
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		inserted = false
	case err != nil:
		return false, 0, err
	default:
		inserted = true
	}

	if inserted {
		const inc = `
UPDATE videos
SET like_count = like_count + 1
WHERE id = $1
  AND deleted_at IS NULL;
`
		if _, err := tx.Exec(ctx, inc, videoID); err != nil {
			return false, 0, err
		}
	}

	const sel = `
SELECT like_count
FROM videos
WHERE id = $1
  AND deleted_at IS NULL;
`
	if err := tx.QueryRow(ctx, sel, videoID).Scan(&likeCount); err != nil {
		return false, 0, err
	}

	if err := tx.Commit(ctx); err != nil {
		return false, 0, err
	}
	return inserted, likeCount, nil
}

func (r *LikeRepository) Remove(ctx context.Context, videoID, userID int64) (removed bool, likeCount int64, err error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return false, 0, err
	}
	defer tx.Rollback(ctx)

	const del = `
DELETE FROM video_likes
WHERE video_id = $1
  AND user_id = $2
RETURNING video_id;
`
	var dummy int64
	err = tx.QueryRow(ctx, del, videoID, userID).Scan(&dummy)
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		removed = false
	case err != nil:
		return false, 0, err
	default:
		removed = true
	}

	if removed {
		const dec = `
UPDATE videos
SET like_count = GREATEST(like_count - 1, 0)
WHERE id = $1
  AND deleted_at IS NULL;
`
		if _, err := tx.Exec(ctx, dec, videoID); err != nil {
			return false, 0, err
		}
	}

	const sel = `
SELECT like_count
FROM videos
WHERE id = $1
  AND deleted_at IS NULL;
`
	if err := tx.QueryRow(ctx, sel, videoID).Scan(&likeCount); err != nil {
		return false, 0, err
	}

	if err := tx.Commit(ctx); err != nil {
		return false, 0, err
	}
	return removed, likeCount, nil
}
