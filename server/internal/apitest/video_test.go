package apitest

import (
	"context"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/uuid"
)

// uploadedVideo is the minimal info tests need after a successful upload.
type uploadedVideo struct {
	ID    uuid.UUID
	Title string
}

// createPublishedVideo uploads a small valid MP4 as the given client and
// returns the created video's public id.
func createPublishedVideo(t *testing.T, client *http.Client, title string, data []byte) uploadedVideo {
	t.Helper()
	resp, body := uploadVideo(t, client, title, "テスト説明", "clip.mp4", "video/mp4", data)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("upload: expected 201, got %d: %s", resp.StatusCode, string(body))
	}
	var created struct {
		ID     string `json:"id"`
		Title  string `json:"title"`
		Status string `json:"status"`
	}
	decodeData(t, body, &created)
	id, err := uuid.Parse(created.ID)
	if err != nil {
		t.Fatalf("upload: invalid id %q: %v", created.ID, err)
	}
	if created.Status != "published" {
		t.Fatalf("upload: expected published, got %q", created.Status)
	}
	return uploadedVideo{ID: id, Title: created.Title}
}

// TestUploadPersistsToDBAndDisk is the core verification: an uploaded video is
// recorded in PostgreSQL with correct metadata AND its bytes land on disk with
// the exact uploaded size.
func TestUploadPersistsToDBAndDisk(t *testing.T) {
	setupTest(t)
	student := seedUser(t, "student01", "山田 太郎", "classroom-pass", "student")
	client := loginClient(t, student)

	data := mp4Bytes(64 * 1024) // 64 KiB synthetic MP4
	video := createPublishedVideo(t, client, "理科の実験", data)

	// 1) Database row is correct.
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	var (
		uploaderID    int64
		title         string
		status        string
		storageKey    string
		mimeType      string
		fileSizeBytes int64
	)
	err := testPool.QueryRow(ctx, `
SELECT uploader_id, title, status, storage_key, mime_type, file_size_bytes
FROM videos WHERE public_id = $1 AND deleted_at IS NULL`, video.ID).
		Scan(&uploaderID, &title, &status, &storageKey, &mimeType, &fileSizeBytes)
	if err != nil {
		t.Fatalf("verify video row: %v", err)
	}
	if uploaderID != student.ID {
		t.Fatalf("uploader_id: want %d, got %d", student.ID, uploaderID)
	}
	if title != "理科の実験" || status != "published" || mimeType != "video/mp4" {
		t.Fatalf("video row metadata mismatch: title=%q status=%q mime=%q", title, status, mimeType)
	}
	if fileSizeBytes != int64(len(data)) {
		t.Fatalf("file_size_bytes: want %d, got %d", len(data), fileSizeBytes)
	}
	if storageKey == "" {
		t.Fatalf("storage_key should not be empty")
	}

	// 2) The actual file exists on disk with the same size.
	diskPath := filepath.Join(uploadDir, filepath.FromSlash(storageKey))
	info, err := os.Stat(diskPath)
	if err != nil {
		t.Fatalf("stat uploaded file %q: %v", diskPath, err)
	}
	if info.Size() != int64(len(data)) {
		t.Fatalf("on-disk size: want %d, got %d", len(data), info.Size())
	}
}

// TestUploadRejectsNonVideoMIME verifies a declared non-video MIME is rejected.
func TestUploadRejectsNonVideoMIME(t *testing.T) {
	setupTest(t)
	student := seedUser(t, "student01", "山田 太郎", "classroom-pass", "student")
	client := loginClient(t, student)

	resp, body := uploadVideo(t, client, "ダメな動画", "", "note.txt", "text/plain", []byte("hello world"))
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("upload non-video: expected 400, got %d: %s", resp.StatusCode, string(body))
	}
	if code := errorCode(t, body); code != "VALIDATION_ERROR" {
		t.Fatalf("upload non-video: expected VALIDATION_ERROR, got %q", code)
	}
}

// TestUploadRejectsContentMismatch verifies the storage sniffer rejects a file
// whose real bytes do not match the declared MIME type (415).
func TestUploadRejectsContentMismatch(t *testing.T) {
	setupTest(t)
	student := seedUser(t, "student01", "山田 太郎", "classroom-pass", "student")
	client := loginClient(t, student)

	// Declare MP4 + .mp4 extension, but send WebM magic bytes.
	resp, body := uploadVideo(t, client, "偽装動画", "", "fake.mp4", "video/mp4", webmBytes(4096))
	if resp.StatusCode != http.StatusUnsupportedMediaType {
		t.Fatalf("content mismatch: expected 415, got %d: %s", resp.StatusCode, string(body))
	}
	if code := errorCode(t, body); code != "UNSUPPORTED_MEDIA_TYPE" {
		t.Fatalf("content mismatch: expected UNSUPPORTED_MEDIA_TYPE, got %q", code)
	}
}

