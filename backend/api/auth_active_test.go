package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/xiagao/fund-dashboard/backend/middleware"
	"github.com/xiagao/fund-dashboard/backend/store"
)

// A deactivated user who submits the CORRECT password is refused with 403 and
// a generic body — no token, and nothing that reveals the account is disabled.
func TestLogin_Deactivated_Forbidden(t *testing.T) {
	s, adminID, _ := newTestServer(t)
	if err := store.SetFriendActive(context.Background(), s.DB, adminID, false); err != nil {
		t.Fatal(err)
	}

	rr := postLogin(s, "admin", "correct-password")
	if rr.Code != http.StatusForbidden {
		t.Fatalf("deactivated login: got %d want 403 (body=%s)", rr.Code, rr.Body.String())
	}
	body := rr.Body.String()
	for _, leak := range []string{"停用", "disabled", "deactivat", "inactive"} {
		if strings.Contains(strings.ToLower(body), strings.ToLower(leak)) {
			t.Errorf("403 body leaks account state (%q): %s", leak, body)
		}
	}
	if h := rr.Header().Get("Set-Cookie"); h != "" {
		t.Errorf("deactivated login must not set a session cookie, got %q", h)
	}
}

// A deactivated user submitting the WRONG password gets the ordinary 401 — the
// active check runs only after a correct password, so it can't be used to probe
// which accounts are disabled without knowing the password.
func TestLogin_Deactivated_WrongPasswordStill401(t *testing.T) {
	s, adminID, _ := newTestServer(t)
	if err := store.SetFriendActive(context.Background(), s.DB, adminID, false); err != nil {
		t.Fatal(err)
	}

	rr := postLogin(s, "admin", "wrong-password")
	if rr.Code != http.StatusUnauthorized {
		t.Errorf("deactivated + wrong password: got %d want 401", rr.Code)
	}
}

// Deactivating a friend invalidates any live session on the next request, the
// same way a password change does.
func TestSession_DeactivatedKillsLiveSession(t *testing.T) {
	s, adminID, hash := newTestServer(t)
	tok, _, err := middleware.IssueToken(testJWTSecret, adminID, "admin", true,
		middleware.PasswordVersion(hash))
	if err != nil {
		t.Fatal(err)
	}
	handler := s.auth(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) })
	doReq := func() int {
		req := httptest.NewRequest("GET", "/api/me", nil)
		req.AddCookie(&http.Cookie{Name: middleware.CookieName, Value: tok})
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)
		return rr.Code
	}

	if code := doReq(); code != http.StatusOK {
		t.Fatalf("active session: got %d want 200", code)
	}
	if err := store.SetFriendActive(context.Background(), s.DB, adminID, false); err != nil {
		t.Fatal(err)
	}
	if code := doReq(); code != http.StatusUnauthorized {
		t.Errorf("after deactivation: got %d want 401", code)
	}
}
