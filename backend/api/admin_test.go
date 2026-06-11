package api

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"golang.org/x/crypto/bcrypt"

	"github.com/xiagao/fund-dashboard/backend/middleware"
	"github.com/xiagao/fund-dashboard/backend/store"
)

const testJWTSecret = "api-test-secret-xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx"

// newTestServer spins up a Server over a fresh on-disk fund.db (real WAL +
// pragma path) with one admin friend. Returns the server, the admin's id,
// and the admin's password hash (for token fingerprints).
func newTestServer(t *testing.T) (*Server, int64, string) {
	t.Helper()
	db, err := store.Open(filepath.Join(t.TempDir(), "fund.db"))
	if err != nil {
		t.Fatalf("open test db: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	hash, err := bcrypt.GenerateFromPassword([]byte("correct-password"), bcrypt.MinCost)
	if err != nil {
		t.Fatal(err)
	}
	adminID, err := store.CreateFriend(context.Background(), db, "Admin", "admin", string(hash), true)
	if err != nil {
		t.Fatalf("create admin: %v", err)
	}
	return &Server{DB: db, JWTSecret: testJWTSecret}, adminID, string(hash)
}

// postCashEvent drives handleAdminCashEvent directly with admin claims injected.
func postCashEvent(t *testing.T, s *Server, adminID int64, body map[string]any) *httptest.ResponseRecorder {
	t.Helper()
	buf, _ := json.Marshal(body)
	req := httptest.NewRequest("POST", "/api/admin/cash-events", bytes.NewReader(buf))
	req = req.WithContext(middleware.WithClaims(req.Context(),
		&middleware.Claims{FriendID: adminID, Username: "admin", IsAdmin: true}))
	rr := httptest.NewRecorder()
	s.handleAdminCashEvent(rr, req)
	return rr
}

func countSnapshots(t *testing.T, db *sql.DB) int {
	t.Helper()
	var n int
	if err := db.QueryRow(`SELECT COUNT(*) FROM nav_snapshots`).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}

// Manual-NAV deposit: shares = amount/nav, and a current-time event writes an
// at-event snapshot.
func TestAdminCashEvent_ManualNAV(t *testing.T) {
	s, adminID, _ := newTestServer(t)

	rr := postCashEvent(t, s, adminID, map[string]any{
		"username": "admin", "type": "deposit", "amount_usdt": 500.0, "manual_nav": 1.25,
	})
	if rr.Code != http.StatusCreated {
		t.Fatalf("status: got %d body=%s", rr.Code, rr.Body.String())
	}
	var resp struct {
		SharesDelta float64 `json:"shares_delta"`
		NAVAtEvent  float64 `json:"nav_at_event"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if resp.NAVAtEvent != 1.25 || resp.SharesDelta != 400 {
		t.Errorf("nav=%v sharesDelta=%v, want 1.25 / 400", resp.NAVAtEvent, resp.SharesDelta)
	}
	if n := countSnapshots(t, s.DB); n != 1 {
		t.Errorf("snapshots after current-time event: got %d want 1", n)
	}
}

// A heavily backdated event must NOT write an at-event snapshot — the equity
// is today's, stamping it onto a past taken_at would fake the curve.
func TestAdminCashEvent_BackdatedSkipsSnapshot(t *testing.T) {
	s, adminID, _ := newTestServer(t)

	occurred := time.Now().Add(-48 * time.Hour).UnixMilli()
	rr := postCashEvent(t, s, adminID, map[string]any{
		"username": "admin", "type": "deposit", "amount_usdt": 500.0,
		"manual_nav": 1.0, "occurred_at_ms": occurred,
	})
	if rr.Code != http.StatusCreated {
		t.Fatalf("status: got %d body=%s", rr.Code, rr.Body.String())
	}
	if n := countSnapshots(t, s.DB); n != 0 {
		t.Errorf("snapshots after backdated event: got %d want 0", n)
	}
	// The ledger row itself must still exist.
	shares, err := store.TotalShares(context.Background(), s.DB)
	if err != nil || shares != 500 {
		t.Errorf("total shares: got %v err=%v, want 500", shares, err)
	}
}

// Withdrawing more shares than held is rejected.
func TestAdminCashEvent_OverWithdrawRejected(t *testing.T) {
	s, adminID, _ := newTestServer(t)

	if rr := postCashEvent(t, s, adminID, map[string]any{
		"username": "admin", "type": "deposit", "amount_usdt": 100.0, "manual_nav": 1.0,
	}); rr.Code != http.StatusCreated {
		t.Fatalf("seed deposit: %d %s", rr.Code, rr.Body.String())
	}
	rr := postCashEvent(t, s, adminID, map[string]any{
		"username": "admin", "type": "withdraw", "amount_usdt": 200.0, "manual_nav": 1.0,
	})
	if rr.Code != http.StatusBadRequest {
		t.Errorf("over-withdraw: got %d want 400 (body=%s)", rr.Code, rr.Body.String())
	}
}

func postLogin(s *Server, username, password string) *httptest.ResponseRecorder {
	buf, _ := json.Marshal(map[string]string{"username": username, "password": password})
	req := httptest.NewRequest("POST", "/api/login", bytes.NewReader(buf))
	req.RemoteAddr = "203.0.113.7:55555"
	rr := httptest.NewRecorder()
	s.handleLogin(rr, req)
	return rr
}

// After loginMaxFailures failed attempts the key is locked out — even for the
// correct password — until the window slides.
func TestLogin_RateLimit(t *testing.T) {
	s, _, _ := newTestServer(t)

	for i := 0; i < loginMaxFailures; i++ {
		if rr := postLogin(s, "admin", "wrong-password"); rr.Code != http.StatusUnauthorized {
			t.Fatalf("attempt %d: got %d want 401", i+1, rr.Code)
		}
	}
	if rr := postLogin(s, "admin", "wrong-password"); rr.Code != http.StatusTooManyRequests {
		t.Errorf("after %d failures: got %d want 429", loginMaxFailures, rr.Code)
	}
	if rr := postLogin(s, "admin", "correct-password"); rr.Code != http.StatusTooManyRequests {
		t.Errorf("correct password while locked out: got %d want 429", rr.Code)
	}
	// A different username+IP key is unaffected.
	if rr := postLogin(s, "someone-else", "whatever"); rr.Code != http.StatusUnauthorized {
		t.Errorf("other user: got %d want 401", rr.Code)
	}
}

// A successful login resets the failure counter.
func TestLogin_SuccessResetsLimiter(t *testing.T) {
	s, _, _ := newTestServer(t)

	for i := 0; i < loginMaxFailures-1; i++ {
		postLogin(s, "admin", "wrong-password")
	}
	if rr := postLogin(s, "admin", "correct-password"); rr.Code != http.StatusOK {
		t.Fatalf("login: got %d want 200 (%s)", rr.Code, rr.Body.String())
	}
	// Counter cleared — a fresh failure is 401, not 429.
	if rr := postLogin(s, "admin", "wrong-password"); rr.Code != http.StatusUnauthorized {
		t.Errorf("post-reset failure: got %d want 401", rr.Code)
	}
}

// Tokens carry a password-hash fingerprint; changing the password (or deleting
// the friend) kills live sessions at the validateSession layer.
func TestSessionInvalidation_OnPasswordChange(t *testing.T) {
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
		t.Fatalf("valid session: got %d want 200", code)
	}

	newHash, _ := bcrypt.GenerateFromPassword([]byte("brand-new-password"), bcrypt.MinCost)
	if _, err := s.DB.Exec(`UPDATE friends SET password_hash = ? WHERE id = ?`, string(newHash), adminID); err != nil {
		t.Fatal(err)
	}
	if code := doReq(); code != http.StatusUnauthorized {
		t.Errorf("after password change: got %d want 401", code)
	}
}
