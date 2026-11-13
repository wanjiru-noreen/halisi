package middleware

import (
	"sync"
	"time"
)

type rateEntry struct {
	count   int
	expires time.Time
}

// AttemptLimiter is a bounded fixed-window limiter for a single server process.
// Unknown keys fail closed at capacity, preventing unbounded attacker-controlled memory.
type AttemptLimiter struct {
	mu       sync.Mutex
	entries  map[string]rateEntry
	window   time.Duration
	capacity int
	now      func() time.Time
}

func NewAttemptLimiter() *AttemptLimiter {
	return &AttemptLimiter{entries: make(map[string]rateEntry), window: 15 * time.Minute, capacity: 10000, now: time.Now}
}

func (l *AttemptLimiter) Allow(key string, limit int) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	now := l.now()
	entry, exists := l.entries[key]
	if !exists || !now.Before(entry.expires) {
		if len(l.entries) >= l.capacity {
			for k, v := range l.entries {
				if !now.Before(v.expires) {
					delete(l.entries, k)
				}
			}
			if len(l.entries) >= l.capacity {
				return false
			}
		}
		entry = rateEntry{expires: now.Add(l.window)}
	}
	if entry.count >= limit {
		return false
	}
	entry.count++
	l.entries[key] = entry
	return true
}
