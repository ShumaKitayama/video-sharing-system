package storage

import (
	"bytes"
	"context"
	"net/http"
	"os"
	"testing"
	"time"
)

// Round-trips a real object through a real R2 bucket. Skipped unless the R2_*
// environment variables are present.
func TestLiveR2RoundTrip(t *testing.T) {
	cfg := R2Config{
		AccountID:       os.Getenv("R2_ACCOUNT_ID"),
		AccessKeyID:     os.Getenv("R2_ACCESS_KEY_ID"),
		SecretAccessKey: os.Getenv("R2_SECRET_ACCESS_KEY"),
		Bucket:          os.Getenv("R2_BUCKET"),
		PublicBaseURL:   os.Getenv("R2_PUBLIC_BASE_URL"),
	}
	if cfg.AccountID == "" {
		t.Skip("R2_* environment variables are not set")
	}

	client, err := NewR2Client(cfg)
	if err != nil {
		t.Fatalf("NewR2Client: %v", err)
	}
	store := NewVideoStorage(t.TempDir(), "", client)

	if !store.RemoteEnabled() {
		t.Fatal("remote storage should be enabled")
	}

	slot, err := store.IssueDirectUpload("video/mp4", 10*time.Minute)
	if err != nil {
		t.Fatalf("IssueDirectUpload: %v", err)
	}
	t.Logf("public url: %s", slot.PublicURL)

	if !store.AcceptsUploadedURL(slot.PublicURL) {
		t.Fatalf("storage refused the URL it just generated: %s", slot.PublicURL)
	}

	// A minimal MP4 header followed by filler, so the object looks like a video.
	payload := append([]byte{
		0x00, 0x00, 0x00, 0x18, 'f', 't', 'y', 'p',
		'i', 's', 'o', 'm', 0x00, 0x00, 0x02, 0x00,
		'i', 's', 'o', 'm', 'm', 'p', '4', '1',
	}, bytes.Repeat([]byte{0x42}, 2048)...)

	req, err := http.NewRequest(http.MethodPut, slot.UploadURL, bytes.NewReader(payload))
	if err != nil {
		t.Fatalf("build PUT: %v", err)
	}
	req.Header.Set("Content-Type", slot.ContentType)
	req.ContentLength = int64(len(payload))

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("PUT to presigned URL: %v", err)
	}
	body := make([]byte, 512)
	n, _ := res.Body.Read(body)
	res.Body.Close()
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		t.Fatalf("PUT rejected with status %d: %s", res.StatusCode, body[:n])
	}
	t.Logf("upload status: %d", res.StatusCode)

	ctx := context.Background()
	size, contentType, err := store.StatRemote(ctx, slot.PublicURL)
	if err != nil {
		t.Fatalf("HEAD on public URL: %v", err)
	}
	if size != int64(len(payload)) {
		t.Errorf("size = %d, want %d", size, len(payload))
	}
	if contentType != "video/mp4" {
		t.Errorf("content type = %q, want %q", contentType, "video/mp4")
	}
	t.Logf("head: size=%d contentType=%s", size, contentType)

	if err := store.Delete(slot.PublicURL); err != nil {
		t.Fatalf("Delete: %v", err)
	}

	if _, _, err := store.StatRemote(ctx, slot.PublicURL); err == nil {
		t.Error("object is still readable after delete")
	} else {
		t.Logf("object gone after delete: %v", err)
	}
}
