package apitest

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/cookiejar"
	"net/textproto"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

// seededUser is a user inserted directly into the database for a test, together
// with the plaintext password so the test can log in through the real API.
type seededUser struct {
	ID       int64
	PublicID uuid.UUID
	Username string
	Password string
	Role     string
}

// seedUser inserts a user straight into PostgreSQL with a bcrypt-hashed
// password. This deliberately bypasses the API so tests do not depend on the
// migration-seeded teacher account (whose plaintext password is unknown).
func seedUser(t *testing.T, username, displayName, password, role string) seededUser {
	t.Helper()

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		t.Fatalf("seedUser: hash password: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	const q = `
INSERT INTO users (username, display_name, password_hash, role)
VALUES ($1, $2, $3, $4::user_role)
RETURNING id, public_id;
`
	var u seededUser
	err = testPool.QueryRow(ctx, q, username, displayName, string(hash), role).
		Scan(&u.ID, &u.PublicID)
	if err != nil {
		t.Fatalf("seedUser: insert %q failed: %v", username, err)
	}
	u.Username = username
	u.Password = password
	u.Role = role
	return u
}

// newClient returns an HTTP client with its own cookie jar so each test actor
// keeps an independent session.
func newClient(t *testing.T) *http.Client {
	t.Helper()
	jar, err := cookiejar.New(nil)
	if err != nil {
		t.Fatalf("newClient: cookie jar: %v", err)
	}
	return &http.Client{
		Jar:     jar,
		Timeout: 30 * time.Second,
	}
}

// loginClient performs a real login for a seeded user and returns a client whose cookie jar holds the session.
func loginClient(t *testing.T, u seededUser) *http.Client {
	t.Helper()
	client := newClient(t)
	resp, body := doJSON(t, client, http.MethodPost, "/auth/login", map[string]string{
		"username": u.Username,
		"password": u.Password,
	})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("loginClient: expected 200 for %q, got %d: %s", u.Username, resp.StatusCode, string(body))
	}
	return client
}

// apiURL builds an absolute URL for a path under /api/v1 on the shared server.
func apiURL(path string) string {
	return testServer.URL + "/api/v1" + path
}

// absURL builds an absolute URL against an arbitrary server (used for isolated
// servers in streaming tests).
func absURL(server string, path string) string {
	return server + "/api/v1" + path
}

// doJSON sends a JSON request (body may be nil) and returns the response plus
// its fully-read body. The response body is closed before returning.
func doJSON(t *testing.T, client *http.Client, method, path string, body any) (*http.Response, []byte) {
	t.Helper()
	return doJSONURL(t, client, method, apiURL(path), body)
}

func doJSONURL(t *testing.T, client *http.Client, method, url string, body any) (*http.Response, []byte) {
	t.Helper()

	var reader io.Reader
	if body != nil {
		raw, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("doJSON: marshal body: %v", err)
		}
		reader = bytes.NewReader(raw)
	}

	req, err := http.NewRequest(method, url, reader)
	if err != nil {
		t.Fatalf("doJSON: new request: %v", err)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("doJSON: %s %s failed: %v", method, url, err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("doJSON: read body: %v", err)
	}
	return resp, respBody
}

// decodeData unmarshals the {"data": ...} envelope into target.
func decodeData(t *testing.T, body []byte, target any) {
	t.Helper()
	var env struct {
		Data json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(body, &env); err != nil {
		t.Fatalf("decodeData: outer unmarshal: %v (body=%s)", err, string(body))
	}
	if err := json.Unmarshal(env.Data, target); err != nil {
		t.Fatalf("decodeData: inner unmarshal: %v (data=%s)", err, string(env.Data))
	}
}

// errorCode extracts error.code from a standard API error body.
func errorCode(t *testing.T, body []byte) string {
	t.Helper()
	var env struct {
		Error struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	if err := json.Unmarshal(body, &env); err != nil {
		t.Fatalf("errorCode: unmarshal: %v (body=%s)", err, string(body))
	}
	return env.Error.Code
}

// ---- Synthetic media generation -------------------------------------------
//
// The storage layer sniffs the first bytes of the uploaded stream and requires
// them to match the declared MIME type. We craft minimal byte sequences that
// satisfy that sniffer without bundling real video files:
//   - MP4 : bytes 4..8 must equal "ftyp"
//   - WebM: first 4 bytes must equal 1A 45 DF A3 (EBML magic)
// The remainder is deterministic padding so Range tests can verify byte ranges.

func mp4Bytes(total int) []byte {
	header := []byte{
		0x00, 0x00, 0x00, 0x18, // box size (24)
		'f', 't', 'y', 'p', // box type "ftyp"
		'i', 's', 'o', 'm', // major brand
		0x00, 0x00, 0x02, 0x00, // minor version
		'i', 's', 'o', 'm', 'i', 's', 'o', '2', // compatible brands
	}
	return padTo(header, total)
}

func webmBytes(total int) []byte {
	header := []byte{
		0x1A, 0x45, 0xDF, 0xA3, // EBML magic
		0x01, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x1F,
	}
	return padTo(header, total)
}

func padTo(header []byte, total int) []byte {
	if total < len(header) {
		total = len(header)
	}
	out := make([]byte, total)
	copy(out, header)
	for i := len(header); i < total; i++ {
		// Deterministic, non-trivial pattern for range comparisons.
		out[i] = byte(i % 251)
	}
	return out
}

// uploadVideo posts a multipart/form-data upload through the real handler.
// The file part carries an explicit Content-Type so the handler can read it.
func uploadVideo(t *testing.T, client *http.Client, title, description, filename, mime string, data []byte) (*http.Response, []byte) {
	t.Helper()
	return uploadVideoURL(t, client, apiURL("/videos"), title, description, filename, mime, data)
}

func uploadVideoURL(t *testing.T, client *http.Client, url, title, description, filename, mime string, data []byte) (*http.Response, []byte) {
	t.Helper()

	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)

	if title != "" {
		if err := w.WriteField("title", title); err != nil {
			t.Fatalf("uploadVideo: write title: %v", err)
		}
	}
	if description != "" {
		if err := w.WriteField("description", description); err != nil {
			t.Fatalf("uploadVideo: write description: %v", err)
		}
	}

	partHeader := textproto.MIMEHeader{}
	partHeader.Set("Content-Disposition",
		`form-data; name="file"; filename="`+filename+`"`)
	partHeader.Set("Content-Type", mime)
	part, err := w.CreatePart(partHeader)
	if err != nil {
		t.Fatalf("uploadVideo: create part: %v", err)
	}
	if _, err := part.Write(data); err != nil {
		t.Fatalf("uploadVideo: write file: %v", err)
	}
	if err := w.Close(); err != nil {
		t.Fatalf("uploadVideo: close writer: %v", err)
	}

	req, err := http.NewRequest(http.MethodPost, url, &buf)
	if err != nil {
		t.Fatalf("uploadVideo: new request: %v", err)
	}
	req.Header.Set("Content-Type", w.FormDataContentType())

	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("uploadVideo: request failed: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("uploadVideo: read body: %v", err)
	}
	return resp, body
}

// videoStorageKey reads the storage_key column for a video by public id.
func videoStorageKey(t *testing.T, publicID uuid.UUID) string {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	var key string
	err := testPool.QueryRow(ctx,
		`SELECT storage_key FROM videos WHERE public_id = $1`, publicID).Scan(&key)
	if err != nil {
		t.Fatalf("videoStorageKey: query: %v", err)
	}
	return key
}

// mustContain fails the test if haystack does not contain needle.
func mustContain(t *testing.T, haystack, needle string) {
	t.Helper()
	if !strings.Contains(haystack, needle) {
		t.Fatalf("expected %q to contain %q", haystack, needle)
	}
}

// mustUUID parses a UUID string or fails the test.
func mustUUID(t *testing.T, s string) uuid.UUID {
	t.Helper()
	id, err := uuid.Parse(s)
	if err != nil {
		t.Fatalf("mustUUID: invalid uuid %q: %v", s, err)
	}
	return id
}
