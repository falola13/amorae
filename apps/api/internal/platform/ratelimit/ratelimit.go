// Package ratelimit counts events per key in fixed time windows: "at most
// 10 login attempts per email per 15 minutes". It's in-memory, so limits are
// per process. That's correct while the API runs as one instance. When it
// scales out, put a Redis-backed type with the same Allow method behind the
// same consumer interfaces; only internal/app changes.
package ratelimit

import (
	"sync"
	"time"
)

type window struct {
	count   int
	resetAt time.Time
}

// Limiter allows `limit` events per key in each window. The zero value is not
// usable; construct one with New.
type Limiter struct {
	limit  int
	period time.Duration
	now    func() time.Time

	mu        sync.Mutex
	windows   map[string]*window
	nextSweep time.Time
}

func New(limit int, period time.Duration, now func() time.Time) *Limiter {
	return &Limiter{
		limit:   limit,
		period:  period,
		now:     now,
		windows: make(map[string]*window),
	}
}

// Allow records one event for key and reports whether it is within the
// limit. When it isn't, retryAfter is how long until the key's window resets.
func (l *Limiter) Allow(key string) (allowed bool, retryAfter time.Duration) {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := l.now()
	l.sweep(now)

	w, ok := l.windows[key]
	if !ok || !now.Before(w.resetAt) {
		w = &window{resetAt: now.Add(l.period)}
		l.windows[key] = w
	}
	w.count++

	if w.count > l.limit {
		return false, w.resetAt.Sub(now)
	}
	return true, 0
}

// sweep drops expired windows at most once per period, so memory holds only
// keys seen recently and stays bounded however many distinct keys arrive.
func (l *Limiter) sweep(now time.Time) {
	if now.Before(l.nextSweep) {
		return
	}
	for key, w := range l.windows {
		if !now.Before(w.resetAt) {
			delete(l.windows, key)
		}
	}
	l.nextSweep = now.Add(l.period)
}
