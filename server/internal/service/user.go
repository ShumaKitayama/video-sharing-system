package service

import (
	"context"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"video-sharing-system/server/internal/pgutil"
	"video-sharing-system/server/internal/repository"
)

type UserService struct {
	repo     *repository.UserRepository
	sessions *repository.SessionRepository
}

func NewUserService(repo *repository.UserRepository, sessions *repository.SessionRepository) *UserService {
	return &UserService{repo: repo, sessions: sessions}
}

type UserDTO struct {
	ID          uuid.UUID `json:"id"`
	Username    string    `json:"username"`
	DisplayName string    `json:"display_name"`
	Role        string    `json:"role"`
	IsActive    bool      `json:"is_active"`
	CreatedAt   string    `json:"created_at,omitempty"`
	UpdatedAt   string    `json:"updated_at,omitempty"`
}

func dtoFromList(row repository.UserListRow) UserDTO {
	return UserDTO{
		ID:          row.PublicID,
		Username:    row.Username,
		DisplayName: row.DisplayName,
		Role:        row.Role,
		IsActive:    row.IsActive,
		CreatedAt:   formatRFC3339(row.CreatedAt),
		UpdatedAt:   formatRFC3339(row.UpdatedAt),
	}
}

type UserDTOCompact struct {
	ID          uuid.UUID `json:"id"`
	Username    string    `json:"username"`
	DisplayName string    `json:"display_name"`
	Role        string    `json:"role"`
	IsActive    bool      `json:"is_active,omitempty"`
}

func (s *UserService) CreateTeacherUser(ctx context.Context, username, displayName, password, role string) (UserDTOCompact, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), 12)
	if err != nil {
		return UserDTOCompact{}, err
	}
	u, err := s.repo.Create(ctx, username, displayName, string(hash), role)
	if err != nil {
		if pgutil.IsUniqueViolation(err) {
			return UserDTOCompact{}, ErrConflict
		}
		return UserDTOCompact{}, err
	}
	return UserDTOCompact{
		ID:          u.PublicID,
		Username:    u.Username,
		DisplayName: u.DisplayName,
		Role:        u.Role,
		IsActive:    u.IsActive,
	}, nil
}

func (s *UserService) ListUsers(ctx context.Context, role *string, active *bool, q *string, page, perPage int) ([]UserDTO, int64, error) {
	offset := int64((page - 1) * perPage)
	items, err := s.repo.List(ctx, role, active, q, int32(perPage), int32(offset))
	if err != nil {
		return nil, 0, err
	}
	total, err := s.repo.Count(ctx, role, active, q)
	if err != nil {
		return nil, 0, err
	}
	out := make([]UserDTO, 0, len(items))
	for _, row := range items {
		out = append(out, dtoFromList(row))
	}
	return out, total, nil
}

func (s *UserService) GetUser(ctx context.Context, target uuid.UUID, actorRole string, actorPublic uuid.UUID) (UserDTO, error) {
	if actorRole != "teacher" && target != actorPublic {
		return UserDTO{}, ErrForbidden
	}

	rowFull, err := s.repo.FindByPublicID(ctx, target)
	if err != nil {
		return UserDTO{}, err
	}
	if rowFull == nil {
		return UserDTO{}, ErrNotFound
	}

	listLike := repository.UserListRow{
		PublicID:    rowFull.PublicID,
		Username:    rowFull.Username,
		DisplayName: rowFull.DisplayName,
		Role:        rowFull.Role,
		IsActive:    rowFull.IsActive,
		CreatedAt:   rowFull.CreatedAt,
		UpdatedAt:   rowFull.UpdatedAt,
	}
	return dtoFromList(listLike), nil
}

type PatchUserInput struct {
	DisplayName *string
	Password    *string // plaintext optional update

	Role     *string // teacher-only fields below
	IsActive *bool
}

func (s *UserService) PatchUser(ctx context.Context, target uuid.UUID, actorRole string, actorPublic uuid.UUID, in PatchUserInput) (UserDTO, error) {
	isTeacher := actorRole == "teacher"
	isSelf := target == actorPublic

	if !isSelf && !isTeacher {
		return UserDTO{}, ErrForbidden
	}
	if !isTeacher && (in.Role != nil || in.IsActive != nil) {
		return UserDTO{}, ErrForbidden
	}

	rowFull, err := s.repo.FindByPublicID(ctx, target)
	if err != nil {
		return UserDTO{}, err
	}
	if rowFull == nil {
		return UserDTO{}, ErrNotFound
	}

	var patch repository.UserPatch
	if in.DisplayName != nil {
		v := *in.DisplayName
		patch.DisplayName = &v
	}

	if in.Password != nil {
		hashBytes, err := bcrypt.GenerateFromPassword([]byte(*in.Password), 12)
		if err != nil {
			return UserDTO{}, err
		}
		h := string(hashBytes)
		patch.PasswordHash = &h
	}

	if isTeacher {
		if in.Role != nil {
			r := *in.Role
			patch.Role = &r
		}
		if in.IsActive != nil {
			patch.IsActive = in.IsActive
		}
	}

	outRow, err := s.repo.Update(ctx, target, patch)
	if err != nil {
		if pgutil.IsUniqueViolation(err) {
			return UserDTO{}, ErrConflict
		}
		return UserDTO{}, err
	}
	if outRow == nil {
		return UserDTO{}, ErrNotFound
	}

	// Invalidate sessions when passwords or privilege-bearing fields change.
	if in.Password != nil || (isTeacher && (in.Role != nil || in.IsActive != nil)) {
		_ = s.sessions.DeleteByUserID(ctx, rowFull.ID)
	}

	return dtoFromList(*outRow), nil
}

func (s *UserService) SoftDeleteUser(ctx context.Context, target uuid.UUID, actorRole string) error {
	if actorRole != "teacher" {
		return ErrForbidden
	}

	u, err := s.repo.FindByPublicID(ctx, target)
	if err != nil {
		return err
	}
	if u == nil {
		return ErrNotFound
	}
	if err := s.sessions.DeleteByUserID(ctx, u.ID); err != nil {
		return err
	}

	ok, err := s.repo.SoftDelete(ctx, target)
	if err != nil {
		return err
	}
	if !ok {
		return ErrNotFound
	}
	return nil
}
