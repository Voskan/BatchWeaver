package runtime

import "time"

// Clock abstracts time so that the scheduler can be tested deterministically.
// The production implementation is returned by SystemClock; tests may supply a
// controllable clock.
type Clock interface {
	// Now returns the current time.
	Now() time.Time
	// NewTimer returns a Timer that fires after d. A non-positive d fires as soon
	// as possible.
	NewTimer(d time.Duration) Timer
}

// Timer is a one-shot timer whose channel receives once after the configured
// duration. It mirrors the subset of time.Timer the scheduler uses.
type Timer interface {
	// C returns the channel on which the time is delivered.
	C() <-chan time.Time
	// Stop prevents the timer from firing. It reports whether it stopped the
	// timer before it fired.
	Stop() bool
	// Reset changes the timer to fire after d. It reports whether the timer was
	// active. Callers must follow the standard drain rules before Reset.
	Reset(d time.Duration) bool
}

// systemClock is the production Clock backed by the time package.
type systemClock struct{}

// SystemClock returns a Clock backed by the standard library time package. When
// used inside a testing/synctest bubble, its time is the bubble's fake time.
func SystemClock() Clock { return systemClock{} }

func (systemClock) Now() time.Time { return time.Now() }

func (systemClock) NewTimer(d time.Duration) Timer {
	if d <= 0 {
		// Fire essentially immediately while keeping the Timer contract.
		d = time.Nanosecond
	}
	return &systemTimer{t: time.NewTimer(d)}
}

// systemTimer adapts *time.Timer to the Timer interface.
type systemTimer struct {
	t *time.Timer
}

func (s *systemTimer) C() <-chan time.Time { return s.t.C }
func (s *systemTimer) Stop() bool          { return s.t.Stop() }
func (s *systemTimer) Reset(d time.Duration) bool {
	if d <= 0 {
		d = time.Nanosecond
	}
	return s.t.Reset(d)
}
