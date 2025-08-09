# SPDX-License-Identifier: Apache-2.0
# SPDX-FileCopyrightText: 2025-Present Defense Unicorns

.DEFAULT_GOAL := all

all: maru2 maru2-publish

maru2:
	CGO_ENABLED=0 go build -o bin/ -ldflags="-s -w" -trimpath ./cmd/maru2
	$(MAKE) schemas

maru2-publish:
	CGO_ENABLED=0 go build -o bin/ -ldflags="-s -w" -trimpath ./cmd/maru2-publish

schemas:
	go run cmd/maru2-schema/main.go > schema/v0/schema.json

lint:
	golangci-lint run ./...

clean:
	rm -rf bin/

hello-world:
	echo "Hello, World!"

ARGS ?=
%:
	./bin/maru2 $* $(ARGS)

.PHONY: all maru2 maru2-publish schemas lint clean hello-world
