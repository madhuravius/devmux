.DEFAULT_GOAL := help

CLI_BIN := build/cli
SVC_BIN := build/svc

CLI_SRC := ./cmd/cli
SVC_SRC := ./cmd/svc

.PHONY: help deps build test clean start

help: ## Show this help message
	@echo "LottaLogs Make Targets"
	@echo "Usage: make [target]"
	@echo "Common cases to start quickly will be: \"make deps\" and \"make start\""
	@echo ""
	@echo "Targets:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  %-30s %s\n", $$1, $$2}'

deps: ## Download go module dependencies
	go mod tidy

build: ## Build cli and svc binaries
	@echo "Building cli..."
	go build -o $(CLI_BIN) $(CLI_SRC)
	@echo "Building svc..."
	go build -o $(SVC_BIN) $(SVC_SRC)

test: ## Run all unit tests
	go test ./... -v

clean: ## Remove generated binaries and test cache
	@echo "Cleaning up..."
	rm -rf bin
	go clean -testcache

