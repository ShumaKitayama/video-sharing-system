package config

import (
	"os"
	"strconv"
	"strings"
)

// Config holds runtime configuration loaded from environment variables.
type Config struct {
	Port           string
	DatabaseURL    string
	UploadDir      string
	MigrationsDir  string
	CORSOrigins    []string
	CookieSecure   bool
	CookieSameSite string // lax / strict / none
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

	return Config{
		Port:           port,
		DatabaseURL:    getenv("DATABASE_URL", "postgres://postgres:postgres@localhost:5432/videoshare?sslmode=disable"),
		UploadDir:      getenv("UPLOAD_DIR", "./uploads"),
		MigrationsDir:  getenv("MIGRATIONS_DIR", "./migrations"),
		CORSOrigins:    origins,
		CookieSecure:   secure,
		CookieSameSite: strings.ToLower(getenv("COOKIE_SAMESITE", "lax")),
	}
}

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
