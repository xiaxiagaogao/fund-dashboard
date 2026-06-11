// Package middleware provides JWT cookie auth that wraps net/http handlers.
//
// The token is signed HMAC-SHA256, payload carries friend_id + is_admin + exp.
// Cookies are HttpOnly + SameSite=Strict + Secure (when behind Cloudflare).
//
// Why hand-rolled and not jwt-go? The dashboard issues exactly one token shape
// and verifies it in exactly one middleware. A 100-line implementation is
// auditable; pulling jwt-go drags in claims registries and signing-method
// negotiation we'll never use.
package middleware

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"
)

const (
	CookieName = "fund_session"
	cookieTTL  = 7 * 24 * time.Hour
)

// Claims is the JWT payload. Tiny on purpose.
type Claims struct {
	FriendID int64  `json:"sub"`
	Username string `json:"u"`
	IsAdmin  bool   `json:"adm"`
	Exp      int64  `json:"exp"`
	// Pwv is a fingerprint of the password hash current at issue time. The
	// session-validation layer (api.Server.validateSession) compares it against
	// the live hash so a password change immediately invalidates old tokens.
	Pwv string `json:"pwv,omitempty"`
}

// PasswordVersion derives a short fingerprint of a bcrypt password hash for
// embedding in Claims.Pwv. Not secret material — just a change detector.
func PasswordVersion(passwordHash string) string {
	sum := sha256.Sum256([]byte(passwordHash))
	return base64.RawURLEncoding.EncodeToString(sum[:])[:12]
}

// ctxKey is a private type so other packages can't accidentally collide.
type ctxKey int

const ctxClaimsKey ctxKey = 1

// FromContext returns the authenticated claims, or nil if the request was not
// authenticated (handler reached via a public route).
func FromContext(ctx context.Context) *Claims {
	if c, ok := ctx.Value(ctxClaimsKey).(*Claims); ok {
		return c
	}
	return nil
}

// WithClaims is for tests / login handler that need to inject claims directly.
func WithClaims(ctx context.Context, c *Claims) context.Context {
	return context.WithValue(ctx, ctxClaimsKey, c)
}

// IssueToken signs a fresh token for a friend and returns the cookie-ready string.
// pwv is the PasswordVersion of the friend's current password hash.
func IssueToken(secret string, friendID int64, username string, isAdmin bool, pwv string) (string, time.Time, error) {
	exp := time.Now().Add(cookieTTL)
	c := Claims{FriendID: friendID, Username: username, IsAdmin: isAdmin, Exp: exp.Unix(), Pwv: pwv}
	payload, err := json.Marshal(c)
	if err != nil {
		return "", time.Time{}, err
	}
	body := base64.RawURLEncoding.EncodeToString(payload)
	sig := signHMAC(secret, body)
	return body + "." + sig, exp, nil
}

// SetSessionCookie writes the issued token onto the response.
// Secure=true assumes deployment behind HTTPS (Cloudflare) — toggle off for local http dev.
func SetSessionCookie(w http.ResponseWriter, token string, exp time.Time, secure bool) {
	http.SetCookie(w, &http.Cookie{
		Name:     CookieName,
		Value:    token,
		Path:     "/",
		Expires:  exp,
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteStrictMode,
	})
}

func ClearSessionCookie(w http.ResponseWriter, secure bool) {
	http.SetCookie(w, &http.Cookie{
		Name:     CookieName,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteStrictMode,
	})
}

// VerifyToken returns the parsed claims if signature + exp are good.
var (
	ErrBadToken = errors.New("middleware: malformed token")
	ErrBadSig   = errors.New("middleware: signature mismatch")
	ErrExpired  = errors.New("middleware: token expired")
)

func VerifyToken(secret, token string) (*Claims, error) {
	body, sig, ok := strings.Cut(token, ".")
	if !ok {
		return nil, ErrBadToken
	}
	expected := signHMAC(secret, body)
	if !hmac.Equal([]byte(expected), []byte(sig)) {
		return nil, ErrBadSig
	}
	payload, err := base64.RawURLEncoding.DecodeString(body)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrBadToken, err)
	}
	var c Claims
	if err := json.Unmarshal(payload, &c); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrBadToken, err)
	}
	if time.Now().Unix() > c.Exp {
		return nil, ErrExpired
	}
	return &c, nil
}

// RequireAuth wraps a handler so it returns 401 unless the cookie carries a
// valid token. The verified Claims are attached to the request context.
func RequireAuth(secret string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie(CookieName)
		if err != nil {
			http.Error(w, "unauthenticated", http.StatusUnauthorized)
			return
		}
		claims, err := VerifyToken(secret, cookie.Value)
		if err != nil {
			http.Error(w, "invalid session", http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r.WithContext(WithClaims(r.Context(), claims)))
	})
}

// RequireAdmin chains on top of RequireAuth and additionally enforces is_admin.
func RequireAdmin(secret string, next http.Handler) http.Handler {
	return RequireAuth(secret, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c := FromContext(r.Context())
		if c == nil || !c.IsAdmin {
			http.Error(w, "admin only", http.StatusForbidden)
			return
		}
		next.ServeHTTP(w, r)
	}))
}

func signHMAC(secret, body string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(body))
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}
