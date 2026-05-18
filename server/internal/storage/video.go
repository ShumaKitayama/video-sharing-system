package storage

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"

	"video-sharing-system/server/internal/validation"
)

const copyBufferSize = 32 * 1024
const maxVideoBytes = 500 * 1024 * 1024

type VideoStorage struct {
	root string
}

func NewVideoStorage(root string) *VideoStorage {
	return &VideoStorage{root: root}
}

func (s *VideoStorage) Root() string {
	return s.root
}

// AbsolutePath resolves a relative storage key under root.
func (s *VideoStorage) AbsolutePath(storageKey string) string {
	clean := filepath.Clean(storageKey)
	return filepath.Join(s.root, clean)
}

// PrepareWritableDir ensures root exists and is writable.
func (s *VideoStorage) PrepareWritableDir(ctx context.Context) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}
	if err := os.MkdirAll(s.root, 0o755); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(s.root, ".writetest-*")
	if err != nil {
		return err
	}
	path := tmp.Name()
	_ = tmp.Close()
	return os.Remove(path)
}

// SaveUploadedVideo streams src into a new UUID-named object under videos/YYYY/MM/.
func (s *VideoStorage) SaveUploadedVideo(ctx context.Context, src io.Reader, mime string, originalFilename string) (storageKey string, written int64, err error) {
	_ = originalFilename

	select {
	case <-ctx.Done():
		return "", 0, ctx.Err()
	default:
	}

	ext := validation.ExtForMIME(mime)
	if ext == "" {
		return "", 0, fmt.Errorf("unsupported mime")
	}

	now := time.Now().UTC()
	relDir := filepath.Join("videos", fmt.Sprintf("%04d", now.Year()), fmt.Sprintf("%02d", int(now.Month())))
	dir := filepath.Join(s.root, relDir)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", 0, err
	}

	id := uuid.NewString()
	fileName := id + ext
	fullPath := filepath.Join(dir, fileName)
	tmpPath := fullPath + ".part"

	f, err := os.OpenFile(tmpPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
	if err != nil {
		return "", 0, err
	}

	buf := make([]byte, copyBufferSize)
	wrapped := readerWithContext(ctx, src)
	written, err = io.CopyBuffer(f, wrapped, buf)
	cerr := f.Close()
	if err != nil {
		_ = os.Remove(tmpPath)
		return "", 0, err
	}
	if cerr != nil {
		_ = os.Remove(tmpPath)
		return "", 0, cerr
	}
	if written > maxVideoBytes {
		_ = os.Remove(tmpPath)
		return "", 0, fmt.Errorf("payload too large")
	}
	if written <= 0 {
		_ = os.Remove(tmpPath)
		return "", 0, fmt.Errorf("empty file")
	}

	headerSize := 512
	if written < int64(headerSize) {
		headerSize = int(written)
	}
	header := make([]byte, headerSize)
	select {
	case <-ctx.Done():
		_ = os.Remove(tmpPath)
		return "", 0, ctx.Err()
	default:
	}

	hf, err := os.Open(tmpPath)
	if err != nil {
		_ = os.Remove(tmpPath)
		return "", 0, err
	}
	if _, err := io.ReadFull(hf, header); err != nil {
		_ = hf.Close()
		_ = os.Remove(tmpPath)
		return "", 0, err
	}
	_ = hf.Close()

	detected := sniffVideoMIME(header)
	if detected == "" || detected != strings.TrimSpace(strings.ToLower(mime)) {
		_ = os.Remove(tmpPath)
		return "", 0, fmt.Errorf("unsupported media type")
	}

	select {
	case <-ctx.Done():
		_ = os.Remove(tmpPath)
		return "", 0, ctx.Err()
	default:
	}

	if err := os.Rename(tmpPath, fullPath); err != nil {
		_ = os.Remove(tmpPath)
		return "", 0, err
	}

	key := filepath.ToSlash(filepath.Join(relDir, fileName))
	return key, written, nil
}

// readerWithContext wraps r so reads observe ctx cancellation without buffering.
func readerWithContext(ctx context.Context, r io.Reader) io.Reader {
	if ctx == nil {
		return r
	}
	return &ctxReader{ctx: ctx, r: r}
}

type ctxReader struct {
	ctx context.Context
	r   io.Reader
}

func (cr *ctxReader) Read(p []byte) (int, error) {
	select {
	case <-cr.ctx.Done():
		return 0, cr.ctx.Err()
	default:
	}
	return cr.r.Read(p)
}

func sniffVideoMIME(header []byte) string {
	if len(header) >= 12 && string(header[4:8]) == "ftyp" {
		return "video/mp4"
	}
	if len(header) >= 4 && header[0] == 0x1A && header[1] == 0x45 && header[2] == 0xDF && header[3] == 0xA3 {
		return "video/webm"
	}
	return ""
}

// Delete removes an object by relative storage key.
func (s *VideoStorage) Delete(storageKey string) error {
	path := s.AbsolutePath(storageKey)
	rel, err := filepath.Rel(filepath.Clean(s.root), filepath.Clean(path))
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return fmt.Errorf("invalid storage key")
	}
	err = os.Remove(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	return err
}
