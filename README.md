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
- [Go 1.21+](https://go.dev/dl/)
- [Node.js 18+](https://nodejs.org/)
- [Wails v2](https://wails.io/): `go install github.com/wailsapp/wails/v2/cmd/wails@latest`

---

## Install

```bash
git clone https://github.com/freemod/freemod
cd freemod
wails build
```

The built app lands at `build/bin/freemod.app`.

---

## Usage

FreeMod needs root to read and write another process's memory.

```bash
sudo open build/bin/freemod.app
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

```bash
wails dev   # live reload, opens devtools with Cmd+Option+I
```

> **Note:** `wails dev` also needs `sudo` for memory access.

---

## License

MIT
