##r 🔭 otel-explorer-go-docs 🔭
SHELL := /bin/bash
MAKEFLAGS += --silent

BINARY_NAME_BASE=otel-explorer-go-docs

all: help

.PHONY: help
help: ## ❓ Makefile commands
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  \033[36m%-12s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST)

.PHONY: build
build:
	go build -o $(BINARY_NAME_BASE) ./cmd/scanner

.PHONY: clean
clean: ## 🧹 Cleanup build artifacts
	go clean && rm -rf .repo $(BINARY_NAME_BASE) coverage.* insturmentation-list.yaml

.PHONY: dev
dev: ## 🚀 Start development server
	go run ./cmd/scanner

.PHONY: lint
lint: ## 🧹 Run linter checks
	golangci-lint run

.PHONY: fmt
fmt: ## ✨ Format code
	go fmt ./...

.PHONY: tidy
tidy: ## 📚 Tidy modules
	go mod tidy

.PHONY: docs
docs: ## 📖 Godocs
	go doc -http

.PHONY: test
test: ## 🧪 Run all tests
	go test -test.v -race -covermode=atomic -coverprofile=coverage.out ./... && \
	go tool cover -html=coverage.out -o coverage.html && \
	echo "Coverage report saved to coverage.html" && \
	rm -f coverage.out

.PHONY: test-perf
test-perf: ## ⚡ Run benchmark tests
	go test -test.v -benchmem -bench=. -coverprofile=coverage-bench.out ./... && \
	go tool cover -html=coverage-bench.out -o coverage-bench.html && \
	echo "Coverage report saved to coverage-bench.html" && \
	rm -f coverage-bench.out

.PHONY: vuln
vuln: ## 🛡️  Scan for vulnerabilities
	go install golang.org/x/vuln/cmd/govulncheck@latest
	govulncheck ./...

.PHONY: pre-commit
pre-commit: fmt tidy lint test ## ✅ Run all checks

# pass through CLI flags to ./cmd/
%:
	@:
