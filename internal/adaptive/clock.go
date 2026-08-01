package adaptive

import (
	"sync"
	"time"
)

// Clock abstracts time so collection, controller decay, replay, and simulation
// are deterministic under test. The production implementation delegates to the
// time package; tests supply a FakeClock.
type Clock interface {
	// Now returns the current time.
	Now() time.Time
}

// systemClock is the production Clock.
type systemClock struct{}

// SystemClock returns a Clock backed by the standard library.
func SystemClock() Clock { return systemClock{} }

func (systemClock) Now() time.Time { return time.Now() }

// FakeClock is a manually advanced Clock for deterministic tests and replay. It
// is safe for concurrent use.
type FakeClock struct {
	mu  sync.Mutex
	now time.Time
}

// NewFakeClock returns a FakeClock set to start.
func NewFakeClock(start time.Time) *FakeClock {
	return &FakeClock{now: start}
}

// Now returns the current fake time.
func (c *FakeClock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.now
}

// Advance moves the fake clock forward by d.
func (c *FakeClock) Advance(d time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.now = c.now.Add(d)
}

// Set moves the fake clock to t.
func (c *FakeClock) Set(t time.Time) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.now = t
}
