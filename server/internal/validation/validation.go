package validation

import (
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"unicode/utf8"

	"video-sharing-system/server/internal/apperror"
)

var usernameRe = regexp.MustCompile(`^[a-zA-Z0-9_]{3,32}$`)

func Field(field, msg string) apperror.FieldDetail {
	return apperror.FieldDetail{Field: field, Message: msg}
}

func Username(username string) *apperror.FieldDetail {
	if !usernameRe.MatchString(username) {
		f := Field("username", "3〜32文字の英数字とアンダースコアのみ利用できます")
		return &f
	}
	return nil
}

func DisplayName(displayName string) *apperror.FieldDetail {
	s := strings.TrimSpace(displayName)
	if utf8.RuneCountInString(s) < 1 || utf8.RuneCountInString(s) > 40 {
		f := Field("display_name", "1〜40文字で入力してください")
		return &f
	}
	return nil
}

func Password(password string) *apperror.FieldDetail {
	n := len(password)
	if n < 8 || n > 72 {
		f := Field("password", "8〜72文字で入力してください")
		return &f
	}
	return nil
}

// ParseDurationSeconds parses optional upload metadata from multipart form.
func ParseDurationSeconds(raw string) (*int32, *apperror.FieldDetail) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, nil
	}
	n, err := strconv.ParseInt(raw, 10, 32)
	if err != nil || n < 0 {
		f := Field("duration_seconds", "0以上の整数で指定してください")
		return nil, &f
	}
	if n > 86400 {
		f := Field("duration_seconds", "86400秒以下で指定してください")
		return nil, &f
	}
	v := int32(n)
	return &v, nil
}

func Title(title string) *apperror.FieldDetail {
	s := strings.TrimSpace(title)
	if utf8.RuneCountInString(s) < 1 || utf8.RuneCountInString(s) > 80 {
		f := Field("title", "1〜80文字で入力してください")
		return &f
	}
	return nil
}

func Description(description string) *apperror.FieldDetail {
	if utf8.RuneCountInString(description) > 1000 {
		f := Field("description", "1000文字以内で入力してください")
		return &f
	}
	return nil
}

func CommentBody(body string) *apperror.FieldDetail {
	s := strings.TrimSpace(body)
	if utf8.RuneCountInString(s) < 1 || utf8.RuneCountInString(s) > 300 {
		f := Field("body", "1〜300文字で入力してください")
		return &f
	}
	return nil
}

func ParseUserRole(raw string) (string, *apperror.FieldDetail) {
	switch strings.TrimSpace(strings.ToLower(raw)) {
	case "student":
		return "student", nil
	case "teacher":
		return "teacher", nil
	default:
		f := Field("role", "student または teacher を指定してください")
		return "", &f
	}
}

func ParseVideoStatus(raw string) (string, *apperror.FieldDetail) {
	switch strings.TrimSpace(strings.ToLower(raw)) {
	case "published":
		return "published", nil
	case "hidden":
		return "hidden", nil
	default:
		f := Field("status", "published または hidden を指定してください")
		return "", &f
	}
}

func ParseVideoSort(raw string) string {
	switch strings.TrimSpace(strings.ToLower(raw)) {
	case "oldest", "most_viewed", "most_liked":
		return strings.TrimSpace(strings.ToLower(raw))
	default:
		return "latest"
	}
}

// ParseBoolQueryParam parses optional boolean query param.
func ParseBoolQueryParam(raw string) (*bool, *apperror.FieldDetail) {
	raw = strings.TrimSpace(strings.ToLower(raw))
	if raw == "" {
		return nil, nil
	}
	switch raw {
	case "true":
		v := true
		return &v, nil
	case "false":
		v := false
		return &v, nil
	default:
		f := Field("is_active", "true または false を指定してください")
		return nil, &f
	}
}

func ParseVideoListStatusFilter(raw string, teacher bool) (*string, []apperror.FieldDetail) {
	raw = strings.TrimSpace(strings.ToLower(raw))
	if raw == "" {
		return nil, nil
	}
	switch raw {
	case "published", "hidden":
		s := raw
		return &s, nil
	case "all":
		if !teacher {
			return nil, []apperror.FieldDetail{Field("status", "この絞り込みは先生のみ利用できます")}
		}
		s := "all"
		return &s, nil
	default:
		return nil, []apperror.FieldDetail{Field("status", "published, hidden, all のいずれかを指定してください")}
	}
}

func IsAllowedVideoMIME(mime string) bool {
	m := strings.TrimSpace(strings.ToLower(mime))
	return m == "video/mp4" || m == "video/webm"
}

// FilenameMatchesMIME ensures extension aligns with declared MIME (api-design 14.1).
func FilenameMatchesMIME(filename string, mime string) bool {
	ext := strings.ToLower(filepath.Ext(strings.TrimSpace(filename)))
	switch strings.TrimSpace(strings.ToLower(mime)) {
	case "video/mp4":
		return ext == ".mp4"
	case "video/webm":
		return ext == ".webm"
	default:
		return false
	}
}

func ExtForMIME(mime string) string {
	switch strings.TrimSpace(strings.ToLower(mime)) {
	case "video/mp4":
		return ".mp4"
	case "video/webm":
		return ".webm"
	default:
		return ""
	}
}
