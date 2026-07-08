package api

import (
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"strings"

	"golang.org/x/crypto/bcrypt"

	"github.com/xiagao/fund-dashboard/backend/middleware"
	"github.com/xiagao/fund-dashboard/backend/store"
)

// clientIP extracts the client IP used to key the login rate limiter.
//
// Only X-Real-IP is trusted, and only when TrustProxyHeaders is set — i.e. the
// sole ingress is our own nginx, which overwrites X-Real-IP with the socket peer
// it actually saw, so a client cannot forge it. CF-Connecting-IP is deliberately
// NOT trusted: nginx passes it through unchanged, so anyone reaching the origin
// directly (bypassing Cloudflare) can spoof it. When headers aren't trusted, the
// raw socket peer is the only honest source.
func (s *Server) clientIP(r *http.Request) string {
	if s.TrustProxyHeaders {
		if ip := r.Header.Get("X-Real-IP"); ip != "" {
			return ip
		}
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

type loginReq struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	var in loginReq
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeErr(w, http.StatusBadRequest, "bad json")
		return
	}
	if in.Username == "" || in.Password == "" {
		writeErr(w, http.StatusBadRequest, "username and password required")
		return
	}
	// Brute-force guard: bcrypt makes each attempt slow, but friend-chosen
	// passwords deserve a hard cap too. Two keys — a tight per-username+IP one
	// (so one friend fat-fingering can't lock everyone out) and a looser
	// per-username one that survives an attacker rotating IPs / forging headers.
	uname := strings.ToLower(in.Username)
	ipKey := "ip|" + uname + "|" + s.clientIP(r)
	userKey := "user|" + uname
	if !s.limiter().allow(ipKey, loginMaxFailures) || !s.limiter().allow(userKey, loginMaxFailuresPerUser) {
		writeErr(w, http.StatusTooManyRequests, "too many failed attempts, try again later")
		return
	}
	recordFailure := func() {
		s.limiter().fail(ipKey)
		s.limiter().fail(userKey)
	}
	f, err := store.GetFriendByUsername(r.Context(), s.DB, in.Username)
	if errors.Is(err, store.ErrFriendNotFound) {
		// Constant-time-ish: still hash the password to avoid leaking timing.
		bcrypt.CompareHashAndPassword([]byte("$2a$10$invalidsaltinvalidsaltinvalidsaltinvalidsa"), []byte(in.Password))
		recordFailure()
		writeErr(w, http.StatusUnauthorized, "invalid credentials")
		return
	}
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "lookup failed")
		return
	}
	if err := bcrypt.CompareHashAndPassword([]byte(f.PasswordHash), []byte(in.Password)); err != nil {
		recordFailure()
		writeErr(w, http.StatusUnauthorized, "invalid credentials")
		return
	}
	s.limiter().reset(ipKey)
	s.limiter().reset(userKey)

	tok, exp, err := middleware.IssueToken(s.JWTSecret, f.ID, f.Username, f.IsAdmin, middleware.PasswordVersion(f.PasswordHash))
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "issue token failed")
		return
	}
	middleware.SetSessionCookie(w, tok, exp, s.CookieSecure)
	store.WriteAudit(r.Context(), s.DB, f.Username, "login", map[string]any{"friend_id": f.ID})
	writeJSON(w, http.StatusOK, map[string]any{
		"id":       f.ID,
		"username": f.Username,
		"name":     f.Name,
		"is_admin": f.IsAdmin,
	})
}

func (s *Server) handleLogout(w http.ResponseWriter, r *http.Request) {
	c := middleware.FromContext(r.Context())
	if c != nil {
		store.WriteAudit(r.Context(), s.DB, c.Username, "logout", nil)
	}
	middleware.ClearSessionCookie(w, s.CookieSecure)
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleMe(w http.ResponseWriter, r *http.Request) {
	c := middleware.FromContext(r.Context())
	f, err := store.GetFriendByID(r.Context(), s.DB, c.FriendID)
	if err != nil {
		writeErr(w, http.StatusNotFound, "friend gone")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"id":       f.ID,
		"username": f.Username,
		"name":     f.Name,
		"is_admin": f.IsAdmin,
	})
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func writeErr(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}
