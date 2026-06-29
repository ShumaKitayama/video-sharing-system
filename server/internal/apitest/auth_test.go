package apitest

import (
	"context"
	"net/http"
	"testing"
	"time"

	"video-sharing-system/server/internal/middleware"
)

// TestRegisterSuccessCreatesStudent verifies registration creates a student account
// and issues a session cookie.
func TestRegisterSuccessCreatesStudent(t *testing.T) {
	setupTest(t)

	client := newClient(t)
	resp, body := doJSON(t, client, http.MethodPost, "/auth/register", map[string]string{
		"username":     "newstudent",
		"display_name": "新規 太郎",
		"password":     "classroom-pass",
	})
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("register: expected 201, got %d: %s", resp.StatusCode, string(body))
	}

	var data struct {
		User struct {
			Username    string `json:"username"`
			DisplayName string `json:"display_name"`
			Role        string `json:"role"`
		} `json:"user"`
	}
	decodeData(t, body, &data)
	if data.User.Username != "newstudent" || data.User.Role != "student" {
		t.Fatalf("register: unexpected user payload: %+v", data.User)
	}

	var found bool
	for _, ck := range resp.Cookies() {
		if ck.Name == middleware.SessionCookieName {
			found = true
			if !ck.HttpOnly {
				t.Fatalf("session cookie should be HttpOnly")
			}
		}
	}
	if !found {
		t.Fatalf("register: %q cookie not set", middleware.SessionCookieName)
	}

	respMe, bodyMe := doJSON(t, client, http.MethodGet, "/auth/me", nil)
	if respMe.StatusCode != http.StatusOK {
		t.Fatalf("/auth/me after register: expected 200, got %d: %s", respMe.StatusCode, string(bodyMe))
	}
}

// TestRegisterDuplicateUsername verifies duplicate usernames return 409.
func TestRegisterDuplicateUsername(t *testing.T) {
	setupTest(t)
	seedUser(t, "student01", "山田 太郎", "classroom-pass", "student")

	resp, body := doJSON(t, newClient(t), http.MethodPost, "/auth/register", map[string]string{
		"username":     "student01",
		"display_name": "別の名前",
		"password":     "classroom-pass",
	})
	if resp.StatusCode != http.StatusConflict {
		t.Fatalf("register: expected 409, got %d: %s", resp.StatusCode, string(body))
	}
	if code := errorCode(t, body); code != "CONFLICT" {
		t.Fatalf("register: expected CONFLICT, got %q", code)
	}
}

// TestLoginSuccessSetsCookie verifies a valid login returns the user payload
// and issues an HttpOnly session cookie.
func TestLoginSuccessSetsCookie(t *testing.T) {
	setupTest(t)
	seedUser(t, "student01", "山田 太郎", "classroom-pass", "student")

	client := newClient(t)
	resp, body := doJSON(t, client, http.MethodPost, "/auth/login", map[string]string{
		"username": "student01",
		"password": "classroom-pass",
	})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("login: expected 200, got %d: %s", resp.StatusCode, string(body))
	}

	var data struct {
		User struct {
			Username    string `json:"username"`
			DisplayName string `json:"display_name"`
			Role        string `json:"role"`
		} `json:"user"`
	}
	decodeData(t, body, &data)
	if data.User.Username != "student01" || data.User.Role != "student" {
		t.Fatalf("login: unexpected user payload: %+v", data.User)
	}

	// The session cookie must be present and HttpOnly.
	var found bool
	for _, ck := range resp.Cookies() {
		if ck.Name == middleware.SessionCookieName {
			found = true
			if !ck.HttpOnly {
				t.Fatalf("session cookie should be HttpOnly")
			}
			if ck.Value == "" {
				t.Fatalf("session cookie value should not be empty")
			}
		}
	}
	if !found {
		t.Fatalf("login: %q cookie not set", middleware.SessionCookieName)
	}
}

// TestLoginWrongPassword verifies wrong credentials return 401.
func TestLoginWrongPassword(t *testing.T) {
	setupTest(t)
	seedUser(t, "student01", "山田 太郎", "classroom-pass", "student")

	resp, body := doJSON(t, newClient(t), http.MethodPost, "/auth/login", map[string]string{
		"username": "student01",
		"password": "wrong-password",
	})
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("login: expected 401, got %d: %s", resp.StatusCode, string(body))
	}
	if code := errorCode(t, body); code != "UNAUTHENTICATED" {
		t.Fatalf("login: expected UNAUTHENTICATED, got %q", code)
	}
}

// TestLoginInactiveUser verifies a deactivated account cannot log in (403).
func TestLoginInactiveUser(t *testing.T) {
	setupTest(t)
	u := seedUser(t, "blocked01", "停止 太郎", "classroom-pass", "student")

	// Deactivate directly in the database.
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if _, err := testPool.Exec(ctx, `UPDATE users SET is_active = FALSE WHERE id = $1`, u.ID); err != nil {
		t.Fatalf("deactivate user: %v", err)
	}

	resp, body := doJSON(t, newClient(t), http.MethodPost, "/auth/login", map[string]string{
		"username": "blocked01",
		"password": "classroom-pass",
	})
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("login: expected 403, got %d: %s", resp.StatusCode, string(body))
	}
	if code := errorCode(t, body); code != "FORBIDDEN" {
		t.Fatalf("login: expected FORBIDDEN, got %q", code)
	}
}

// TestAuthMe verifies /auth/me returns the logged-in user and rejects anonymous.
func TestAuthMe(t *testing.T) {
	setupTest(t)
	u := seedUser(t, "student01", "山田 太郎", "classroom-pass", "student")
	client := loginClient(t, u)

	resp, body := doJSON(t, client, http.MethodGet, "/auth/me", nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("/auth/me: expected 200, got %d: %s", resp.StatusCode, string(body))
	}
	var me struct {
		ID       string `json:"id"`
		Username string `json:"username"`
		Role     string `json:"role"`
	}
	decodeData(t, body, &me)
	if me.Username != "student01" || me.ID != u.PublicID.String() {
		t.Fatalf("/auth/me: unexpected payload: %+v", me)
	}

	// Anonymous request must be rejected.
	respAnon, bodyAnon := doJSON(t, newClient(t), http.MethodGet, "/auth/me", nil)
	if respAnon.StatusCode != http.StatusUnauthorized {
		t.Fatalf("/auth/me anonymous: expected 401, got %d: %s", respAnon.StatusCode, string(bodyAnon))
	}
}

// TestLogout verifies logout returns 204 and invalidates the session.
func TestLogout(t *testing.T) {
	setupTest(t)
	u := seedUser(t, "student01", "山田 太郎", "classroom-pass", "student")
	client := loginClient(t, u)

	// Confirm the session works first.
	resp, _ := doJSON(t, client, http.MethodGet, "/auth/me", nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("pre-logout /auth/me: expected 200, got %d", resp.StatusCode)
	}

	respLogout, bodyLogout := doJSON(t, client, http.MethodPost, "/auth/logout", nil)
	if respLogout.StatusCode != http.StatusNoContent {
		t.Fatalf("logout: expected 204, got %d: %s", respLogout.StatusCode, string(bodyLogout))
	}

	// Cookie is cleared client-side; server-side the row is gone.

	// The session token should now be invalid server-side. Even though the jar
	// cleared the cookie, re-confirm an anonymous follow-up is unauthorized.
	respAfter, _ := doJSON(t, client, http.MethodGet, "/auth/me", nil)
	if respAfter.StatusCode != http.StatusUnauthorized {
		t.Fatalf("post-logout /auth/me: expected 401, got %d", respAfter.StatusCode)
	}
}
