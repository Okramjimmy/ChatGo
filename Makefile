# ChatGo Makefile
#
# Usage:
#   make build       – compile server and migrate binaries
#   make run         – run the server locally (dev mode)
#   make migrate     – apply database migrations
#   make test        – run all tests
#   make lint        – run golangci-lint
#   make clean       – remove build artifacts
#   make install     – install binaries to /usr/local/bin

BINARY_SERVER   := chatgo-server
BINARY_MIGRATE  := chatgo-migrate
BUILD_DIR       := bin
CMD_SERVER      := ./cmd/server
CMD_MIGRATE     := ./cmd/migrate
VERSION         := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS         := -ldflags "-X main.Version=$(VERSION) -s -w"
CONFIG          ?= configs/config.yaml
MIGRATIONS_PATH ?= file://migrations

.PHONY: all build build-server build-migrate run migrate migrate-down \
        migrate-version test lint clean install vet tidy

all: build

## ── Build ────────────────────────────────────────────────────────────────────

build: build-server build-migrate

build-server:
	@echo "Building $(BINARY_SERVER)..."
	@mkdir -p $(BUILD_DIR)
	go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_SERVER) $(CMD_SERVER)

build-migrate:
	@echo "Building $(BINARY_MIGRATE)..."
	@mkdir -p $(BUILD_DIR)
	go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_MIGRATE) $(CMD_MIGRATE)

## ── Run ─────────────────────────────────────────────────────────────────────

run:
	@CHATGO_CONFIG=$(CONFIG) CHATGO_LOG_PRODUCTION=false \
	  go run $(CMD_SERVER)/main.go

## ── Database ─────────────────────────────────────────────────────────────────

migrate:
	@CHATGO_CONFIG=$(CONFIG) CHATGO_MIGRATIONS_PATH=$(MIGRATIONS_PATH) \
	  go run $(CMD_MIGRATE)/main.go up

migrate-down:
	@CHATGO_CONFIG=$(CONFIG) CHATGO_MIGRATIONS_PATH=$(MIGRATIONS_PATH) \
	  go run $(CMD_MIGRATE)/main.go down

migrate-version:
	@CHATGO_CONFIG=$(CONFIG) CHATGO_MIGRATIONS_PATH=$(MIGRATIONS_PATH) \
	  go run $(CMD_MIGRATE)/main.go version

## ── Test & Quality ───────────────────────────────────────────────────────────

test:
	go test -race -cover ./...

vet:
	go vet ./...

tidy:
	go mod tidy

lint:
	golangci-lint run ./...

## ── Install & Clean ──────────────────────────────────────────────────────────

install: build
	@echo "Installing to /usr/local/bin ..."
	install -m 0755 $(BUILD_DIR)/$(BINARY_SERVER)  /usr/local/bin/$(BINARY_SERVER)
	install -m 0755 $(BUILD_DIR)/$(BINARY_MIGRATE) /usr/local/bin/$(BINARY_MIGRATE)

clean:
	@rm -rf $(BUILD_DIR)
	@echo "Cleaned."

## ── Setup (development) ──────────────────────────────────────────────────────

setup-dirs:
	@mkdir -p /var/chatgo/files /var/log/chatgo
	@chown -R $(shell id -u):$(shell id -g) /var/chatgo /var/log/chatgo

## ── Help ─────────────────────────────────────────────────────────────────────

help:
	@echo "ChatGo Makefile targets:"
	@echo "  build          Compile server and migrate binaries"
	@echo "  run            Run server in dev mode"
	@echo "  migrate        Apply pending migrations"
	@echo "  migrate-down   Roll back all migrations"
	@echo "  test           Run all tests"
	@echo "  lint           Run golangci-lint"
	@echo "  install        Install binaries to /usr/local/bin"
	@echo "  clean          Remove build artifacts"
	@echo "  tidy           Run go mod tidy"
