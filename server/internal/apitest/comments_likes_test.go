package apitest

import (
	"net/http"
	"testing"
)

// getVideoCounts reads comment_count / like_count via the public detail API.
func getVideoCounts(t *testing.T, client *http.Client, id string) (commentCount, likeCount int64) {
	t.Helper()
	resp, body := doJSON(t, client, http.MethodGet, "/videos/"+id, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("get video counts: expected 200, got %d: %s", resp.StatusCode, string(body))
	}
	var v struct {
		CommentCount int64 `json:"comment_count"`
		LikeCount    int64 `json:"like_count"`
	}
	decodeData(t, body, &v)
	return v.CommentCount, v.LikeCount
}

// TestCommentLifecycle verifies create -> list -> count -> delete with counters.
func TestCommentLifecycle(t *testing.T) {
	setupTest(t)
	owner := seedUser(t, "owner01", "投稿者", "classroom-pass", "student")
	commenter := seedUser(t, "viewer01", "視聴者", "classroom-pass", "student")

	ownerClient := loginClient(t, owner)
	video := createPublishedVideo(t, ownerClient, "コメント対象", mp4Bytes(4096))

	commenterClient := loginClient(t, commenter)

	// Create a comment.
	resp, body := doJSON(t, commenterClient, http.MethodPost, "/videos/"+video.ID.String()+"/comments", map[string]string{
		"body": "分かりやすかった",
	})
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("create comment: expected 201, got %d: %s", resp.StatusCode, string(body))
	}
	var created struct {
		ID     string `json:"id"`
		Body   string `json:"body"`
		Author struct {
			DisplayName string `json:"display_name"`
		} `json:"author"`
	}
	decodeData(t, body, &created)
	if created.Body != "分かりやすかった" || created.Author.DisplayName != "視聴者" {
		t.Fatalf("create comment: unexpected payload: %+v", created)
	}

	// List shows it, total is 1.
	respList, bodyList := doJSON(t, newClient(t), http.MethodGet, "/videos/"+video.ID.String()+"/comments", nil)
	if respList.StatusCode != http.StatusOK {
		t.Fatalf("list comments: expected 200, got %d: %s", respList.StatusCode, string(bodyList))
	}
	var comments []struct {
		ID string `json:"id"`
	}
	decodeData(t, bodyList, &comments)
	if len(comments) != 1 {
		t.Fatalf("list comments: expected 1, got %d", len(comments))
	}

	// comment_count on the video is 1.
	if cc, _ := getVideoCounts(t, newClient(t), video.ID.String()); cc != 1 {
		t.Fatalf("comment_count: want 1, got %d", cc)
	}

	// Author deletes it; count returns to 0.
	respDel, bodyDel := doJSON(t, commenterClient, http.MethodDelete, "/comments/"+created.ID, nil)
	if respDel.StatusCode != http.StatusNoContent {
		t.Fatalf("delete comment: expected 204, got %d: %s", respDel.StatusCode, string(bodyDel))
	}
	if cc, _ := getVideoCounts(t, newClient(t), video.ID.String()); cc != 0 {
		t.Fatalf("comment_count after delete: want 0, got %d", cc)
	}
}

// TestCommentValidationAndAuth verifies empty body (400) and anonymous (401).
func TestCommentValidationAndAuth(t *testing.T) {
	setupTest(t)
	owner := seedUser(t, "owner01", "投稿者", "classroom-pass", "student")
	ownerClient := loginClient(t, owner)
	video := createPublishedVideo(t, ownerClient, "対象", mp4Bytes(4096))

	// Empty body -> 400.
	resp, body := doJSON(t, ownerClient, http.MethodPost, "/videos/"+video.ID.String()+"/comments", map[string]string{
		"body": "   ",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("empty comment: expected 400, got %d: %s", resp.StatusCode, string(body))
	}

	// Anonymous -> 401.
	respAnon, _ := doJSON(t, newClient(t), http.MethodPost, "/videos/"+video.ID.String()+"/comments", map[string]string{
		"body": "こっそり",
	})
	if respAnon.StatusCode != http.StatusUnauthorized {
		t.Fatalf("anonymous comment: expected 401, got %d", respAnon.StatusCode)
	}
}

// TestCommentDeleteForbiddenForOther verifies non-author students cannot delete.
func TestCommentDeleteForbiddenForOther(t *testing.T) {
	setupTest(t)
	owner := seedUser(t, "owner01", "投稿者", "classroom-pass", "student")
	commenter := seedUser(t, "viewer01", "視聴者", "classroom-pass", "student")
	other := seedUser(t, "other01", "別人", "classroom-pass", "student")

	ownerClient := loginClient(t, owner)
	video := createPublishedVideo(t, ownerClient, "対象", mp4Bytes(4096))

	commenterClient := loginClient(t, commenter)
	resp, body := doJSON(t, commenterClient, http.MethodPost, "/videos/"+video.ID.String()+"/comments", map[string]string{
		"body": "コメント",
	})
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("setup comment: expected 201, got %d: %s", resp.StatusCode, string(body))
	}
	var created struct {
		ID string `json:"id"`
	}
	decodeData(t, body, &created)

	otherClient := loginClient(t, other)
	respDel, _ := doJSON(t, otherClient, http.MethodDelete, "/comments/"+created.ID, nil)
	if respDel.StatusCode != http.StatusForbidden {
		t.Fatalf("delete other's comment: expected 403, got %d", respDel.StatusCode)
	}
}

