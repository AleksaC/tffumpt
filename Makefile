SRC := $(shell find . -type f -name '*.go')
COVERAGE_FILE := coverage.out
BINARY := tffumpt
ifeq ($(OS),Windows_NT)
	BINARY := $(BINARY).exe
endif

.PHONY: help ## Show this help message
help:
	@echo "Available commands:"
	@echo ""
	@grep -E '^\.PHONY: [a-zA-Z_-]+ ## .*$$' $(MAKEFILE_LIST) | sed 's/^\.PHONY: //' | sort | awk 'BEGIN {FS = " ## "}; {printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2}'
	@echo ""

.PHONY: build ## Build the tffumpt binary
build:
	@go build -o $(BINARY) ./cmd/tffumpt/

.PHONY: test ## Run tests with coverage
$(COVERAGE_FILE) test: $(SRC)
	@go test ./... -timeout 1s -coverprofile $(COVERAGE_FILE)

.PHONY: coverage ## Generate HTML coverage report
coverage: $(COVERAGE_FILE)
	@go tool cover -html=$(COVERAGE_FILE)

.PHONY: lint ## Run golangci-lint
lint:
	@pre-commit run golangci-lint-full --all-files

.PHONY: pre-commit ## Run all pre-commit hooks
pre-commit:
	@pre-commit run --all-files

.PHONY: check ## Run pre-commit hooks and tests
check: pre-commit test

.PHONY: release ## Create a new release
release:
	@goreleaser release --clean

.PHONY: snapshot ## Create a snapshot release
snapshot:
	@goreleaser release --clean --snapshot

.PHONY: clean ## Clean build artifacts
clean:
	@rm -f $(BINARY)
	@rm -rf dist
	@rm -f $(COVERAGE_FILE)
