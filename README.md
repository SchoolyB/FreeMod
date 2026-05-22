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
git clone https://github.com/SchoolyB/freemod
cd freemod
make build
```

`make help` lists all available targets.

---

## Try the demo

FreeMod ships with a **FreeMod Demo Target** trainer and a small demo process so you can see cheats work without a real game.

In two terminals:

```bash
# terminal 1
make demo

# terminal 2
make dev
```

In FreeMod: select **FreeMod Demo Target**, wait a moment for it to auto-connect, then toggle **Infinite Health** — the health value in terminal 1 locks at 9999.

---

## Usage

```bash
make dev
```

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

`image` is optional — a cover-art file (PNG/JPG/WebP) in the same folder as the trainer JSON, shown in the game gallery. Vertical 2:3 art looks best; without it the game gets a lettered placeholder tile.

For dynamic memory, add an `"offsets"` array to walk a pointer chain:

```json
"base_offset": "0x01234ABC",
"offsets": ["0x58", "0x10", "0x1C"]
```

### Dev Mode tab

Use Dev Mode to find memory addresses in an unknown game:

1. Click **Refresh** and select the game process.
2. Enter the current value and click **Scan**.
3. Change the value in-game, enter the new value, click **Narrow**.
4. Repeat until one address remains — the `base_offset` is shown automatically.
5. Click the address to copy it to the Write panel to test writing a value.

Once you have a stable address or pointer chain, add it to a trainer JSON.

---

## Development

```bash
make build  # build app + demo process
make demo   # terminal 1 — kills any stale instance, starts fresh
make dev    # terminal 2 — live reload
```

In the GUI, select **FreeMod Demo Target** — it connects on the spot — and toggle **Infinite Health**; terminal 1 flips to `health = 9999`.

> `freemod-demo` prints a `Static offset` on startup. `make build` patches this into `trainers/target.json` automatically, so the demo always works after a fresh build.

---

## License

MIT
