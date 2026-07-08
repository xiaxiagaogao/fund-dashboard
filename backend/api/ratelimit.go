package api

import (
	"sync"
	"time"
)

const (
	// loginMaxFailures failed attempts from one username+IP within loginWindow
	// lock that pair out until the oldest failure ages past the window. Tight,
	// because a single client fat-fingering their password won't hit it.
	loginMaxFailures = 8
	// loginMaxFailuresPerUser is an IP-independent ceiling on failed attempts
	// against a single username within the window. It backstops the per-IP
	// counter against an attacker who rotates source IPs (or forges proxy
	// headers): even spread across many IPs, one account can't be guessed more
	// than this many times per window. Set generously so ordinary retries never
	// reach it; the tradeoff is that a determined attacker can keep one specific
	// account locked, which self-heals as the window slides.
	loginMaxFailuresPerUser = 50
	loginWindow             = 10 * time.Minute
)

// loginLimiter is a tiny in-memory sliding-window counter of FAILED login
// attempts. Callers key it both per username+IP and per username alone (see
// handleLogin). Successful logins clear the keys. State is process-local and
// lost on restart — fine for a single-instance dashboard; bcrypt remains the
// per-attempt cost floor either way.
type loginLimiter struct {
	mu       sync.Mutex
	failures map[string][]time.Time
}

func newLoginLimiter() *loginLimiter {
	return &loginLimiter{failures: map[string][]time.Time{}}
}

// allow reports whether another attempt may proceed for key, given the maximum
// failures tolerated for that key within the window.
func (l *loginLimiter) allow(key string, max int) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	return len(l.prune(key)) < max
}

// fail records one failed attempt for key.
func (l *loginLimiter) fail(key string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.failures[key] = append(l.prune(key), time.Now())
}

// reset clears key after a successful login.
func (l *loginLimiter) reset(key string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	delete(l.failures, key)
}

// prune drops failures older than the window. Caller holds l.mu.
func (l *loginLimiter) prune(key string) []time.Time {
	cutoff := time.Now().Add(-loginWindow)
	kept := l.failures[key][:0]
	for _, t := range l.failures[key] {
		if t.After(cutoff) {
			kept = append(kept, t)
		}
	}
	if len(kept) == 0 {
		delete(l.failures, key)
		return nil
	}
	l.failures[key] = kept
	return kept
}
