package service

import "errors"

var (
	ErrConflict          = errors.New("conflict")
	ErrNotFound          = errors.New("not_found")
	ErrForbidden         = errors.New("forbidden")
	ErrPayloadTooLarge   = errors.New("payload_too_large")
	ErrUnsupportedMedia    = errors.New("unsupported_media")
	ErrInvalidBlobURL      = errors.New("invalid_blob_url")
	ErrInvalidCredentials  = errors.New("invalid_credentials")
	ErrInactiveUser        = errors.New("inactive_user")
)
