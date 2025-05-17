# SPDX-License-Identifier: Apache-2.0
# SPDX-FileCopyrightText: 2025-Present Defense Unicorns

.DEFAULT_GOAL := build

build:
	CGO_ENABLED=0 go build -o bin/ -ldflags="-s -w" -trimpath ./cmd/maru2
	go run gen/main.go

lint:
	golangci-lint run ./...

clean:
	rm -rf bin/

alias:
	@echo "alias maru2='$(PWD)/bin/maru2'" >>  ~/.config/fish/config.fish
	@echo "MARU2_COMPLETION=true maru2 completion fish | source" >> ~/.config/fish/config.fish

hello-world:
	echo "Hello, World!"

.PHONY: build clean alias hello-world lint
