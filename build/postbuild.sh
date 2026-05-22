#!/bin/sh
# Wails post-build hook — runs after `wails build` and every `wails dev` rebuild.
#
# It does two things:
#   1. Builds the bundled demo target process (build/bin/target).
#   2. Code-signs FreeMod with the debugger entitlement so it can read game
#      memory without root. No sudo means nothing in the source tree gets
#      created root-owned (which is what used to poison frontend/dist).
#
# $1 is the compiled FreeMod binary path, passed by Wails as ${bin}.
set -e

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
ENT="$ROOT/build/entitlements.plist"

# 1. Build the demo target.
go build -C "$ROOT" -o build/bin/target ./cmd/target

# 2. Sign FreeMod. ${bin} may be the app bundle, a binary inside it, or a
#    bare dev binary — sign the .app bundle when there is one.
[ -n "$1" ] || { echo "post-build: no binary path supplied" >&2; exit 1; }
case "$1" in
	*.app)   target="$1" ;;
	*.app/*) target="${1%.app/*}.app" ;;
	*)       target="$1" ;;
esac
codesign --sign - --entitlements "$ENT" --force "$target"

echo "post-build: built demo target; signed $target with debugger entitlement"