// TestUploadRequiresAuth verifies anonymous uploads are rejected.
func TestUploadRequiresAuth(t *testing.T) {
	setupTest(t)
	resp, body := uploadVideo(t, newClient(t), "匿名", "", "clip.mp4", "video/mp4", mp4Bytes(4096))
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("anonymous upload: expected 401, got %d: %s", resp.StatusCode, string(body))
	}
}

// TestGetVideo verifies the detail endpoint returns the stored metadata.
func TestGetVideo(t *testing.T) {
	setupTest(t)
	student := seedUser(t, "student01", "山田 太郎", "classroom-pass", "student")
	client := loginClient(t, student)
	video := createPublishedVideo(t, client, "公開動画", mp4Bytes(8192))

	resp, body := doJSON(t, newClient(t), http.MethodGet, "/videos/"+video.ID.String(), nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("get video: expected 200, got %d: %s", resp.StatusCode, string(body))
	}
	var got struct {
		ID            string `json:"id"`
		Title         string `json:"title"`
		MimeType      string `json:"mime_type"`
		FileSizeBytes int64  `json:"file_size_bytes"`
		Uploader      struct {
			DisplayName string `json:"display_name"`
		} `json:"uploader"`
	}
	decodeData(t, body, &got)
	if got.ID != video.ID.String() || got.Title != "公開動画" {
		t.Fatalf("get video: unexpected payload: %+v", got)
	}
	if got.Uploader.DisplayName != "山田 太郎" {
		t.Fatalf("get video: uploader display name mismatch: %q", got.Uploader.DisplayName)
	}
}

// TestPatchVideoOwner verifies an owner can edit title/description.
func TestPatchVideoOwner(t *testing.T) {
	setupTest(t)
	student := seedUser(t, "student01", "山田 太郎", "classroom-pass", "student")
	client := loginClient(t, student)
	video := createPublishedVideo(t, client, "旧タイトル", mp4Bytes(8192))

	resp, body := doJSON(t, client, http.MethodPatch, "/videos/"+video.ID.String(), map[string]string{
		"title":       "新タイトル",
		"description": "更新後の説明",
	})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("patch video: expected 200, got %d: %s", resp.StatusCode, string(body))
	}
	var out struct {
		Title       string `json:"title"`
		Description string `json:"description"`
	}
	decodeData(t, body, &out)
	if out.Title != "新タイトル" || out.Description != "更新後の説明" {
		t.Fatalf("patch video: not updated: %+v", out)
	}
}

// TestPatchVideoForbiddenForOtherStudent verifies non-owners are blocked.
func TestPatchVideoForbiddenForOtherStudent(t *testing.T) {
	setupTest(t)
	owner := seedUser(t, "owner01", "投稿者", "classroom-pass", "student")
	other := seedUser(t, "other01", "別の生徒", "classroom-pass", "student")

	ownerClient := loginClient(t, owner)
	video := createPublishedVideo(t, ownerClient, "他人の動画", mp4Bytes(8192))

	otherClient := loginClient(t, other)
	resp, body := doJSON(t, otherClient, http.MethodPatch, "/videos/"+video.ID.String(), map[string]string{
		"title": "勝手に変更",
	})
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("patch as other student: expected 403, got %d: %s", resp.StatusCode, string(body))
	}
}

// TestStudentCannotChangeVideoStatus verifies status changes are teacher-only.
func TestStudentCannotChangeVideoStatus(t *testing.T) {
	setupTest(t)
	owner := seedUser(t, "owner01", "投稿者", "classroom-pass", "student")
	ownerClient := loginClient(t, owner)
	video := createPublishedVideo(t, ownerClient, "自分の動画", mp4Bytes(8192))

	resp, body := doJSON(t, ownerClient, http.MethodPatch, "/videos/"+video.ID.String(), map[string]string{
		"status": "hidden",
	})
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("status change by student: expected 403, got %d: %s", resp.StatusCode, string(body))
	}
}

