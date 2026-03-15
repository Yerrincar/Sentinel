SHELL := /bin/bash

APP_NAME := sentinel
CMD_PATH := ./cmd/sentinel
MAIN_FILE := ./cmd/sentinel/main.go

.PHONY: help init run build clean deps check-go check-docker check-systemd check-kubectl fmt test

help:
	@echo "Targets:"
	@echo "  make init      - Prepare project (bin dir + Go modules + dependency checks)"
	@echo "  make run       - Run Sentinel"
	@echo "  make build     - Build binary to ./bin/$(APP_NAME)"
	@echo "  make fmt       - Format Go code"
	@echo "  make test      - Run tests"
	@echo "  make clean     - Remove build artifacts"
	@echo "  make deps      - Check optional runtime dependencies"

init: check-go deps
	@mkdir -p bin
	@go mod download
	@echo "Initialization complete."

deps: check-docker check-systemd check-kubectl

check-go:
	@command -v go >/dev/null 2>&1 || { echo "Error: Go is required."; exit 1; }

check-docker:
	@if ! command -v docker >/dev/null 2>&1; then \
		echo "Warning: 'docker' not found. Docker services/actions will not work."; \
	fi

check-systemd:
	@if ! command -v systemctl >/dev/null 2>&1; then \
		echo "Warning: 'systemctl' not found. systemd actions/status checks may fail."; \
	fi

check-kubectl:
	@if ! command -v kubectl >/dev/null 2>&1; then \
		echo "Warning: 'kubectl' not found. Kubernetes workflows may fail."; \
	fi

run: check-go
	@go run $(MAIN_FILE)

build: check-go
	@mkdir -p bin
	@go build -o bin/$(APP_NAME) $(CMD_PATH)
	@echo "Built bin/$(APP_NAME)"

fmt: check-go
	@go fmt ./...

test: check-go
	@go test ./...

clean:
	@rm -rf bin
	@echo "Cleaned build artifacts."
