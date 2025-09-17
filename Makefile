# SPDX-License-Identifier: Apache-2.0
# SPDX-FileCopyrightText: 2025-Present Defense Unicorns

.DEFAULT_GOAL := all

export CGO_ENABLED=0

all: maru2 maru2-publish maru2-mcp ## Build all binaries

SCHEMA_DEPS := schema.go schema/*.go builtins/*.go

maru2: maru2.schema.json schema/v0/schema.json schema/v1/schema.json ## Build maru2 binary and generate schemas
	go build -o bin/ -ldflags="-s -w" -trimpath ./cmd/maru2

maru2.schema.json: $(SCHEMA_DEPS) schema/v0/*.go schema/v1/*.go
	go run cmd/maru2-schema/main.go > maru2.schema.json

schema/v0/schema.json: $(SCHEMA_DEPS) schema/v0/*.go
	go run cmd/maru2-schema/main.go v0 > schema/v0/schema.json

schema/v1/schema.json: $(SCHEMA_DEPS) schema/v1/*.go
	go run cmd/maru2-schema/main.go v1 > schema/v1/schema.json

maru2-publish: ## Build maru2-publish binary
	go build -o bin/ -ldflags="-s -w" -trimpath ./cmd/maru2-publish

maru2-mcp: ## Build maru2-mcp binary
	go build -o bin/ -ldflags="-s -w" -trimpath ./cmd/maru2-mcp

lint: ## Run linters
	golangci-lint run ./...

lint-fix: ## Run linters with auto-fix
	golangci-lint run --fix ./...

clean: ## Remove build artifacts
	rm -rf bin/ dist/ maru2.schema.json schema/v0/schema.json schema/v1/schema.json

install: ## Installs local builds
	go install -v ./cmd/maru2*

hello-world:
	echo "Hello, World!"

ARGS ?=
%:
	./bin/maru2 $* $(ARGS)

help: ## Show this help message
	@echo 'Usage: make [target]'
	@echo ''
	@echo 'Available targets:'
	@awk 'BEGIN {FS = ":.*## "} /^[a-zA-Z0-9_-]+:.*## / {printf "  %-15s %s\n", $$1, $$2}' $(MAKEFILE_LIST)
	@echo ''
	@echo 'Special targets:'
	@echo '  <task-name>     Run any maru2 task via: make <task-name> [ARGS="--flag"]'

.PHONY: all maru2 maru2-publish lint lint-fix clean install hello-world
