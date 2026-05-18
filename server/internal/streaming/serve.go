package streaming

import (
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"video-sharing-system/server/internal/storage"
)

// ServeFile streams file with Range support without buffering entire file.
func ServeFile(w http.ResponseWriter, req *http.Request, store *storage.VideoStorage, storageKey string, modTime time.Time) error {
	path := store.AbsolutePath(storageKey)
	root := filepath.Clean(store.Root())
	target := filepath.Clean(path)
	rel, err := filepath.Rel(root, target)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return errors.New("invalid storage path")
	}

	f, err := os.Open(target)
	if err != nil {
		return err
	}
	defer f.Close()

	stat, err := f.Stat()
	if err != nil {
		return err
	}

	http.ServeContent(w, req, stat.Name(), modTime.UTC(), f)
	return nil
}
