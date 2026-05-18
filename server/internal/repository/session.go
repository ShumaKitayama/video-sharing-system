package repository

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type SessionRepository struct {
	pool *pgxpool.Pool
}

func NewSessionRepository(pool *pgxpool.Pool) *SessionRepository {
	return &SessionRepository{pool: pool}
}

type SessionWithUser struct {
	SessionID int64
	UserID    int64
	ExpiresAt time.Time

	PublicID    uuid.UUID
	Username    string
	DisplayName string
	Role        string
	IsActive    bool
}

func (r *SessionRepository) Create(ctx context.Context, userID int64, tokenHash string, expiresAt time.Time) (int64, error) {
	const q = `
INSERT INTO sessions (
  user_id,
  token_hash,
  expires_at
) VALUES (
  $1, $2, $3
)
RETURNING id;
`
	var id int64
	if err := r.pool.QueryRow(ctx, q, userID, tokenHash, expiresAt).Scan(&id); err != nil {
		return 0, err
	}
	return id, nil
}

func (r *SessionRepository) FindActiveWithUser(ctx context.Context, tokenHash string) (*SessionWithUser, error) {
	const q = `
SELECT
  s.id AS session_id,
  s.user_id,
  s.expires_at,
  u.public_id,
  u.username,
  u.display_name,
  u.role,
  u.is_active
FROM sessions s
JOIN users u ON u.id = s.user_id
WHERE s.token_hash = $1
  AND s.expires_at > NOW()
  AND u.deleted_at IS NULL
  AND u.is_active = TRUE;
`
	var s SessionWithUser
	err := r.pool.QueryRow(ctx, q, tokenHash).Scan(
		&s.SessionID,
		&s.UserID,
		&s.ExpiresAt,
		&s.PublicID,
		&s.Username,
		&s.DisplayName,
		&s.Role,
		&s.IsActive,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &s, nil
}

func (r *SessionRepository) Touch(ctx context.Context, sessionID int64) error {
	const q = `
UPDATE sessions
SET last_seen_at = NOW()
WHERE id = $1;
`
	_, err := r.pool.Exec(ctx, q, sessionID)
	return err
}

func (r *SessionRepository) DeleteByTokenHash(ctx context.Context, tokenHash string) error {
	const q = `
DELETE FROM sessions
WHERE token_hash = $1;
`
	_, err := r.pool.Exec(ctx, q, tokenHash)
	return err
}

func (r *SessionRepository) DeleteByUserID(ctx context.Context, userID int64) error {
	const q = `
DELETE FROM sessions
WHERE user_id = $1;
`
	_, err := r.pool.Exec(ctx, q, userID)
	return err
}

func (r *SessionRepository) DeleteExpired(ctx context.Context) (int64, error) {
	const q = `
DELETE FROM sessions
WHERE expires_at <= NOW();
`
	tag, err := r.pool.Exec(ctx, q)
	if err != nil {
		return 0, err
	}
	return tag.RowsAffected(), nil
}
