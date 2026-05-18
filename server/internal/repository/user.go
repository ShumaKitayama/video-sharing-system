package repository

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// User represents a row returned for API-facing operations.
type User struct {
	ID           int64
	PublicID     uuid.UUID
	Username     string
	DisplayName  string
	PasswordHash string
	Role         string
	IsActive     bool
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

type UserPatch struct {
	DisplayName  *string
	PasswordHash *string
	Role         *string
	IsActive     *bool
}

type UserRepository struct {
	pool *pgxpool.Pool
}

func NewUserRepository(pool *pgxpool.Pool) *UserRepository {
	return &UserRepository{pool: pool}
}

func (r *UserRepository) Create(ctx context.Context, username, displayName, passwordHash, role string) (*User, error) {
	const q = `
INSERT INTO users (
  username,
  display_name,
  password_hash,
  role
) VALUES (
  $1, $2, $3, $4::user_role
)
RETURNING id, public_id, username, display_name, password_hash, role, is_active, created_at, updated_at;
`
	row := r.pool.QueryRow(ctx, q, username, displayName, passwordHash, role)
	var u User
	if err := row.Scan(&u.ID, &u.PublicID, &u.Username, &u.DisplayName, &u.PasswordHash, &u.Role, &u.IsActive, &u.CreatedAt, &u.UpdatedAt); err != nil {
		return nil, err
	}
	return &u, nil
}

func (r *UserRepository) FindByUsername(ctx context.Context, username string) (*User, error) {
	const q = `
SELECT
  id,
  public_id,
  username,
  display_name,
  password_hash,
  role,
  is_active,
  created_at,
  updated_at
FROM users
WHERE username = $1
  AND deleted_at IS NULL;
`
	var u User
	err := r.pool.QueryRow(ctx, q, username).Scan(
		&u.ID,
		&u.PublicID,
		&u.Username,
		&u.DisplayName,
		&u.PasswordHash,
		&u.Role,
		&u.IsActive,
		&u.CreatedAt,
		&u.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &u, nil
}

func (r *UserRepository) FindByPublicID(ctx context.Context, id uuid.UUID) (*User, error) {
	const q = `
SELECT
  id,
  public_id,
  username,
  display_name,
  password_hash,
  role,
  is_active,
  created_at,
  updated_at
FROM users
WHERE public_id = $1
  AND deleted_at IS NULL;
`
	var u User
	err := r.pool.QueryRow(ctx, q, id).Scan(
		&u.ID,
		&u.PublicID,
		&u.Username,
		&u.DisplayName,
		&u.PasswordHash,
		&u.Role,
		&u.IsActive,
		&u.CreatedAt,
		&u.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &u, nil
}

type UserListRow struct {
	PublicID    uuid.UUID
	Username    string
	DisplayName string
	Role        string
	IsActive    bool
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

func (r *UserRepository) List(ctx context.Context, role *string, active *bool, qtext *string, limit, offset int32) ([]UserListRow, error) {
	const base = `
SELECT
  public_id,
  username,
  display_name,
  role,
  is_active,
  created_at,
  updated_at
FROM users
WHERE deleted_at IS NULL
  AND ($1::user_role IS NULL OR role = $1)
  AND ($2::BOOLEAN IS NULL OR is_active = $2)
  AND (
    $3::TEXT IS NULL
    OR username ILIKE '%' || $3 || '%'
    OR display_name ILIKE '%' || $3 || '%'
  )
ORDER BY created_at DESC
LIMIT $4
OFFSET $5;
`
	rows, err := r.pool.Query(ctx, base, role, active, qtext, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []UserListRow
	for rows.Next() {
		var item UserListRow
		if err := rows.Scan(&item.PublicID, &item.Username, &item.DisplayName, &item.Role, &item.IsActive, &item.CreatedAt, &item.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	return out, rows.Err()
}

func (r *UserRepository) Count(ctx context.Context, role *string, active *bool, qtext *string) (int64, error) {
	const base = `
SELECT COUNT(*)
FROM users
WHERE deleted_at IS NULL
  AND ($1::user_role IS NULL OR role = $1)
  AND ($2::BOOLEAN IS NULL OR is_active = $2)
  AND (
    $3::TEXT IS NULL
    OR username ILIKE '%' || $3 || '%'
    OR display_name ILIKE '%' || $3 || '%'
  );
`
	var total int64
	if err := r.pool.QueryRow(ctx, base, role, active, qtext).Scan(&total); err != nil {
		return 0, err
	}
	return total, nil
}

func (r *UserRepository) Update(ctx context.Context, publicID uuid.UUID, patch UserPatch) (*UserListRow, error) {
	const q = `
UPDATE users
SET
  display_name = COALESCE($2, display_name),
  password_hash = COALESCE($3, password_hash),
  role = COALESCE($4::user_role, role),
  is_active = COALESCE($5, is_active),
  updated_at = NOW()
WHERE public_id = $1
  AND deleted_at IS NULL
RETURNING public_id, username, display_name, role, is_active, created_at, updated_at;
`
	row := r.pool.QueryRow(ctx, q,
		publicID,
		patch.DisplayName,
		patch.PasswordHash,
		patch.Role,
		patch.IsActive,
	)
	var out UserListRow
	err := row.Scan(&out.PublicID, &out.Username, &out.DisplayName, &out.Role, &out.IsActive, &out.CreatedAt, &out.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &out, nil
}

func (r *UserRepository) SoftDelete(ctx context.Context, publicID uuid.UUID) (bool, error) {
	const q = `
UPDATE users
SET
  deleted_at = NOW(),
  is_active = FALSE,
  updated_at = NOW()
WHERE public_id = $1
  AND deleted_at IS NULL;
`
	tag, err := r.pool.Exec(ctx, q, publicID)
	if err != nil {
		return false, err
	}
	return tag.RowsAffected() > 0, nil
}
