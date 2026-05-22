# FreeMod build tasks. Run `make` (or `make build`) to build everything.
.DEFAULT_GOAL := build
.PHONY: build dev target run clean help

## build: compile the signed app + demo target into build/bin, auto-patch target.json
build:
	@test -f trainers/target.json || cp trainers/target.json.example trainers/target.json
	go build -o build/bin/freemod-demo ./cmd/target
	@addr=$$(go tool nm build/bin/freemod-demo | awk '/main\.health$$/{print $$1}'); \
	offset=$$(printf "0x%x" $$(( 0x$$addr - 0x100000000 ))); \
	sed -i '' "s|\"base_offset\": \"0x[0-9a-fA-F]*\"|\"base_offset\": \"$$offset\"|g" trainers/target.json; \
	echo "Patched trainers/target.json base_offset → $$offset"
	wails build

## dev: run with live reload — no sudo needed, the app is signed for memory access
dev:
	wails dev

## demo: build and run the demo process (kills any existing instance first)
demo:
	@pkill -x freemod-demo 2>/dev/null || true
	go build -o build/bin/freemod-demo ./cmd/target
	build/bin/freemod-demo

## run: launch the built app
run:
	open build/bin/freemod.app

## clean: remove build output
clean:
	rm -rf build/bin frontend/dist

## help: list available targets
help:
	@grep -E '^## ' $(MAKEFILE_LIST) | sed 's/^## /  /'
