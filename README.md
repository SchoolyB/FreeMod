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
wails build
```

The built app lands at `build/bin/freemod.app`. `wails build` also builds a
demo process at `build/bin/target` (see [Try the demo](#try-the-demo)).

---

## Try the demo

FreeMod ships with a **FreeMod Demo Target** trainer paired with a tiny demo
process (`build/bin/target`) — so you can see a cheat work without a real game.

In two terminals:

```bash
./build/bin/target               # terminal 1 — the process to cheat on
sudo open build/bin/freemod.app  # terminal 2 — FreeMod (needs root)
```

In FreeMod: select **FreeMod Demo Target**, wait ~2s for it to auto-connect,
then toggle **Infinite Health** — `health` in terminal 1 locks at 9999.

> Clicking Connect with no `target` process running gives `"target" is not
> running` — that's expected; start `./build/bin/target` first.

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

Developing against the demo target uses **two terminals**: one runs the demo
process you cheat on, the other runs FreeMod with live reload.

**Terminal 1 — the demo target.** `wails dev` does **not** run build hooks, so
build the demo process yourself once, then run it (no `sudo` — it's the target,
not FreeMod):

```bash
go build -o build/bin/target ./cmd/target
./build/bin/target
```

Leave it running; it prints `health` once a second.

**Terminal 2 — FreeMod.** Run `wails dev` with `sudo` — reading another
process's memory needs root, and without it Connect fails with
`task_for_pid … kern_return 5`:

```bash
sudo wails dev   # live reload; opens devtools with Cmd+Option+I
```

With both running, select **FreeMod Demo Target** in the GUI — it auto-connects
within ~2s — and toggle **Infinite Health**; Terminal 1 flips to `health = 9999`.

> **Why two terminals / why sudo:** the demo target is a separate program —
> FreeMod connects *to* it. The target only inspects its *own* memory, so it
> needs no privileges; FreeMod reads *another* process, which is root-only on
> macOS. `wails dev` only launches FreeMod, never the target.

On startup `target` prints a `Static offset` — the value `target.json` puts in
`base_offset`.

> If toggling **Infinite Health** doesn't move `health`, the printed `Static
> offset` no longer matches `base_offset` in `trainers/target.json` (it can
> shift when the demo is rebuilt with a different Go toolchain) — update the
> JSON to match.

---

## License

MIT
