package service

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"video-sharing-system/server/internal/pgutil"
	"video-sharing-system/server/internal/repository"
)

const sessionDuration = 24 * time.Hour

type AuthService struct {
	users    *repository.UserRepository
	sessions *repository.SessionRepository
}

func NewAuthService(users *repository.UserRepository, sessions *repository.SessionRepository) *AuthService {
	return &AuthService{
		users:    users,
		sessions: sessions,
	}
}

type AuthUser struct {
	ID          uuid.UUID `json:"id"`
	Username    string    `json:"username"`
	DisplayName string    `json:"display_name"`
	Role        string    `json:"role"`
}

type SessionPrincipal struct {
	SessionID int64
	UserID    int64
	PublicID  uuid.UUID
	Username  string
	Role      string
}

func hashSessionToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

func randomSessionToken() (string, error) {
	var b [32]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	return hex.EncodeToString(b[:]), nil
}

func (s *AuthService) Register(ctx context.Context, username, displayName, password string) (cookieToken string, user AuthUser, err error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), 12)
	if err != nil {
		return "", AuthUser{}, err
	}

	u, err := s.users.Create(ctx, username, displayName, string(hash), "student")
	if err != nil {
		if pgutil.IsUniqueViolation(err) {
			return "", AuthUser{}, fmt.Errorf("%w", ErrConflict)
		}
		return "", AuthUser{}, err
	}

	token, err := randomSessionToken()
	if err != nil {
		return "", AuthUser{}, err
	}
	expires := time.Now().UTC().Add(sessionDuration)
	if _, err := s.sessions.Create(ctx, u.ID, hashSessionToken(token), expires); err != nil {
		return "", AuthUser{}, err
	}

	user = AuthUser{
		ID:          u.PublicID,
		Username:    u.Username,
		DisplayName: u.DisplayName,
		Role:        u.Role,
	}
	return token, user, nil
}

func (s *AuthService) Login(ctx context.Context, username, password string) (cookieToken string, user AuthUser, err error) {
	u, err := s.users.FindByUsername(ctx, username)
	if err != nil {
		return "", AuthUser{}, err
	}
	if u == nil {
		return "", AuthUser{}, fmt.Errorf("%w", ErrInvalidCredentials)
	}
	if !u.IsActive {
		return "", AuthUser{}, fmt.Errorf("%w", ErrInactiveUser)
	}

	if err := bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(password)); err != nil {
		return "", AuthUser{}, fmt.Errorf("%w", ErrInvalidCredentials)
	}

	token, err := randomSessionToken()
	if err != nil {
		return "", AuthUser{}, err
	}
	expires := time.Now().UTC().Add(sessionDuration)
	if _, err := s.sessions.Create(ctx, u.ID, hashSessionToken(token), expires); err != nil {
		return "", AuthUser{}, err
	}

	user = AuthUser{
		ID:          u.PublicID,
		Username:    u.Username,
		DisplayName: u.DisplayName,
		Role:        u.Role,
	}
	return token, user, nil
}

func (s *AuthService) Logout(ctx context.Context, cookieToken string) error {
	if cookieToken == "" {
		return nil
	}
	return s.sessions.DeleteByTokenHash(ctx, hashSessionToken(cookieToken))
}

func (s *AuthService) ResolveSession(ctx context.Context, cookieToken string) (*SessionPrincipal, error) {
	if cookieToken == "" {
		return nil, nil
	}
	su, err := s.sessions.FindActiveWithUser(ctx, hashSessionToken(cookieToken))
	if err != nil {
		return nil, err
	}
	if su == nil {
		return nil, nil
	}

	go func(sessionID int64) {
		cctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = s.sessions.Touch(cctx, sessionID)
	}(su.SessionID)

	return &SessionPrincipal{
		SessionID: su.SessionID,
		UserID:    su.UserID,
		PublicID:  su.PublicID,
		Username:  su.Username,
		Role:      su.Role,
	}, nil
}

func (s *AuthService) GetAuthUser(ctx context.Context, publicID uuid.UUID) (*AuthUser, error) {
	u, err := s.users.FindByPublicID(ctx, publicID)
	if err != nil {
		return nil, err
	}
	if u == nil {
		return nil, nil
	}
	out := &AuthUser{
		ID:          u.PublicID,
		Username:    u.Username,
		DisplayName: u.DisplayName,
		Role:        u.Role,
	}
	return out, nil
}
