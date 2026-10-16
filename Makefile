.PHONY: all test test-go test-ts test-py lint lint-go lint-ts lint-py format fmt format-go format-ts format-py format-check upgrade upgrade-go upgrade-ts upgrade-py doc-lint ci clean

all: test format lint

# ---------------------------------------------------------
# Testing
# ---------------------------------------------------------
test: test-go test-ts test-py

test-go:
	go test -race ./pkg/... ./tests/...

test-ts:
	cd ts/logger-ts && bun test

test-py:
	cd py/logger-py && uv run --extra dev pytest tests -v

# ---------------------------------------------------------
# Linting & Static Analysis
# ---------------------------------------------------------
lint: lint-go lint-ts lint-py doc-lint

lint-go:
	go tool golangci-lint run ./pkg/... ./tests/...

lint-ts:
	cd ts/logger-ts && bun run lint

lint-py:
	cd py/logger-py && uv run --extra dev ruff check .

# ---------------------------------------------------------
# Formatting
# ---------------------------------------------------------
format: fmt
fmt: format-go format-ts format-py

format-go:
	gofmt -s -w pkg tests

format-ts:
	cd ts/logger-ts && bun run format

format-py:
	cd py/logger-py && uv run --extra dev ruff format .

format-check:
	@echo "Checking Go formatting..."
	@test -z "$$(gofmt -l pkg tests)" || (echo "Unformatted Go files detected:" && gofmt -l pkg tests && exit 1)
	@echo "Checking Python formatting..."
	cd py/logger-py && uv run --extra dev ruff format --check .
	@echo "Checking TypeScript formatting..."
	cd ts/logger-ts && bun run lint

# ---------------------------------------------------------
# Dependency Management
# ---------------------------------------------------------
upgrade: upgrade-go upgrade-ts upgrade-py

upgrade-go:
	go get -u ./...
	go get -tool github.com/golangci/golangci-lint/v2/cmd/golangci-lint@latest
	go mod tidy

upgrade-ts:
	cd ts/logger-ts && bun update

upgrade-py:
	cd py/logger-py && uv lock --upgrade

# ---------------------------------------------------------
# Documentation Governance & CI
# ---------------------------------------------------------
doc-lint:
	@chmod +x scripts/doc-lint.sh
	./scripts/doc-lint.sh

ci: format-check lint test

clean:
	rm -rf py/logger-py/.pytest_cache py/logger-py/.ruff_cache ts/logger-ts/node_modules
