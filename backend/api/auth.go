package api

import (
	"encoding/json"
	"errors"
	"net/http"

	"golang.org/x/crypto/bcrypt"

	"github.com/xiagao/fund-dashboard/backend/middleware"
	"github.com/xiagao/fund-dashboard/backend/store"
)

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
	f, err := store.GetFriendByUsername(r.Context(), s.DB, in.Username)
	if errors.Is(err, store.ErrFriendNotFound) {
		// Constant-time-ish: still hash the password to avoid leaking timing.
		bcrypt.CompareHashAndPassword([]byte("$2a$10$invalidsaltinvalidsaltinvalidsaltinvalidsa"), []byte(in.Password))
		writeErr(w, http.StatusUnauthorized, "invalid credentials")
		return
	}
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "lookup failed")
		return
	}
	if err := bcrypt.CompareHashAndPassword([]byte(f.PasswordHash), []byte(in.Password)); err != nil {
		writeErr(w, http.StatusUnauthorized, "invalid credentials")
		return
	}

	tok, exp, err := middleware.IssueToken(s.JWTSecret, f.ID, f.Username, f.IsAdmin)
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
