package service

import (
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
)

const uploadTokenTTL = 15 * time.Minute

var ErrInvalidUploadToken = errors.New("invalid upload token")

// IssueUploadToken returns a short-lived token for direct multipart uploads to the API.
func (s *AuthService) IssueUploadToken(userID int64, role, secret string) (string, error) {
	if secret == "" {
		return "", fmt.Errorf("upload token secret is not configured")
	}
	exp := time.Now().UTC().Add(uploadTokenTTL).Unix()
	payload := fmt.Sprintf("%d:%s:%d", userID, role, exp)
	mac := hmacSHA256(secret, payload)
	return payload + ":" + mac, nil
}

// ResolveUploadToken validates a token and returns the authenticated principal.
func (s *AuthService) ResolveUploadToken(token, secret string) (userID int64, role string, err error) {
	if secret == "" || token == "" {
		return 0, "", ErrInvalidUploadToken
	}

	parts := strings.Split(token, ":")
	if len(parts) != 4 {
		return 0, "", ErrInvalidUploadToken
	}

	uid, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil || uid <= 0 {
		return 0, "", ErrInvalidUploadToken
	}
	role = parts[1]
	if role != "student" && role != "teacher" {
		return 0, "", ErrInvalidUploadToken
	}
	exp, err := strconv.ParseInt(parts[2], 10, 64)
	if err != nil || time.Now().UTC().Unix() > exp {
		return 0, "", ErrInvalidUploadToken
	}

	expected := hmacSHA256(secret, strings.Join(parts[:3], ":"))
	if subtle.ConstantTimeCompare([]byte(expected), []byte(parts[3])) != 1 {
		return 0, "", ErrInvalidUploadToken
	}

	return uid, role, nil
}

func hmacSHA256(secret, payload string) string {
	m := hmac.New(sha256.New, []byte(secret))
	_, _ = m.Write([]byte(payload))
	return hex.EncodeToString(m.Sum(nil))
}
