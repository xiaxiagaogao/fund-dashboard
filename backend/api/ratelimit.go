package api

import (
	"sync"
	"time"
)

const (
	// loginMaxFailures failed attempts within loginWindow lock the key out
	// until the oldest failure ages past the window.
	loginMaxFailures = 8
	loginWindow      = 10 * time.Minute
)

// loginLimiter is a tiny in-memory sliding-window counter of FAILED login
// attempts, keyed by username+IP. Successful logins clear the key. State is
// process-local and lost on restart — fine for a single-instance dashboard;
// bcrypt remains the per-attempt cost floor either way.
type loginLimiter struct {
	mu       sync.Mutex
	failures map[string][]time.Time
}

func newLoginLimiter() *loginLimiter {
	return &loginLimiter{failures: map[string][]time.Time{}}
}

// allow reports whether another attempt may proceed for key.
func (l *loginLimiter) allow(key string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	return len(l.prune(key)) < loginMaxFailures
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
