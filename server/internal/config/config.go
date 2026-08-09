package config

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"strconv"
	"strings"
)

// Config holds runtime configuration loaded from environment variables.
type Config struct {
	Port               string
	DatabaseURL        string
	UploadDir          string
	MigrationsDir      string
	CORSOrigins        []string
	CookieSecure       bool
	CookieSameSite     string // lax / strict / none
	BlobReadWriteToken string
	UploadTokenSecret  string

	// Cloudflare R2 holds the video files in the internet-facing deployment.
	// Leave these empty to keep videos on local disk (classroom LAN / Docker).
	R2AccountID       string
	R2AccessKeyID     string
	R2SecretAccessKey string
	R2Bucket          string
	R2PublicBaseURL   string
}

// Load reads configuration from the environment with LAN-friendly defaults.
func Load() Config {
	port := getenv("PORT", "8080")
	originsStr := getenv("CORS_ORIGINS", "http://localhost:5173,http://127.0.0.1:5173")
	var origins []string
	for _, o := range strings.Split(originsStr, ",") {
		o = strings.TrimSpace(o)
		if o != "" {
			origins = append(origins, o)
		}
	}
	if len(origins) == 0 {
		origins = []string{"http://localhost:5173"}
	}

	secure, _ := strconv.ParseBool(getenv("COOKIE_SECURE", "false"))
	dbURL := getenv("DATABASE_URL", "postgres://postgres:postgres@localhost:5432/videoshare?sslmode=disable")

	return Config{
		Port:               port,
		DatabaseURL:        dbURL,
		UploadDir:          getenv("UPLOAD_DIR", "./uploads"),
		MigrationsDir:      getenv("MIGRATIONS_DIR", "./migrations"),
		CORSOrigins:        origins,
		CookieSecure:       secure,
		CookieSameSite:     strings.ToLower(getenv("COOKIE_SAMESITE", "lax")),
		BlobReadWriteToken: os.Getenv("BLOB_READ_WRITE_TOKEN"),
		UploadTokenSecret:  uploadTokenSecret(dbURL),

		R2AccountID:       os.Getenv("R2_ACCOUNT_ID"),
		R2AccessKeyID:     os.Getenv("R2_ACCESS_KEY_ID"),
		R2SecretAccessKey: os.Getenv("R2_SECRET_ACCESS_KEY"),
		R2Bucket:          os.Getenv("R2_BUCKET"),
		R2PublicBaseURL:   os.Getenv("R2_PUBLIC_BASE_URL"),
	}
}

func uploadTokenSecret(dbURL string) string {
	if secret := os.Getenv("UPLOAD_TOKEN_SECRET"); secret != "" {
		return secret
	}
	sum := sha256.Sum256([]byte("upload-token:" + dbURL))
	return hex.EncodeToString(sum[:])
}

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
