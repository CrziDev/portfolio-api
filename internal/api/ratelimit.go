package api

import (
	"net"
	"net/http"
	"strings"
	"sync"
	"time"
)

// rateLimit allows each key at most `limit` events per `window`. It's kept in memory,
// so a restart resets it, which is fine for slowing down comment spam.
type rateLimit struct {
	limit  int
	window time.Duration

	mu     sync.Mutex
	events map[string][]time.Time
}

func newRateLimit(limit int, window time.Duration) *rateLimit {
	return &rateLimit{limit: limit, window: window, events: map[string][]time.Time{}}
}

// allow records an event for key and reports whether it's within the limit.
func (l *rateLimit) allow(key string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := time.Now()
	// Forget events that have left the window, for every key, so the map doesn't grow without bound.
	for k, times := range l.events {
		recent := times[:0]
		for _, t := range times {
			if now.Sub(t) < l.window {
				recent = append(recent, t)
			}
		}
		if len(recent) == 0 {
			delete(l.events, k)
		} else {
			l.events[k] = recent
		}
	}

	if len(l.events[key]) >= l.limit {
		return false
	}
	l.events[key] = append(l.events[key], now)
	return true
}

// clientIP is the visitor's address. In production the API is only reachable through Caddy,
// which sets X-Forwarded-For to the address it saw; the last entry is the one Caddy added.
func clientIP(r *http.Request) string {
	if forwarded := r.Header.Get("X-Forwarded-For"); forwarded != "" {
		parts := strings.Split(forwarded, ",")
		return strings.TrimSpace(parts[len(parts)-1])
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}
