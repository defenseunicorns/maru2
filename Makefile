# SPDX-License-Identifier: Apache-2.0
# SPDX-FileCopyrightText: 2025-Present Defense Unicorns

.DEFAULT_GOAL := all

all: maru2 maru2-publish

maru2:
	CGO_ENABLED=0 go build -o bin/ -ldflags="-s -w" -trimpath ./cmd/maru2
	go run cmd/maru2-schema/main.go v0 > schema/v0/schema.json
	go run cmd/maru2-schema/main.go > maru2.schema.json

maru2-publish:
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

.PHONY: all maru2 maru2-publish lint clean hello-world
