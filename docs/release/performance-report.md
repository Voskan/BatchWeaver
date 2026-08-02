# Performance Assurance Report

The baseline environment is Apple arm64, macOS, Go 1.26.5. Machine-readable
budgets live in `release/performance-budgets.json`. The deterministic assurance
harness runs at least 20 allocation samples over 1,024 logical requests and 25
wall-time samples. The wall limit is a coarse release safety bound, not a claim
of statistically significant improvement.

Compiler, runtime, adapter, adaptive, and daemon benchmarks remain available in
their packages. A release comparison must record CPU, memory, OS, toolchain,
commit, raw `go test -bench` output, and benchstat confidence before calling a
change a regression. A single noisy measurement cannot block a release. Any
exception needs an owner, reason, tracking issue, and expiry; none exist now.
