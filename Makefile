# SPDX-License-Identifier: Apache-2.0
# SPDX-FileCopyrightText: 2025-Present Defense Unicorns

.DEFAULT_GOAL := build

build:
	CGO_ENABLED=0 go build -o bin/ -ldflags="-s -w" -trimpath ./cmd/maru2
	go run cmd/maru2-schema/main.go > maru2.schema.json

lint:
	golangci-lint run ./...

clean:
	rm -rf bin/

hello-world:
	echo "Hello, World!"

ARGS ?=
%:
	./bin/maru2 $* $(ARGS)

.PHONY: build lint clean hello-world
