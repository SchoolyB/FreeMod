# FreeMod build tasks. Run `make` (or `make build`) to build everything.
.DEFAULT_GOAL := build
.PHONY: build dev target run clean help

## build: compile the signed app + demo target into build/bin
build:
	wails build

## dev: run with live reload — no sudo needed, the app is signed for memory access
dev:
	wails dev

## target: build only the demo target process
target:
	go build -o build/bin/target ./cmd/target

## run: launch the built app
run:
	open build/bin/freemod.app

## clean: remove build output
clean:
	rm -rf build/bin frontend/dist

## help: list available targets
help:
	@grep -E '^## ' $(MAKEFILE_LIST) | sed 's/^## /  /'