// TestCommentDeleteByTeacher verifies teachers can delete any comment.
func TestCommentDeleteByTeacher(t *testing.T) {
	setupTest(t)
	teacher := seedUser(t, "teacher01", "先生", "teacher-pass", "teacher")
	owner := seedUser(t, "owner01", "投稿者", "classroom-pass", "student")
	commenter := seedUser(t, "viewer01", "視聴者", "classroom-pass", "student")

	ownerClient := loginClient(t, owner)
	video := createPublishedVideo(t, ownerClient, "対象", mp4Bytes(4096))

	commenterClient := loginClient(t, commenter)
	respCreate, bodyCreate := doJSON(t, commenterClient, http.MethodPost, "/videos/"+video.ID.String()+"/comments", map[string]string{
		"body": "コメント",
	})
	if respCreate.StatusCode != http.StatusCreated {
		t.Fatalf("setup comment: expected 201, got %d: %s", respCreate.StatusCode, string(bodyCreate))
	}
	var created struct {
		ID string `json:"id"`
	}
	decodeData(t, bodyCreate, &created)

	teacherClient := loginClient(t, teacher)
	respDel, _ := doJSON(t, teacherClient, http.MethodDelete, "/comments/"+created.ID, nil)
	if respDel.StatusCode != http.StatusNoContent {
		t.Fatalf("teacher delete comment: expected 204, got %d", respDel.StatusCode)
	}
}

// TestLikeLifecycle verifies put (idempotent) -> get -> delete with counters.
func TestLikeLifecycle(t *testing.T) {
	setupTest(t)
	owner := seedUser(t, "owner01", "投稿者", "classroom-pass", "student")
	liker := seedUser(t, "liker01", "いいねする人", "classroom-pass", "student")

	ownerClient := loginClient(t, owner)
	video := createPublishedVideo(t, ownerClient, "いいね対象", mp4Bytes(4096))

	likerClient := loginClient(t, liker)

	// PUT like -> 200, liked true, like_count 1.
	resp, body := doJSON(t, likerClient, http.MethodPut, "/videos/"+video.ID.String()+"/like", nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("put like: expected 200, got %d: %s", resp.StatusCode, string(body))
	}
	var put struct {
		Liked     bool  `json:"liked"`
		LikeCount int64 `json:"like_count"`
	}
	decodeData(t, body, &put)
	if !put.Liked || put.LikeCount != 1 {
		t.Fatalf("put like: unexpected payload: %+v", put)
	}

	// PUT again is idempotent: still count 1.
	resp2, body2 := doJSON(t, likerClient, http.MethodPut, "/videos/"+video.ID.String()+"/like", nil)
	if resp2.StatusCode != http.StatusOK {
		t.Fatalf("put like again: expected 200, got %d", resp2.StatusCode)
	}
	var put2 struct {
		LikeCount int64 `json:"like_count"`
	}
	decodeData(t, body2, &put2)
	if put2.LikeCount != 1 {
		t.Fatalf("put like idempotent: like_count want 1, got %d", put2.LikeCount)
	}

	// GET like state -> liked true.
	respGet, bodyGet := doJSON(t, likerClient, http.MethodGet, "/videos/"+video.ID.String()+"/like", nil)
	if respGet.StatusCode != http.StatusOK {
		t.Fatalf("get like: expected 200, got %d", respGet.StatusCode)
	}
	var likeState struct {
		Liked bool `json:"liked"`
	}
	decodeData(t, bodyGet, &likeState)
	if !likeState.Liked {
		t.Fatalf("get like: expected liked true")
	}

	// Video like_count is 1.
	if _, lc := getVideoCounts(t, newClient(t), video.ID.String()); lc != 1 {
		t.Fatalf("like_count: want 1, got %d", lc)
	}

	// DELETE like -> 204, count back to 0.
	respDel, _ := doJSON(t, likerClient, http.MethodDelete, "/videos/"+video.ID.String()+"/like", nil)
	if respDel.StatusCode != http.StatusNoContent {
		t.Fatalf("delete like: expected 204, got %d", respDel.StatusCode)
	}
	if _, lc := getVideoCounts(t, newClient(t), video.ID.String()); lc != 0 {
		t.Fatalf("like_count after delete: want 0, got %d", lc)
	}

	// DELETE again is idempotent.
	respDel2, _ := doJSON(t, likerClient, http.MethodDelete, "/videos/"+video.ID.String()+"/like", nil)
	if respDel2.StatusCode != http.StatusNoContent {
		t.Fatalf("delete like again: expected 204, got %d", respDel2.StatusCode)
	}
}

// TestLikeRequiresAuth verifies like endpoints require authentication.
func TestLikeRequiresAuth(t *testing.T) {
	setupTest(t)
	owner := seedUser(t, "owner01", "投稿者", "classroom-pass", "student")
	ownerClient := loginClient(t, owner)
	video := createPublishedVideo(t, ownerClient, "対象", mp4Bytes(4096))

	resp, _ := doJSON(t, newClient(t), http.MethodPut, "/videos/"+video.ID.String()+"/like", nil)
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("anonymous like: expected 401, got %d", resp.StatusCode)
	}
}
