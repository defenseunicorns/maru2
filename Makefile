# SPDX-License-Identifier: Apache-2.0
# SPDX-FileCopyrightText: 2025-Present Defense Unicorns

.DEFAULT_GOAL := build

build: build-maru2 build-maru2-publish

build-maru2:
	CGO_ENABLED=0 go build -o bin/ -ldflags="-s -w" -trimpath ./cmd/maru2
	go run cmd/maru2-schema/main.go > maru2.schema.json

build-maru2-publish:
	CGO_ENABLED=0 go build -o bin/ -ldflags="-s -w" -trimpath ./cmd/maru2-publish

lint:
	golangci-lint run ./...

clean:
	rm -rf bin/

hello-world:
	echo "Hello, World!"

ARGS ?=
%:
	./bin/maru2 $* $(ARGS)

.PHONY: build build-publish lint clean hello-world
