package middleware

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestAttemptLimiter(t *testing.T) {
	now := time.Now()
	l := NewAttemptLimiter()
	l.now = func() time.Time { return now }
	for i := 0; i < 5; i++ {
		if !l.Allow("email", 5) {
			t.Fatal("attempt rejected too early")
		}
	}
	if l.Allow("email", 5) {
		t.Fatal("rate limit was not enforced")
	}
	if !l.Allow("other", 5) {
		t.Fatal("independent key was blocked")
	}
	now = now.Add(15 * time.Minute)
	if !l.Allow("email", 5) {
		t.Fatal("window did not expire")
	}
}

func TestAttemptLimiterCapacity(t *testing.T) {
	now := time.Now()
	l := NewAttemptLimiter()
	l.capacity = 2
	l.now = func() time.Time { return now }
	l.Allow("one", 5)
	l.Allow("two", 5)
	if l.Allow("three", 5) || len(l.entries) != 2 {
		t.Fatal("limiter exceeded bounded capacity")
	}
	now = now.Add(15 * time.Minute)
	if !l.Allow("three", 5) {
		t.Fatal("expired entries were not reclaimed")
	}
}

func TestAttemptLimiterConcurrent(t *testing.T) {
	l := NewAttemptLimiter()
	var allowed atomic.Int32
	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if l.Allow("same-key", 5) {
				allowed.Add(1)
			}
		}()
	}
	wg.Wait()
	if allowed.Load() != 5 {
		t.Fatalf("allowed %d attempts, want 5", allowed.Load())
	}
}
