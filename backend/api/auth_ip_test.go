package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

// When not behind a trusted proxy, forwarding headers are attacker-controlled
// and must be ignored — the socket peer is the only trustworthy source.
func TestClientIP_IgnoresHeadersWhenUntrusted(t *testing.T) {
	s := &Server{TrustProxyHeaders: false}
	req := httptest.NewRequest("POST", "/api/login", nil)
	req.RemoteAddr = "198.51.100.9:44444"
	req.Header.Set("X-Real-IP", "10.0.0.1")
	req.Header.Set("CF-Connecting-IP", "10.0.0.2")

	if got := s.clientIP(req); got != "198.51.100.9" {
		t.Errorf("untrusted proxy: got %q, want the socket peer 198.51.100.9", got)
	}
}

// Behind a trusted proxy, nginx overwrites X-Real-IP with the real socket peer
// it saw, so that header is trustworthy and identifies the client.
func TestClientIP_TrustsXRealIPWhenTrusted(t *testing.T) {
	s := &Server{TrustProxyHeaders: true}
	req := httptest.NewRequest("POST", "/api/login", nil)
	req.RemoteAddr = "127.0.0.1:33333"
	req.Header.Set("X-Real-IP", "203.0.113.44")

	if got := s.clientIP(req); got != "203.0.113.44" {
		t.Errorf("trusted proxy: got %q, want X-Real-IP 203.0.113.44", got)
	}
}

// CF-Connecting-IP is never honored: nginx does not strip it, so a client that
// reaches the origin directly can forge it. Only nginx's own X-Real-IP counts.
func TestClientIP_NeverTrustsCFConnectingIP(t *testing.T) {
	s := &Server{TrustProxyHeaders: true}
	req := httptest.NewRequest("POST", "/api/login", nil)
	req.RemoteAddr = "127.0.0.1:33333"
	req.Header.Set("CF-Connecting-IP", "1.2.3.4") // forged, no X-Real-IP present

	if got := s.clientIP(req); got == "1.2.3.4" {
		t.Errorf("CF-Connecting-IP must not be trusted, but clientIP returned it")
	}
	if got := s.clientIP(req); got != "127.0.0.1" {
		t.Errorf("with no X-Real-IP, expected socket peer 127.0.0.1, got %q", got)
	}
}

// postLoginFrom drives handleLogin with an explicit socket peer so tests can
// simulate an attacker rotating source IPs.
func postLoginFrom(s *Server, username, password, remoteAddr string) *httptest.ResponseRecorder {
	buf, _ := json.Marshal(map[string]string{"username": username, "password": password})
	req := httptest.NewRequest("POST", "/api/login", bytes.NewReader(buf))
	req.RemoteAddr = remoteAddr
	rr := httptest.NewRecorder()
	s.handleLogin(rr, req)
	return rr
}

// Rotating the source IP defeats the per-IP counter, but the IP-independent
// per-username floor still bounds total failed attempts against one account.
func TestLogin_PerUsernameFloor(t *testing.T) {
	s, _, _ := newTestServer(t)

	for i := 0; i < loginMaxFailuresPerUser; i++ {
		ip := fmt.Sprintf("192.0.2.%d:40000", i%254+1)
		if rr := postLoginFrom(s, "admin", "wrong-password", ip); rr.Code != http.StatusUnauthorized {
			t.Fatalf("attempt %d from %s: got %d want 401 (%s)", i+1, ip, rr.Code, rr.Body.String())
		}
	}
	// A brand-new IP still gets rejected: the username floor has tripped.
	if rr := postLoginFrom(s, "admin", "wrong-password", "192.0.2.250:12345"); rr.Code != http.StatusTooManyRequests {
		t.Errorf("after %d per-username failures: got %d want 429", loginMaxFailuresPerUser, rr.Code)
	}
	// A different username is unaffected by another account's floor.
	if rr := postLoginFrom(s, "someone-else", "whatever", "192.0.2.251:12345"); rr.Code != http.StatusUnauthorized {
		t.Errorf("unrelated username: got %d want 401", rr.Code)
	}
}
