package apitest

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// loginClientOn logs in against a specific server (used for isolated servers).
func loginClientOn(t *testing.T, srv *httptest.Server, u seededUser) *http.Client {
	t.Helper()
	client := newClient(t)
	resp, body := doJSONURL(t, client, http.MethodPost, absURL(srv.URL, "/auth/login"), map[string]string{
		"username": u.Username,
		"password": u.Password,
	})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("loginClientOn: expected 200, got %d: %s", resp.StatusCode, string(body))
	}
	return client
}

// streamGet performs a GET on the stream endpoint with an optional Range header.
func streamGet(t *testing.T, client *http.Client, url, rangeHeader string) (*http.Response, []byte) {
	t.Helper()
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		t.Fatalf("streamGet: new request: %v", err)
	}
	if rangeHeader != "" {
		req.Header.Set("Range", rangeHeader)
	}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("streamGet: request failed: %v", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("streamGet: read body: %v", err)
	}
	return resp, body
}

// videoViewCount reads the view_count column directly.
func videoViewCount(t *testing.T, publicID fmt.Stringer) int64 {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	var n int64
	if err := testPool.QueryRow(ctx,
		`SELECT view_count FROM videos WHERE public_id = $1`, publicID).Scan(&n); err != nil {
		t.Fatalf("videoViewCount: %v", err)
	}
	return n
}

// TestStreamFullAndRange verifies full delivery (200) and Range delivery (206)
// without buffering, plus rejection of an unsatisfiable Range (416).
//
// A dedicated server instance is used so the in-memory view-dedupe map starts
// empty (table ids are reset by TRUNCATE between tests).
func TestStreamFullAndRange(t *testing.T) {
	setupTest(t)

	srv := newTestServer(uploadDir)
	defer srv.Close()

	student := seedUser(t, "student01", "山田 太郎", "classroom-pass", "student")
	client := loginClientOn(t, srv, student)

	const size = 4000
	data := mp4Bytes(size)
	resp, body := uploadVideoURL(t, client, absURL(srv.URL, "/videos"), "配信動画", "", "clip.mp4", "video/mp4", data)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("upload: expected 201, got %d: %s", resp.StatusCode, string(body))
	}
	var created struct {
		ID string `json:"id"`
	}
	decodeData(t, body, &created)
	streamURL := absURL(srv.URL, "/videos/"+created.ID+"/stream")

	// 1) Full GET: 200, full body, Accept-Ranges advertised.
	respFull, bodyFull := streamGet(t, client, streamURL, "")
	if respFull.StatusCode != http.StatusOK {
		t.Fatalf("stream full: expected 200, got %d", respFull.StatusCode)
	}
	if len(bodyFull) != size {
		t.Fatalf("stream full: expected %d bytes, got %d", size, len(bodyFull))
	}
	if !bytes.Equal(bodyFull, data) {
		t.Fatalf("stream full: body does not match uploaded bytes")
	}
	if respFull.Header.Get("Accept-Ranges") != "bytes" {
		t.Fatalf("stream full: expected Accept-Ranges: bytes, got %q", respFull.Header.Get("Accept-Ranges"))
	}

	// 2) Range GET: 206 Partial Content with correct headers and slice.
	respRange, bodyRange := streamGet(t, client, streamURL, "bytes=0-99")
	if respRange.StatusCode != http.StatusPartialContent {
		t.Fatalf("stream range: expected 206, got %d", respRange.StatusCode)
	}
	if len(bodyRange) != 100 {
		t.Fatalf("stream range: expected 100 bytes, got %d", len(bodyRange))
	}
	if !bytes.Equal(bodyRange, data[0:100]) {
		t.Fatalf("stream range: bytes do not match expected slice")
	}
	wantCR := fmt.Sprintf("bytes 0-99/%d", size)
	if got := respRange.Header.Get("Content-Range"); got != wantCR {
		t.Fatalf("stream range: Content-Range = %q, want %q", got, wantCR)
	}
	if got := respRange.Header.Get("Content-Length"); got != "100" {
		t.Fatalf("stream range: Content-Length = %q, want 100", got)
	}

	// 3) Unsatisfiable Range: 416.
	respBad, _ := streamGet(t, client, streamURL, "bytes=99999999-")
	if respBad.StatusCode != http.StatusRequestedRangeNotSatisfiable {
		t.Fatalf("stream bad range: expected 416, got %d", respBad.StatusCode)
	}
}

// TestStreamViewCountIncrementAndDedupe verifies the first playback increments
// view_count and repeated playback from the same viewer within the dedupe
// window does NOT double-count.
func TestStreamViewCountIncrementAndDedupe(t *testing.T) {
	setupTest(t)

	srv := newTestServer(uploadDir)
	defer srv.Close()

	student := seedUser(t, "student01", "山田 太郎", "classroom-pass", "student")
	client := loginClientOn(t, srv, student)

	data := mp4Bytes(4000)
	resp, body := uploadVideoURL(t, client, absURL(srv.URL, "/videos"), "再生回数動画", "", "clip.mp4", "video/mp4", data)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("upload: expected 201, got %d: %s", resp.StatusCode, string(body))
	}
	var created struct {
		ID string `json:"id"`
	}
	decodeData(t, body, &created)
	streamURL := absURL(srv.URL, "/videos/"+created.ID+"/stream")

	// Brand new video starts at zero views.
	pid := mustUUID(t, created.ID)
	if got := videoViewCount(t, pid); got != 0 {
		t.Fatalf("initial view_count: want 0, got %d", got)
	}

	// First playback bumps to 1 (increment is synchronous in ServeHTTP).
	if r, _ := streamGet(t, client, streamURL, ""); r.StatusCode != http.StatusOK {
		t.Fatalf("first stream: expected 200, got %d", r.StatusCode)
	}
	if got := videoViewCount(t, pid); got != 1 {
		t.Fatalf("after first play: want view_count 1, got %d", got)
	}

	// Several more playbacks by the same viewer within the dedupe window must
	// not increment the count again.
	for i := 0; i < 3; i++ {
		streamGet(t, client, streamURL, "bytes=0-99")
	}
	if got := videoViewCount(t, pid); got != 1 {
		t.Fatalf("after repeated play (dedupe): want view_count 1, got %d", got)
	}
}
