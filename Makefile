# BatchWeaver developer Makefile.
# Run `make help` for a list of targets.

# Fail fast and treat unset variables as errors inside recipes.
SHELL := /usr/bin/env bash
.SHELLFLAGS := -eu -o pipefail -c

MODULE := github.com/Voskan/BatchWeaver
BIN_DIR := bin
BINARY := $(BIN_DIR)/batchweaver
PKG := ./cmd/batchweaver

# Pinned development tool versions. Keep in sync with tools/go.mod.
GOLANGCI_LINT_VERSION := v2.12.2
GOVULNCHECK_VERSION := v1.6.0

# The pinned toolchain. Building the linters via `go run tool@version` must use
# a Go at least as new as this module's language version, so tool targets set
# GOTOOLCHAIN explicitly rather than relying on the caller's default.
GO_TOOLCHAIN := go1.26.5

# Version metadata injected into the binary at build time. Release builds should
# pass an explicit BUILD_DATE for reproducibility.
VERSION ?= dev
COMMIT ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo unknown)
BUILD_DATE ?= unknown

LDFLAGS := -X $(MODULE)/internal/buildinfo.version=$(VERSION) \
	-X $(MODULE)/internal/buildinfo.commit=$(COMMIT) \
	-X $(MODULE)/internal/buildinfo.buildDate=$(BUILD_DATE)

.DEFAULT_GOAL := help

.PHONY: help
help: ## Show this help
	@grep -hE '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) \
		| awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-14s\033[0m %s\n", $$1, $$2}'

.PHONY: fmt
fmt: ## Format Go source with gofmt
	git ls-files -z '*.go' | xargs -0 gofmt -w

.PHONY: fmt-check
fmt-check: ## Verify all Go source is gofmt-formatted
	@unformatted="$$(git ls-files -z '*.go' | xargs -0 gofmt -l)"; \
	if [ -n "$$unformatted" ]; then \
		echo "The following files are not gofmt-formatted:"; \
		echo "$$unformatted"; \
		exit 1; \
	fi

.PHONY: vet
vet: ## Run go vet
	go vet ./...

.PHONY: test
test: ## Run unit tests
	go test ./...

.PHONY: test-race
test-race: ## Run unit tests with the race detector
	go test -race ./...

.PHONY: test-cover
test-cover: ## Run unit tests and write coverage.out
	go test -coverprofile=coverage.out ./...
	@go tool cover -func=coverage.out | tail -1

.PHONY: assurance
assurance: ## Run API, schema, differential, mutation, fault, soak, and budget tests
	go test ./internal/release ./internal/assurance -count=1

.PHONY: extension-check
extension-check: ## Clean-install and verify the VS Code extension (requires Node 22)
	cd editors/vscode && npm ci && npm audit --audit-level=high
	cd editors/vscode && npm run lint && npm run typecheck && npm run compile && npm test && npm run package

.PHONY: release-snapshot
release-snapshot: build ## Build and verify an unpublished snapshot under dist/
	$(BINARY) release build --snapshot --output dist
	$(BINARY) release verify dist/release-manifest.json

.PHONY: build
build: ## Build the CLI into bin/batchweaver
	@mkdir -p $(BIN_DIR)
	go build -trimpath -ldflags '$(LDFLAGS)' -o $(BINARY) $(PKG)

.PHONY: run
run: ## Build and run the CLI (use ARGS="version")
	go run $(PKG) $(ARGS)

.PHONY: lint
lint: ## Run golangci-lint
	GOTOOLCHAIN=$(GO_TOOLCHAIN) go run github.com/golangci/golangci-lint/v2/cmd/golangci-lint@$(GOLANGCI_LINT_VERSION) run ./...

.PHONY: vulncheck
vulncheck: ## Scan for known vulnerabilities
	GOTOOLCHAIN=$(GO_TOOLCHAIN) go run golang.org/x/vuln/cmd/govulncheck@$(GOVULNCHECK_VERSION) ./...

.PHONY: docs-check
docs-check: ## Lint Markdown and YAML (skips a linter if it is not installed)
	@if command -v markdownlint >/dev/null 2>&1; then \
		markdownlint '**/*.md' --ignore '**/node_modules/**'; \
	elif command -v npx >/dev/null 2>&1; then \
		npx --yes markdownlint-cli '**/*.md' --ignore '**/node_modules/**'; \
	else \
		echo "markdownlint not found; skipping Markdown lint"; \
	fi
	@if command -v yamllint >/dev/null 2>&1; then \
		yamllint -c .yamllint.yml .; \
	elif command -v python3 >/dev/null 2>&1 && python3 -c 'import yamllint' >/dev/null 2>&1; then \
		python3 -m yamllint -c .yamllint.yml .; \
	else \
		echo "yamllint not found; skipping YAML lint"; \
	fi

.PHONY: site
site: ## Build the deterministic documentation site under _site/
	scripts/build-docs-site.sh _site

.PHONY: release-script-tests
release-script-tests: ## Verify publication helpers fail closed
	scripts/release-script-tests.sh

.PHONY: check
check: fmt-check vet test test-race assurance build lint vulncheck docs-check site release-script-tests ## Run all mandatory local gates

.PHONY: clean
clean: ## Remove build and coverage output
	rm -rf $(BIN_DIR) coverage.out coverage.html
