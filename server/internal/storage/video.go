package storage

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"

	"video-sharing-system/server/internal/validation"
)

const copyBufferSize = 32 * 1024
const maxVideoBytes = 500 * 1024 * 1024

// remoteDeleteTimeout bounds best-effort cleanup calls to the bucket so a slow
// provider can never stall a delete request.
const remoteDeleteTimeout = 15 * time.Second

// VideoStorage decides where a video file physically lives. Exactly one of
// three backends is active:
//
//	Cloudflare R2   … set the R2_* variables (used on Vercel)
//	local disk      … the default, used by classroom LAN and Docker runs
//
// Videos uploaded before the R2 migration still point at Vercel Blob; those
// URLs stay readable and deletable through legacyBlobToken.
type VideoStorage struct {
	root            string
	legacyBlobToken string
	r2              *R2Client
}

func NewVideoStorage(root string, legacyBlobToken string, r2 *R2Client) *VideoStorage {
	return &VideoStorage{
		root:            root,
		legacyBlobToken: strings.TrimSpace(legacyBlobToken),
		r2:              r2,
	}
}

func (s *VideoStorage) Root() string {
	return s.root
}

// RemoteEnabled reports whether video files are stored in a cloud bucket
// instead of on this machine's disk.
func (s *VideoStorage) RemoteEnabled() bool {
	return s.r2 != nil
}

// DirectUpload is a one-shot, pre-authorized upload slot handed to the browser.
type DirectUpload struct {
	// UploadURL accepts a single HTTP PUT of the video file.
	UploadURL string
	// PublicURL is where the finished file will be readable from.
	PublicURL string
	// ContentType must be sent verbatim on the PUT; it is part of the signature.
	ContentType string
	// ExpiresInSeconds is how long UploadURL stays usable.
	ExpiresInSeconds int
}

// IssueDirectUpload reserves a new object key and signs an upload URL for it.
// The video file goes from the browser straight to the bucket, so it never
// touches this server's memory or Vercel's request-body size limit.
func (s *VideoStorage) IssueDirectUpload(mime string, validFor time.Duration) (DirectUpload, error) {
	if s.r2 == nil {
		return DirectUpload{}, errors.New("remote video storage is not configured")
	}

	ext := validation.ExtForMIME(mime)
	if ext == "" {
		return DirectUpload{}, fmt.Errorf("unsupported mime")
	}

	objectKey := uuid.NewString() + ext
	uploadURL, err := s.r2.PresignPut(objectKey, mime, validFor)
	if err != nil {
		return DirectUpload{}, err
	}

	return DirectUpload{
		UploadURL:        uploadURL,
		PublicURL:        s.r2.PublicURL(objectKey),
		ContentType:      mime,
		ExpiresInSeconds: int(validFor.Seconds()),
	}, nil
}

// AcceptsUploadedURL reports whether a URL sent by the browser really points at
// the bucket this server hands out upload slots for. Without this check a
// client could register any address on the internet as a video file.
func (s *VideoStorage) AcceptsUploadedURL(rawURL string) bool {
	return s.r2 != nil && s.r2.OwnsURL(rawURL)
}

// StatRemote reads the authoritative size and content type of an uploaded
// object without downloading it.
func (s *VideoStorage) StatRemote(ctx context.Context, objectURL string) (int64, string, error) {
	return headObject(ctx, objectURL)
}

// AbsolutePath resolves a relative storage key under root.
func (s *VideoStorage) AbsolutePath(storageKey string) string {
	clean := filepath.Clean(storageKey)
	return filepath.Join(s.root, clean)
}

// PrepareWritableDir ensures root exists and is writable.
func (s *VideoStorage) PrepareWritableDir(ctx context.Context) error {
	if s.RemoteEnabled() {
		return nil
	}

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

	key := filepath.ToSlash(filepath.Join(relDir, fileName))

	if s.r2 != nil {
		// Forward the validated temp file to the bucket, streaming it so memory
		// use stays flat regardless of video size.
		uploaded, err := os.Open(tmpPath)
		if err != nil {
			_ = os.Remove(tmpPath)
			return "", 0, err
		}
		err = s.r2.PutObject(ctx, fileName, mime, uploaded, written)
		_ = uploaded.Close()
		_ = os.Remove(tmpPath)
		if err != nil {
			return "", 0, err
		}
		return s.r2.PublicURL(fileName), written, nil
	}

	if err := os.Rename(tmpPath, fullPath); err != nil {
		_ = os.Remove(tmpPath)
		return "", 0, err
	}

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

// Delete removes an object by relative storage key or public URL.
func (s *VideoStorage) Delete(storageKey string) error {
	if strings.HasPrefix(storageKey, "https://") {
		return s.deleteRemote(storageKey)
	}

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

// deleteRemote routes a public URL back to the provider that stores it, so
// videos from before the R2 migration are still cleaned up properly.
func (s *VideoStorage) deleteRemote(objectURL string) error {
	ctx, cancel := context.WithTimeout(context.Background(), remoteDeleteTimeout)
	defer cancel()

	if key, ok := s.r2.ObjectKeyFromURL(objectURL); ok {
		return s.r2.DeleteObject(ctx, key)
	}

	parsed, err := url.Parse(objectURL)
	if err != nil {
		return fmt.Errorf("invalid storage key")
	}
	if s.legacyBlobToken != "" && isLegacyBlobHost(parsed.Hostname()) {
		return deleteFromBlob(ctx, s.legacyBlobToken, objectURL)
	}

	// Nothing here can remove the object; the database row is already gone.
	return nil
}
