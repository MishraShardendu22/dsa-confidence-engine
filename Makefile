.PHONY: all build run test test-race test-coverage clean templ help fmt fmt-check vet tidy-check pre-commit

# Binary name
BINARY_NAME=server

# Toolchain definitions
GO=go
GOPATH=$(shell go env GOPATH)
TEMPL_BIN=$(GOPATH)/bin/templ

all: build

fmt: ## Automatically format all Go files using gofmt
	gofmt -w .

fmt-check: ## Verify all Go files are gofmt-formatted (mirrors CI go-lint)
	@UNFORMATTED=$$(gofmt -l .); \
	if [ -n "$$UNFORMATTED" ]; then \
		echo "The following files are not gofmt-formatted:"; \
		echo "$$UNFORMATTED"; \
		echo "Run 'make fmt' or 'gofmt -w .' locally to fix."; \
		exit 1; \
	fi; \
	echo "All Go files are correctly formatted."

vet: ## Run go vet across all packages
	$(GO) vet ./...

tidy-check: ## Verify go.mod and go.sum are tidy (mirrors CI go-build check)
	@$(GO) mod tidy; \
	if ! git diff --exit-code go.mod go.sum; then \
		echo "go.mod/go.sum are not tidy. Run 'go mod tidy' locally."; \
		exit 1; \
	fi; \
	echo "go.mod and go.sum are tidy."

pre-commit: fmt-check vet tidy-check templ test-race build ## Full local CI mirror gate; must pass before committing or pushing
	@echo "All pre-commit and CI verification checks passed successfully."

templ: ## Generate Go code from Templ templates
	@if command -v templ >/dev/null 2>&1; then \
		echo "Generating Templ components..."; \
		templ generate; \
	elif [ -x "$(TEMPL_BIN)" ]; then \
		echo "Generating Templ components via $(TEMPL_BIN)..."; \
		"$(TEMPL_BIN)" generate; \
	else \
		echo "Generating Templ components via go run..."; \
		$(GO) run github.com/a-h/templ/cmd/templ@latest generate; \
	fi

build: templ ## Compile the server executable
	$(GO) build -o $(BINARY_NAME) ./cmd/server

run: templ ## Generate templates and run the server application
	$(GO) run ./cmd/server

test: templ ## Run unit and integration tests
	$(GO) test -v ./...

test-race: templ ## Run tests with data race detector
	$(GO) test -race ./...

test-coverage: templ ## Run tests and generate HTML coverage report
	$(GO) test -coverprofile=coverage.out ./...
	$(GO) tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated at coverage.html"

clean: ## Remove build artifacts and temporary databases
	rm -f $(BINARY_NAME) coverage.out coverage.html
	rm -f *.db *.db-journal data/*.db data/*.db-journal
	rm -rf scripts/__pycache__ **/__pycache__ *.pyc
	@echo "Clean complete."

help: ## Display available Makefile targets
	@echo "Usage: make [target]"
	@echo ""
	@echo "Targets:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-16s\033[0m %s\n", $$1, $$2}'
