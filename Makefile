# -----------------------------
# CONFIG
# -----------------------------
GO            ?= go
BUILD_VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)

export BUILD_VERSION

# Source packages measured for coverage (informational only — see cover-func note).
COVER_PKG_LIST := $(shell $(GO) list ./pkg/... | tr '\n' ',' | sed 's/,$$//')

.PHONY: setup help tidy fmt fmt-check vet lint test test-ci build ci \
        cover cover-func godoc mod-verify vuln-check clean

# -----------------------------
# SETUP
# -----------------------------

setup:
	$(GO) mod download
	@echo "Modules downloaded — ready to build (no .env needed: this module has no runtime config)"

help:
	@echo "Available commands:"
	@echo "  make setup           - go mod download (module has no .env / runtime config)"
	@echo "  make tidy            - go mod tidy"
	@echo "  make fmt             - go fmt ./..."
	@echo "  make vet             - go vet all packages"
	@echo "  make lint            - run golangci-lint"
	@echo "  make test            - run all tests"
	@echo "  make test-ci         - run all tests with race detector (used in CI)"
	@echo "  make build           - compile-check all packages (no runnable binary ships from this module)"
	@echo "  make cover           - coverage profile + open HTML report"
	@echo "  make cover-func      - coverage summary by function"
	@echo "  make ci              - tidy + vet + lint + test-ci + build"
	@echo "  make godoc           - serve docs locally via pkgsite (http://localhost:8080)"
	@echo "  make fmt-check       - verify gofmt formatting (no changes applied)"
	@echo "  make mod-verify      - go mod verify (check module download integrity)"
	@echo "  make vuln-check      - govulncheck on library packages"
	@echo "  make clean           - remove test/coverage artefacts"

# -----------------------------
# GO BASICS
# -----------------------------

tidy:
	$(GO) mod tidy

fmt:
	$(GO) fmt ./...

vet:
	$(GO) vet ./...

# -----------------------------
# LINT
# -----------------------------

lint:
	@echo "Running linter..."
	$(GO) tool golangci-lint run

# -----------------------------
# TESTS
# -----------------------------
# Tests stay co-located with source (pkg/dsl, pkg/events use package
# `dsl_test`/`events_test`) — idiomatic Go convention, not the siblings'
# test/unit/ layout. Do not move these files.

test:
	$(GO) test ./pkg/... -count=1 -timeout 60s -v

test-ci:
	$(GO) test ./pkg/... -race -count=1 -timeout 60s

# -----------------------------
# BUILD
# -----------------------------
# No cmd/ binary exists in this module — this is a compile-check only.

build:
	@echo "Verifying library packages compile..."
	$(GO) build ./...

# -----------------------------
# CI
# -----------------------------

ci: tidy vet lint test-ci build

# -----------------------------
# COVERAGE
# -----------------------------
# NOTE: pkg/dsl's only executable code is ExpandCalls (expand.go); the rest
# of pkg/dsl and pkg/enums are declarations. No numeric coverage gate is
# wired into CI — see ARCHITECTURE.md and ci.yml.

cover:
	$(GO) test ./pkg/... -race -count=1 -timeout 60s -coverpkg=$(COVER_PKG_LIST) -coverprofile=coverage.out
	$(GO) tool cover -html=coverage.out

cover-func:
	$(GO) test ./pkg/... -race -count=1 -timeout 60s -coverpkg=$(COVER_PKG_LIST) -coverprofile=coverage.out
	$(GO) tool cover -func=coverage.out

# -----------------------------
# GODOC
# -----------------------------

godoc:
	@echo "Starting pkgsite at http://localhost:8080 — press Ctrl-C to stop"
	$(GO) run golang.org/x/pkgsite/cmd/pkgsite@latest -open .

# -----------------------------
# CHECKS (mirror what CI runs; safe to call locally before pushing)
# -----------------------------

fmt-check:
	@unformatted=$$(gofmt -l pkg/); \
	if [ -n "$$unformatted" ]; then \
		echo "FAIL: unformatted files:"; \
		echo "$$unformatted"; \
		exit 1; \
	fi
	@echo "gofmt: all files formatted"

mod-verify:
	$(GO) mod verify

# vuln-check: scan library packages. Version pinned for CI reproducibility
# (matches platform-events' pattern). To upgrade: bump GOVULNCHECK_VERSION.
GOVULNCHECK_VERSION ?= v1.1.4
vuln-check:
	$(GO) run golang.org/x/vuln/cmd/govulncheck@$(GOVULNCHECK_VERSION) ./pkg/...

# -----------------------------
# CLEAN
# -----------------------------

clean:
	rm -f coverage.out coverage.html coverage_*.out *.coverprofile profile.cov
