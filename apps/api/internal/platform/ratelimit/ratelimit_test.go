package ratelimit

import (
	"sync"
	"testing"
	"time"
)

type clock struct{ t time.Time }

func (c *clock) now() time.Time          { return c.t }
func (c *clock) advance(d time.Duration) { c.t = c.t.Add(d) }

func TestAllow_BlocksAfterLimitAndResetsAfterWindow(t *testing.T) {
	c := &clock{t: time.Date(2026, 9, 21, 12, 0, 0, 0, time.UTC)}
	l := New(3, time.Minute, c.now)

	for i := 1; i <= 3; i++ {
		if ok, _ := l.Allow("k"); !ok {
			t.Fatalf("attempt %d blocked, want allowed", i)
		}
	}

	c.advance(20 * time.Second)
	ok, retry := l.Allow("k")
	if ok {
		t.Fatal("4th attempt allowed, want blocked")
	}
	if retry != 40*time.Second {
		t.Errorf("retryAfter = %s, want 40s (time left in the window)", retry)
	}

	c.advance(40 * time.Second)
	if ok, _ := l.Allow("k"); !ok {
		t.Fatal("attempt after the window reset was blocked")
	}
}

func TestAllow_KeysAreIndependent(t *testing.T) {
	c := &clock{t: time.Now()}
	l := New(1, time.Minute, c.now)

	l.Allow("a")
	if ok, _ := l.Allow("a"); ok {
		t.Fatal("second event for a allowed, want blocked")
	}
	if ok, _ := l.Allow("b"); !ok {
		t.Fatal("b blocked by a's limit")
	}
}

func TestSweep_DropsExpiredWindows(t *testing.T) {
	c := &clock{t: time.Now()}
	l := New(5, time.Minute, c.now)

	for _, k := range []string{"a", "b", "c"} {
		l.Allow(k)
	}
	c.advance(2 * time.Minute)
	l.Allow("d")

	if n := len(l.windows); n != 1 {
		t.Fatalf("%d windows kept after expiry, want only the fresh one", n)
	}
}

// Run with -race: Allow is called from every request goroutine at once.
func TestAllow_ConcurrentCallsCountExactly(t *testing.T) {
	l := New(100, time.Minute, time.Now)

	var wg sync.WaitGroup
	var mu sync.Mutex
	allowed := 0
	for range 150 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if ok, _ := l.Allow("k"); ok {
				mu.Lock()
				allowed++
				mu.Unlock()
			}
		}()
	}
	wg.Wait()

	if allowed != 100 {
		t.Fatalf("allowed %d of 150 concurrent events, want exactly 100", allowed)
	}
}
