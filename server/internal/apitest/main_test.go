package apitest

// Package apitest holds black-box style integration tests that exercise the
// real Gin router against a real PostgreSQL database and a temporary upload
// directory. These tests verify three things end to end:
//
//  1. The HTTP API behaves as specified in back/api-design.md.
//  2. Rows are actually written to / read from PostgreSQL.
//  3. Uploaded video bytes are persisted to disk and served back via Range.
//
// The tests are skipped automatically when TEST_DATABASE_URL is not set, so a
// plain `go test ./...` without a database still passes.

import (
	"context"
	"log"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"video-sharing-system/server/internal/api"
	"video-sharing-system/server/internal/config"
	"video-sharing-system/server/internal/db"
	"video-sharing-system/server/internal/ratelimit"
	"video-sharing-system/server/internal/repository"
	"video-sharing-system/server/internal/router"
	"video-sharing-system/server/internal/service"
	"video-sharing-system/server/internal/storage"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Shared test fixtures, initialised once in TestMain.
var (
	testPool   *pgxpool.Pool
	testServer *httptest.Server
	uploadDir  string
)

func TestMain(m *testing.M) {
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		// No database configured: run anyway so individual tests can t.Skip().
		os.Exit(m.Run())
	}

	gin.SetMode(gin.TestMode)

	ctx := context.Background()
	pool, err := db.NewPool(ctx, dsn)
	if err != nil {
		log.Fatalf("apitest: connect database: %v", err)
	}
	// Fail fast if the DB is not actually reachable.
	pingCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	if err := pool.Ping(pingCtx); err != nil {
		cancel()
		log.Fatalf("apitest: ping database: %v", err)
	}
	cancel()
	testPool = pool

	migrationsDir := os.Getenv("MIGRATIONS_DIR")
	if migrationsDir == "" {
		// Default to the repository layout: server/internal/apitest -> server/migrations.
		migrationsDir = "../../migrations"
	}
	if err := db.ApplyMigrations(ctx, pool, migrationsDir); err != nil {
		log.Fatalf("apitest: apply migrations: %v", err)
	}

	dir, err := os.MkdirTemp("", "apitest-uploads-*")
	if err != nil {
		log.Fatalf("apitest: create temp upload dir: %v", err)
	}
	uploadDir = dir

	testServer = newTestServer(uploadDir)

	code := m.Run()

	testServer.Close()
	pool.Close()
	_ = os.RemoveAll(dir)
	os.Exit(code)
}

// newTestServer wires the production router with test-friendly settings:
// generous rate limits (so repeated logins/uploads in tests do not trip 429)
// and the given upload directory.
func newTestServer(dir string) *httptest.Server {
	cfg := config.Config{
		Port:           "0",
		UploadDir:      dir,
		MigrationsDir:  "",
		CORSOrigins:    []string{"http://localhost:5173"},
		CookieSecure:   false,
		CookieSameSite: "lax",
	}

	store := storage.NewVideoStorage(dir)

	userRepo := repository.NewUserRepository(testPool)
	sessRepo := repository.NewSessionRepository(testPool)
	videoRepo := repository.NewVideoRepository(testPool)
	commentRepo := repository.NewCommentRepository(testPool)
	likeRepo := repository.NewLikeRepository(testPool)

	authSvc := service.NewAuthService(userRepo, sessRepo)
	userSvc := service.NewUserService(userRepo, sessRepo)
	videoSvc := service.NewVideoService(videoRepo, store)
	streamSvc := service.NewStreamService(videoRepo, store)
	commentSvc := service.NewCommentService(commentRepo, videoSvc)
	likeSvc := service.NewLikeService(likeRepo, videoSvc)

	const generous = 1_000_000
	engine := router.New(cfg, api.Deps{
		Pool:      testPool,
		Store:     store,
		Auth:      authSvc,
		Users:     userSvc,
		Videos:    videoSvc,
		Stream:    streamSvc,
		Comments:  commentSvc,
		Likes:     likeSvc,
		LoginRL:   ratelimit.New(time.Minute, generous),
		UploadRL:  ratelimit.New(time.Minute, generous),
		CommentRL: ratelimit.New(time.Minute, generous),
	})

	return httptest.NewServer(engine)
}

// requireDB skips the calling test when no database is configured.
func requireDB(t *testing.T) {
	t.Helper()
	if testPool == nil {
		t.Skip("TEST_DATABASE_URL is not set; skipping integration test")
	}
}

// resetDB truncates all mutable tables so each test starts from a clean slate.
// RESTART IDENTITY keeps serial ids deterministic; CASCADE clears dependents.
func resetDB(t *testing.T) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_, err := testPool.Exec(ctx, `TRUNCATE users, sessions, videos, comments, video_likes RESTART IDENTITY CASCADE`)
	if err != nil {
		t.Fatalf("resetDB: truncate failed: %v", err)
	}
}

// setupTest is the standard per-test preamble: require a DB and wipe state.
func setupTest(t *testing.T) {
	t.Helper()
	requireDB(t)
	resetDB(t)
}
