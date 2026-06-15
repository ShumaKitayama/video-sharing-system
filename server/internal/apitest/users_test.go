package apitest

import (
	"context"
	"net/http"
	"testing"
	"time"
)

// TestTeacherCreatesUser verifies a teacher can create a student and the row is
// actually written to PostgreSQL.
func TestTeacherCreatesUser(t *testing.T) {
	setupTest(t)
	teacher := seedUser(t, "teacher01", "先生", "teacher-pass", "teacher")
	client := loginClient(t, teacher)

	resp, body := doJSON(t, client, http.MethodPost, "/users", map[string]string{
		"username":     "newstudent",
		"display_name": "新入生",
		"password":     "student-pass",
		"role":         "student",
	})
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("create user: expected 201, got %d: %s", resp.StatusCode, string(body))
	}

	var created struct {
		ID       string `json:"id"`
		Username string `json:"username"`
		Role     string `json:"role"`
	}
	decodeData(t, body, &created)
	if created.Username != "newstudent" || created.Role != "student" {
		t.Fatalf("create user: unexpected payload: %+v", created)
	}

	// Verify the row exists in the database.
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	var count int
	if err := testPool.QueryRow(ctx,
		`SELECT COUNT(*) FROM users WHERE username = $1 AND deleted_at IS NULL`,
		"newstudent").Scan(&count); err != nil {
		t.Fatalf("verify user row: %v", err)
	}
	if count != 1 {
		t.Fatalf("verify user row: expected 1, got %d", count)
	}
}

// TestStudentCannotCreateUser verifies the teacher-only guard returns 403.
func TestStudentCannotCreateUser(t *testing.T) {
	setupTest(t)
	student := seedUser(t, "student01", "山田 太郎", "classroom-pass", "student")
	client := loginClient(t, student)

	resp, body := doJSON(t, client, http.MethodPost, "/users", map[string]string{
		"username":     "newstudent",
		"display_name": "新入生",
		"password":     "student-pass",
		"role":         "student",
	})
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("create user as student: expected 403, got %d: %s", resp.StatusCode, string(body))
	}
	if code := errorCode(t, body); code != "FORBIDDEN" {
		t.Fatalf("create user as student: expected FORBIDDEN, got %q", code)
	}
}

// TestCreateUserValidation verifies invalid input returns 400 VALIDATION_ERROR.
func TestCreateUserValidation(t *testing.T) {
	setupTest(t)
	teacher := seedUser(t, "teacher01", "先生", "teacher-pass", "teacher")
	client := loginClient(t, teacher)

	resp, body := doJSON(t, client, http.MethodPost, "/users", map[string]string{
		"username":     "ab",      // too short (min 3)
		"display_name": "新入生",
		"password":     "short",   // too short (min 8)
		"role":         "student",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("create user invalid: expected 400, got %d: %s", resp.StatusCode, string(body))
	}
	if code := errorCode(t, body); code != "VALIDATION_ERROR" {
		t.Fatalf("create user invalid: expected VALIDATION_ERROR, got %q", code)
	}
}

// TestTeacherListsUsers verifies the teacher-only list endpoint returns rows.
func TestTeacherListsUsers(t *testing.T) {
	setupTest(t)
	teacher := seedUser(t, "teacher01", "先生", "teacher-pass", "teacher")
	seedUser(t, "student01", "生徒1", "classroom-pass", "student")
	seedUser(t, "student02", "生徒2", "classroom-pass", "student")
	client := loginClient(t, teacher)

	resp, body := doJSON(t, client, http.MethodGet, "/users", nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("list users: expected 200, got %d: %s", resp.StatusCode, string(body))
	}

	var users []struct {
		Username string `json:"username"`
	}
	decodeData(t, body, &users)
	if len(users) != 3 {
		t.Fatalf("list users: expected 3, got %d (%s)", len(users), string(body))
	}
}

// TestGetUserSelfAndForbidden verifies a student can read itself but not others.
func TestGetUserSelfAndForbidden(t *testing.T) {
	setupTest(t)
	student := seedUser(t, "student01", "山田 太郎", "classroom-pass", "student")
	other := seedUser(t, "student02", "生徒2", "classroom-pass", "student")
	client := loginClient(t, student)

	// Self read is allowed.
	resp, body := doJSON(t, client, http.MethodGet, "/users/"+student.PublicID.String(), nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("get self: expected 200, got %d: %s", resp.StatusCode, string(body))
	}

	// Reading another user is forbidden for a student.
	respOther, bodyOther := doJSON(t, client, http.MethodGet, "/users/"+other.PublicID.String(), nil)
	if respOther.StatusCode != http.StatusForbidden {
		t.Fatalf("get other: expected 403, got %d: %s", respOther.StatusCode, string(bodyOther))
	}
}

// TestPatchUserSelfDisplayName verifies a student can update its own display name.
func TestPatchUserSelfDisplayName(t *testing.T) {
	setupTest(t)
	student := seedUser(t, "student01", "山田 太郎", "classroom-pass", "student")
	client := loginClient(t, student)

	resp, body := doJSON(t, client, http.MethodPatch, "/users/"+student.PublicID.String(), map[string]string{
		"display_name": "山田 たろう",
	})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("patch self: expected 200, got %d: %s", resp.StatusCode, string(body))
	}
	var out struct {
		DisplayName string `json:"display_name"`
	}
	decodeData(t, body, &out)
	if out.DisplayName != "山田 たろう" {
		t.Fatalf("patch self: display_name not updated: %q", out.DisplayName)
	}
}

// TestStudentCannotChangeRole verifies students cannot escalate privileges.
func TestStudentCannotChangeRole(t *testing.T) {
	setupTest(t)
	student := seedUser(t, "student01", "山田 太郎", "classroom-pass", "student")
	client := loginClient(t, student)

	resp, body := doJSON(t, client, http.MethodPatch, "/users/"+student.PublicID.String(), map[string]string{
		"role": "teacher",
	})
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("patch role as student: expected 403, got %d: %s", resp.StatusCode, string(body))
	}
}

// TestTeacherDeletesUser verifies the soft-delete path marks deleted_at.
func TestTeacherDeletesUser(t *testing.T) {
	setupTest(t)
	teacher := seedUser(t, "teacher01", "先生", "teacher-pass", "teacher")
	target := seedUser(t, "student01", "山田 太郎", "classroom-pass", "student")
	client := loginClient(t, teacher)

	resp, body := doJSON(t, client, http.MethodDelete, "/users/"+target.PublicID.String(), nil)
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("delete user: expected 204, got %d: %s", resp.StatusCode, string(body))
	}

	// The row should be soft-deleted (deleted_at set, is_active false).
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	var deleted bool
	var active bool
	if err := testPool.QueryRow(ctx,
		`SELECT deleted_at IS NOT NULL, is_active FROM users WHERE id = $1`,
		target.ID).Scan(&deleted, &active); err != nil {
		t.Fatalf("verify soft delete: %v", err)
	}
	if !deleted || active {
		t.Fatalf("verify soft delete: deleted=%v active=%v (want deleted=true active=false)", deleted, active)
	}

	// The deleted user can no longer log in.
	respLogin, _ := doJSON(t, newClient(t), http.MethodPost, "/auth/login", map[string]string{
		"username": "student01",
		"password": "classroom-pass",
	})
	if respLogin.StatusCode == http.StatusOK {
		t.Fatalf("deleted user should not be able to log in")
	}
}
