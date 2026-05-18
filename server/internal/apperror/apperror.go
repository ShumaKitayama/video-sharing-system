package apperror

// Code mirrors api-design.md error codes.
type Code string

const (
	ValidationError       Code = "VALIDATION_ERROR"
	Unauthenticated       Code = "UNAUTHENTICATED"
	Forbidden             Code = "FORBIDDEN"
	NotFound              Code = "NOT_FOUND"
	Conflict              Code = "CONFLICT"
	PayloadTooLarge       Code = "PAYLOAD_TOO_LARGE"
	UnsupportedMediaType  Code = "UNSUPPORTED_MEDIA_TYPE"
	RateLimited           Code = "RATE_LIMITED"
	InternalError         Code = "INTERNAL_ERROR"
	ServiceUnavailable    Code = "SERVICE_UNAVAILABLE"
)

// FieldDetail describes a single validation issue.
type FieldDetail struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

// APIError is returned as JSON under "error".
type APIError struct {
	Code      Code          `json:"code"`
	Message   string        `json:"message"`
	Details   []FieldDetail `json:"details,omitempty"`
	RequestID string        `json:"request_id"`
}
