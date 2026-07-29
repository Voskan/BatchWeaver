package runtime

import "fmt"

// EngineOption configures an Engine during construction. Options validate before
// the engine becomes externally visible.
type EngineOption func(*Engine) error

// WithClock sets the clock used by the scheduler. A nil clock is rejected.
func WithClock(clock Clock) EngineOption {
	return func(e *Engine) error {
		if clock == nil {
			return fmt.Errorf("%w: clock is nil", ErrInvalidOption)
		}
		e.clock = clock
		return nil
	}
}

// WithDefaultQueueLimits sets the default per-operation queue limits, used when a
// binding does not override them.
func WithDefaultQueueLimits(limits QueueLimits) EngineOption {
	return func(e *Engine) error {
		if err := limits.Validate(); err != nil {
			return err
		}
		e.defaultLimits = limits
		return nil
	}
}

// WithDefaultOverflowPolicy sets the default overflow policy.
func WithDefaultOverflowPolicy(policy OverflowPolicy) EngineOption {
	return func(e *Engine) error {
		if policy > OverflowBlock {
			return fmt.Errorf("%w: unknown overflow policy", ErrInvalidOption)
		}
		e.defaultOverflow = policy
		return nil
	}
}

// WithPanicPolicy sets the provider/callback panic policy.
func WithPanicPolicy(policy PanicPolicy) EngineOption {
	return func(e *Engine) error {
		if policy > PanicPolicyRepanic {
			return fmt.Errorf("%w: unknown panic policy", ErrInvalidOption)
		}
		e.panicPolicy = policy
		return nil
	}
}

// WithHooks sets the event hooks invoked by the engine.
func WithHooks(hooks Hooks) EngineOption {
	return func(e *Engine) error {
		e.hooks = hooks
		return nil
	}
}

// WithStatistics enables or disables statistics collection. Statistics are
// enabled by default; disabling them reduces per-request overhead.
func WithStatistics(enabled bool) EngineOption {
	return func(e *Engine) error {
		e.statsEnabled = enabled
		return nil
	}
}
