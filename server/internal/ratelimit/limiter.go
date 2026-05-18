package ratelimit

import (
	"sync"
	"time"
)

// WindowLimiter counts events per key within a sliding-ish fixed window (pruned per Allow call).
type WindowLimiter struct {
	mu     sync.Mutex
	hits   map[string][]time.Time
	window time.Duration
	max    int
}

func New(window time.Duration, max int) *WindowLimiter {
	return &WindowLimiter{
		hits:   map[string][]time.Time{},
		window: window,
		max:    max,
	}
}

// Allow reports whether key is still under the limit; when true, records this hit.
func (l *WindowLimiter) Allow(key string) bool {
	now := time.Now()
	cutoff := now.Add(-l.window)

	l.mu.Lock()
	defer l.mu.Unlock()

	ts := l.hits[key]
	keep := ts[:0]
	for _, t := range ts {
		if t.After(cutoff) {
			keep = append(keep, t)
		}
	}
	if len(keep) >= l.max {
		l.hits[key] = keep
		return false
	}
	keep = append(keep, now)
	l.hits[key] = keep
	return true
}
