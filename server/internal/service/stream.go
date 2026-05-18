package service

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/google/uuid"

	"video-sharing-system/server/internal/repository"
	"video-sharing-system/server/internal/storage"
	"video-sharing-system/server/internal/streaming"
)

type StreamService struct {
	videos *repository.VideoRepository
	store  *storage.VideoStorage

	mu   sync.Mutex
	last map[string]time.Time
}

func NewStreamService(videos *repository.VideoRepository, store *storage.VideoStorage) *StreamService {
	return &StreamService{
		videos: videos,
		store:  store,
		last:   map[string]time.Time{},
	}
}

func streamClientKey(viewerUserID int64, remoteIP string) string {
	if viewerUserID != 0 {
		return fmt.Sprintf("u:%d", viewerUserID)
	}
	return "ip:" + remoteIP
}

func (s *StreamService) maybeBumpView(internalVideoID int64, clientKey string) {
	key := fmt.Sprintf("%d:%s", internalVideoID, clientKey)

	s.mu.Lock()
	now := time.Now()
	if prev, ok := s.last[key]; ok && now.Sub(prev) < 12*time.Second {
		s.mu.Unlock()
		return
	}
	s.last[key] = now
	s.mu.Unlock()

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	_ = s.videos.IncrementViewCount(ctx, internalVideoID)
}

// ServeHTTP validates access and streams bytes using Range-safe primitives.
func (s *StreamService) ServeHTTP(ctx context.Context, publicID uuid.UUID, viewerUserID int64, viewerRole string, remoteIP string, w http.ResponseWriter, req *http.Request) error {
	info, err := s.videos.FindStorageInfo(ctx, publicID)
	if err != nil {
		return err
	}
	if info == nil {
		return ErrNotFound
	}

	if !CanAccessVideo(info.Status, info.UploaderID, viewerUserID, viewerRole) {
		return ErrNotFound
	}

	s.maybeBumpView(info.ID, streamClientKey(viewerUserID, remoteIP))

	if err := streaming.ServeFile(w, req, s.store, info.StorageKey, info.UpdatedAt); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return ErrNotFound
		}
		return err
	}
	return nil
}