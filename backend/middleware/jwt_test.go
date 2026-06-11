package middleware

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

const testSecret = "test-secret-32-bytes-padding-xxxxxxxxxxxxxxxxxx"

func TestIssueAndVerify_Roundtrip(t *testing.T) {
	tok, exp, err := IssueToken(testSecret, 42, "alice", false, "")
	if err != nil {
		t.Fatal(err)
	}
	if !exp.After(time.Now()) {
		t.Errorf("exp should be in the future, got %v", exp)
	}
	c, err := VerifyToken(testSecret, tok)
	if err != nil {
		t.Fatalf("verify: %v", err)
	}
	if c.FriendID != 42 || c.Username != "alice" || c.IsAdmin {
		t.Errorf("claims roundtrip mismatch: %+v", c)
	}
}

func TestVerifyToken_BadSignature(t *testing.T) {
	tok, _, _ := IssueToken(testSecret, 1, "x", false, "")
	tampered := tok[:len(tok)-3] + "xxx"
	_, err := VerifyToken(testSecret, tampered)
	if !errors.Is(err, ErrBadSig) {
		t.Errorf("want ErrBadSig, got %v", err)
	}
}

func TestVerifyToken_WrongSecret(t *testing.T) {
	tok, _, _ := IssueToken(testSecret, 1, "x", false, "")
	_, err := VerifyToken("different-secret", tok)
	if !errors.Is(err, ErrBadSig) {
		t.Errorf("want ErrBadSig, got %v", err)
	}
}

func TestVerifyToken_Malformed(t *testing.T) {
	_, err := VerifyToken(testSecret, "not.a.valid.token.at.all")
	if !errors.Is(err, ErrBadSig) && !errors.Is(err, ErrBadToken) {
		t.Errorf("want ErrBadSig or ErrBadToken, got %v", err)
	}
}

func TestRequireAuth_NoCookie(t *testing.T) {
	h := RequireAuth(testSecret, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("downstream handler should not be invoked")
	}))
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, httptest.NewRequest("GET", "/foo", nil))
	if rr.Code != http.StatusUnauthorized {
		t.Errorf("status: got %d, want 401", rr.Code)
	}
}

func TestRequireAuth_GoodCookie_ContextHasClaims(t *testing.T) {
	tok, _, _ := IssueToken(testSecret, 7, "bob", false, "")
	called := false
	h := RequireAuth(testSecret, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		c := FromContext(r.Context())
		if c == nil || c.FriendID != 7 || c.Username != "bob" {
			t.Errorf("claims missing or wrong: %+v", c)
		}
	}))
	req := httptest.NewRequest("GET", "/foo", nil)
	req.AddCookie(&http.Cookie{Name: CookieName, Value: tok})
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if !called {
		t.Error("handler not called despite valid cookie")
	}
	if rr.Code != http.StatusOK {
		t.Errorf("status: got %d, want 200", rr.Code)
	}
}

func TestRequireAdmin_NonAdminGets403(t *testing.T) {
	tok, _, _ := IssueToken(testSecret, 7, "bob", false, "") // not admin
	h := RequireAdmin(testSecret, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("admin handler should not be invoked for non-admin")
	}))
	req := httptest.NewRequest("GET", "/admin/x", nil)
	req.AddCookie(&http.Cookie{Name: CookieName, Value: tok})
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusForbidden {
		t.Errorf("status: got %d, want 403", rr.Code)
	}
	if !strings.Contains(rr.Body.String(), "admin only") {
		t.Errorf("body: %q does not mention admin only", rr.Body.String())
	}
}

func TestRequireAdmin_AdminAllowed(t *testing.T) {
	tok, _, _ := IssueToken(testSecret, 1, "owner", true, "")
	called := false
	h := RequireAdmin(testSecret, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	}))
	req := httptest.NewRequest("GET", "/admin/x", nil)
	req.AddCookie(&http.Cookie{Name: CookieName, Value: tok})
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if !called || rr.Code != http.StatusOK {
		t.Errorf("admin handler not called or wrong status %d", rr.Code)
	}
}
