.PHONY: build test lint clean help

MODULE  := github.com/ckandag/gcp-hcp-cli
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT  := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
DATE    := $(shell date -u '+%Y-%m-%dT%H:%M:%SZ')
LDFLAGS := -X $(MODULE)/pkg/cli.version=$(VERSION) \
	-X $(MODULE)/pkg/cli.commit=$(COMMIT) \
	-X $(MODULE)/pkg/cli.date=$(DATE)

help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*##' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*## "}; {printf "  %-14s %s\n", $$1, $$2}'

build: ## Build the gcphcp binary
	@mkdir -p bin
	go build -ldflags "$(LDFLAGS)" -o bin/gcphcp ./cmd/gcphcp
	@echo "Built bin/gcphcp"

test: ## Run unit tests
	go test -race ./...

lint: ## Run go vet
	go vet ./...

clean: ## Remove build artifacts
	rm -rf bin/