// TestHiddenVideoVisibility verifies hidden videos are invisible to the public
// but still reachable by the owner and teachers, and excluded from the list.
func TestHiddenVideoVisibility(t *testing.T) {
	setupTest(t)
	teacher := seedUser(t, "teacher01", "先生", "teacher-pass", "teacher")
	owner := seedUser(t, "owner01", "投稿者", "classroom-pass", "student")

	ownerClient := loginClient(t, owner)
	video := createPublishedVideo(t, ownerClient, "非表示予定", mp4Bytes(8192))

	// Teacher hides it.
	teacherClient := loginClient(t, teacher)
	respHide, bodyHide := doJSON(t, teacherClient, http.MethodPatch, "/videos/"+video.ID.String(), map[string]string{
		"status": "hidden",
	})
	if respHide.StatusCode != http.StatusOK {
		t.Fatalf("hide video: expected 200, got %d: %s", respHide.StatusCode, string(bodyHide))
	}

	// Anonymous sees 404.
	respAnon, _ := doJSON(t, newClient(t), http.MethodGet, "/videos/"+video.ID.String(), nil)
	if respAnon.StatusCode != http.StatusNotFound {
		t.Fatalf("anon get hidden: expected 404, got %d", respAnon.StatusCode)
	}

	// Owner still sees it.
	respOwner, _ := doJSON(t, ownerClient, http.MethodGet, "/videos/"+video.ID.String(), nil)
	if respOwner.StatusCode != http.StatusOK {
		t.Fatalf("owner get hidden: expected 200, got %d", respOwner.StatusCode)
	}

	// Teacher still sees it.
	respTeacher, _ := doJSON(t, teacherClient, http.MethodGet, "/videos/"+video.ID.String(), nil)
	if respTeacher.StatusCode != http.StatusOK {
		t.Fatalf("teacher get hidden: expected 200, got %d", respTeacher.StatusCode)
	}

	// Public list must not include the hidden video.
	respList, bodyList := doJSON(t, newClient(t), http.MethodGet, "/videos", nil)
	if respList.StatusCode != http.StatusOK {
		t.Fatalf("list videos: expected 200, got %d: %s", respList.StatusCode, string(bodyList))
	}
	var list []struct {
		ID string `json:"id"`
	}
	decodeData(t, bodyList, &list)
	for _, v := range list {
		if v.ID == video.ID.String() {
			t.Fatalf("hidden video should not appear in public list")
		}
	}
}

// TestDeleteVideoRemovesFile verifies delete soft-deletes the row and removes
// the on-disk file.
func TestDeleteVideoRemovesFile(t *testing.T) {
	setupTest(t)
	owner := seedUser(t, "owner01", "投稿者", "classroom-pass", "student")
	client := loginClient(t, owner)
	video := createPublishedVideo(t, client, "削除予定", mp4Bytes(8192))

	storageKey := videoStorageKey(t, video.ID)
	diskPath := filepath.Join(uploadDir, filepath.FromSlash(storageKey))
	if _, err := os.Stat(diskPath); err != nil {
		t.Fatalf("pre-delete: file should exist: %v", err)
	}

	resp, body := doJSON(t, client, http.MethodDelete, "/videos/"+video.ID.String(), nil)
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("delete video: expected 204, got %d: %s", resp.StatusCode, string(body))
	}

	// File should be gone.
	if _, err := os.Stat(diskPath); !os.IsNotExist(err) {
		t.Fatalf("post-delete: file should be removed, stat err=%v", err)
	}

	// Row should be soft-deleted.
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	var deleted bool
	if err := testPool.QueryRow(ctx,
		`SELECT deleted_at IS NOT NULL FROM videos WHERE public_id = $1`, video.ID).Scan(&deleted); err != nil {
		t.Fatalf("verify soft delete: %v", err)
	}
	if !deleted {
		t.Fatalf("video should be soft-deleted")
	}

	// GET now returns 404.
	respGet, _ := doJSON(t, client, http.MethodGet, "/videos/"+video.ID.String(), nil)
	if respGet.StatusCode != http.StatusNotFound {
		t.Fatalf("get deleted video: expected 404, got %d", respGet.StatusCode)
	}
}

// TestMeVideosIncludesHidden verifies the owner's video list contains both
// published and hidden videos.
func TestMeVideosIncludesHidden(t *testing.T) {
	setupTest(t)
	teacher := seedUser(t, "teacher01", "先生", "teacher-pass", "teacher")
	owner := seedUser(t, "owner01", "投稿者", "classroom-pass", "student")
	ownerClient := loginClient(t, owner)

	v1 := createPublishedVideo(t, ownerClient, "公開1", mp4Bytes(4096))
	v2 := createPublishedVideo(t, ownerClient, "非表示1", mp4Bytes(4096))

	// Teacher hides v2.
	teacherClient := loginClient(t, teacher)
	respHide, _ := doJSON(t, teacherClient, http.MethodPatch, "/videos/"+v2.ID.String(), map[string]string{
		"status": "hidden",
	})
	if respHide.StatusCode != http.StatusOK {
		t.Fatalf("hide v2: got %d", respHide.StatusCode)
	}

	resp, body := doJSON(t, ownerClient, http.MethodGet, "/me/videos", nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("me/videos: expected 200, got %d: %s", resp.StatusCode, string(body))
	}
	var list []struct {
		ID string `json:"id"`
	}
	decodeData(t, body, &list)

	ids := map[string]bool{}
	for _, v := range list {
		ids[v.ID] = true
	}
	if !ids[v1.ID.String()] || !ids[v2.ID.String()] {
		t.Fatalf("me/videos should include both published and hidden videos, got %v", ids)
	}
}
