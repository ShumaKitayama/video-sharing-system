package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"video-sharing-system/server/internal/api"
	"video-sharing-system/server/internal/config"
	"video-sharing-system/server/internal/db"
	"video-sharing-system/server/internal/ratelimit"
	"video-sharing-system/server/internal/repository"
	"video-sharing-system/server/internal/router"
	"video-sharing-system/server/internal/service"
	"video-sharing-system/server/internal/storage"
)

func main() {
	cfg := config.Load()

	ctx := context.Background()
	pool, err := db.NewPool(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("database: %v", err)
	}
	defer pool.Close()

	if err := db.ApplyMigrations(ctx, pool, cfg.MigrationsDir); err != nil {
		log.Fatalf("migrations: %v", err)
	}

	// nil means "no R2 settings present", which keeps videos on local disk.
	r2Client, err := storage.NewR2Client(storage.R2Config{
		AccountID:       cfg.R2AccountID,
		AccessKeyID:     cfg.R2AccessKeyID,
		SecretAccessKey: cfg.R2SecretAccessKey,
		Bucket:          cfg.R2Bucket,
		PublicBaseURL:   cfg.R2PublicBaseURL,
	})
	if err != nil {
		log.Fatalf("storage: %v", err)
	}
	if r2Client != nil {
		log.Printf("video storage: Cloudflare R2 bucket %q", cfg.R2Bucket)
	} else {
		log.Printf("video storage: local disk %q", cfg.UploadDir)
	}

	store := storage.NewVideoStorage(cfg.UploadDir, cfg.BlobReadWriteToken, r2Client)

	userRepo := repository.NewUserRepository(pool)
	sessRepo := repository.NewSessionRepository(pool)
	videoRepo := repository.NewVideoRepository(pool)
	commentRepo := repository.NewCommentRepository(pool)
	likeRepo := repository.NewLikeRepository(pool)

	authSvc := service.NewAuthService(userRepo, sessRepo)
	userSvc := service.NewUserService(userRepo, sessRepo)
	videoSvc := service.NewVideoService(videoRepo, store)
	streamSvc := service.NewStreamService(videoRepo, store)
	commentSvc := service.NewCommentService(commentRepo, videoSvc)
	likeSvc := service.NewLikeService(likeRepo, videoSvc)

	engine := router.New(cfg, api.Deps{
		Pool:      pool,
		Store:     store,
		Auth:      authSvc,
		Users:     userSvc,
		Videos:    videoSvc,
		Stream:    streamSvc,
		Comments:  commentSvc,
		Likes:     likeSvc,
		LoginRL:   ratelimit.New(time.Minute, 10),
		UploadRL:  ratelimit.New(time.Minute, 5),
		CommentRL: ratelimit.New(time.Minute, 30),
	})

	go func() {
		ticker := time.NewTicker(10 * time.Minute)
		defer ticker.Stop()
		for range ticker.C {
			cctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			n, err := sessRepo.DeleteExpired(cctx)
			cancel()
			if err != nil {
				log.Printf("session cleanup error: %v", err)
				continue
			}
			if n > 0 {
				log.Printf("session cleanup: removed %d expired sessions", n)
			}
		}
	}()

	// WriteTimeout caps hung slow clients; large enough for LAN classroom streaming (Range requests).
	const streamWriteTimeout = 45 * time.Minute

	srv := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      engine,
		ReadTimeout:  60 * time.Second,
		WriteTimeout: streamWriteTimeout,
		IdleTimeout:  120 * time.Second,
	}

	go func() {
		log.Printf("API listening on http://localhost:%s", cfg.Port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Printf("shutdown: %v", err)
	}
}
