# FreeMod

Free, open source single-player game trainer for macOS. Works like WeMod — pick a game, flip a switch, cheat. No accounts, no paywalls.

---

## What is it

FreeMod lets you toggle cheats (infinite health, infinite ammo, custom values) in single-player games by reading and writing process memory. Select a game, connect, and toggle cheats on/off with a switch.

It is **not** an anti-cheat bypass. Single-player only.

---

## Prerequisites

- macOS 12 or later (arm64 or x86_64)
- Xcode Command Line Tools: `xcode-select --install`
- [Go 1.23+](https://go.dev/dl/)
- [Node.js 18+](https://nodejs.org/)
- [Wails v2](https://wails.io/): `go install github.com/wailsapp/wails/v2/cmd/wails@latest`

---

## Install

```bash
git clone https://github.com/freemod/freemod
cd freemod
make build
```

`make build` compiles the app to `build/bin/freemod.app`, builds a demo process
at `build/bin/target` (see [Try the demo](#try-the-demo)), and code-signs the
app so it can read game memory without root. It wraps `wails build` — run
either. `make help` lists all targets.

---

## Try the demo

FreeMod ships with a **FreeMod Demo Target** trainer paired with a tiny demo
process (`build/bin/target`) — so you can see a cheat work without a real game.

In two terminals:

```bash
./build/bin/target          # terminal 1 — the process to cheat on
open build/bin/freemod.app  # terminal 2 — FreeMod
```

In FreeMod: select **FreeMod Demo Target**, wait ~2s for it to auto-connect,
then toggle **Infinite Health** — `health` in terminal 1 locks at 9999.

> Clicking Connect with no `target` process running gives `"target" is not
> running` — that's expected; start `./build/bin/target` first.

---

## Usage

```bash
open build/bin/freemod.app
```

No `sudo`: `make build` code-signs FreeMod with the `com.apple.security.cs.debugger`
entitlement — the same one `lldb` uses — so it reads and writes process memory
without root. It can only touch processes **you** own (your single-player games),
never another user's or the system's.

### Trainers tab

1. Select a game from the left panel.
2. Start the game — FreeMod auto-connects within a few seconds.
3. Toggle cheats on/off with the switches.
4. Toggle off to restore the original value.

### Adding trainers

Drop a `.json` trainer file into your trainer folder. Click the 📂 button in the app to open it in Finder.

Trainer format:

```json
{
  "game": "My Game",
  "exe": "mygame",
  "version": "1.0",
  "image": "mygame.png",
  "cheats": [
    {
      "name": "Infinite Health",
      "description": "Locks health at 9999",
      "type": "int32",
      "base_offset": "0x15c344",
      "value": 9999
    }
  ]
}
```

Supported types: `int32`, `int64`, `float32`.

`image` is optional — a cover-art file (PNG/JPG/WebP) in the same folder as the
trainer JSON, shown in the game gallery. Vertical 2:3 art looks best; without
it the game gets a lettered placeholder tile.

For dynamic memory, add an `"offsets"` array to walk a pointer chain:

```json
"base_offset": "0x01234ABC",
"offsets": ["0x58", "0x10", "0x1C"]
```

### Dev Mode tab

Use Dev Mode to find memory addresses in an unknown game:

1. Click **Refresh** and select the target process.
2. Enter the current value and click **Scan**.
3. Change the value in-game, enter the new value, click **Narrow**.
4. Repeat until one address remains.
5. Click the address to copy it to the Write panel, then write any value.

Once you have a stable address or pointer chain, add it to a trainer JSON.

---

## Development

The demo target is a separate process FreeMod connects to, so iterating uses
two terminals. Build once so `build/bin/target` exists:

```bash
make build
```

Then run, in two terminals:

```bash
./build/bin/target   # terminal 1 — leave running; prints `health` each second
make dev             # terminal 2 — live reload, devtools with Cmd+Option+I
```

In the GUI, select **FreeMod Demo Target** — it connects on the spot — and
toggle **Infinite Health**; terminal 1 flips to `health = 9999`.

> `target` prints a `Static offset` on startup — the value `target.json` uses
> as `base_offset`. If toggling doesn't move `health`, that offset no longer
> matches `trainers/target.json`; update the JSON to match.

---

## License

MIT
