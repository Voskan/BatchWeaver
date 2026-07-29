# Getting Started

## Prerequisites

- Go 1.26 or newer. The module pins `toolchain go1.26.5`. With the default
  `GOTOOLCHAIN=auto`, running any `go` command in this repository fetches the
  pinned toolchain automatically if it is not already installed.
- `make` (optional) to run the convenience targets.

## Clone

```bash
git clone https://github.com/Voskan/BatchWeaver.git
cd BatchWeaver
```

## Build

```bash
make build          # produces bin/batchweaver
# or
go build ./cmd/batchweaver
```

## Run

```bash
./bin/batchweaver version
./bin/batchweaver help
# or, without building:
go run ./cmd/batchweaver version
```

Only the `version` and `help` commands exist in this build.

## Test

```bash
make test           # unit tests
make test-race      # unit tests under the race detector
make test-cover     # unit tests with a coverage profile
```

## Run all local checks

```bash
make check
```

`make check` runs formatting verification, `go vet`, tests, race tests, build,
linting, vulnerability scanning, and documentation checks. See
[quality-gates.md](quality-gates.md) for details.
