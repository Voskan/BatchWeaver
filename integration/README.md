# Integration

This directory is reserved for integration tests and end-to-end material that
exercise BatchWeaver across multiple components (for example the analyzer, the
transformation, and the runtime together).

It is empty of code today because those components do not exist yet. Integration
material added here must be deterministic and must not depend on external network
services unless a test explicitly opts in and is skipped by default.
