#!/usr/bin/env python3
"""
ct2freemod — Convert Cheat Engine .ct files to FreeMod trainer JSON.

Usage:
    python3 ct2freemod.py input.ct
    python3 ct2freemod.py input.ct -o output.json
    python3 ct2freemod.py input.ct --game "My Game" --exe "MyGame" --version "1.0"

Output is written to <input>.json by default.
"""

import argparse
import json
import os
import re
import sys
import xml.etree.ElementTree as ET


# ── CT type → FreeMod type mapping ───────────────────────────────────────────

CT_TYPE_MAP = {
    "4 bytes":    "int32",
    "2 bytes":    "int32",   # upcast — FreeMod uses int32 minimum
    "1 byte":     "int32",
    "byte":       "int32",
    "float":      "float32",
    "double":     "float64",
    "8 bytes":    "int64",
    "word":       "int32",
    "dword":      "int32",
    "qword":      "int64",
    "int32":      "int32",
    "int64":      "int64",
    "float32":    "float32",
    "float64":    "float64",
}

def map_type(ct_type: str) -> str:
    if not ct_type:
        return "int32"
    return CT_TYPE_MAP.get(ct_type.lower().strip(), "int32")


# ── Address parsing ───────────────────────────────────────────────────────────

def parse_address(address: str):
    """
    Parse a CT address string into (base_offset, offsets[]).

    CT addresses look like:
      "MyGame.exe+1A2B3C"               → base_offset=0x1a2b3c, offsets=[]
      "MyGame.exe+1A2B3C,10,4C"         → base_offset=0x1a2b3c, offsets=[0x10,0x4c]
      "[MyGame.exe+1A2B3C]+10"          → base_offset=0x1a2b3c, offsets=[0x10]
      "[[MyGame.exe+1A2B3C]+10]+4C"     → base_offset=0x1a2b3c, offsets=[0x10,0x4c]
      "1A2B3C"                          → base_offset=0x1a2b3c, offsets=[]
    """
    if not address:
        return None, []

    address = address.strip()

    # Strip all brackets — they just indicate pointer dereference levels
    address_clean = address.replace("[", "").replace("]", "")

    # Split on "+" to separate module+offset from pointer offsets
    # Pattern: [ModuleName+BaseOffset, offset1, offset2, ...]
    # After stripping brackets, commas separate offsets in some formats
    # "MyGame.exe+1A2B3C,10,4C"
    parts = re.split(r'[,+]', address_clean)
    parts = [p.strip() for p in parts if p.strip()]

    base_offset = None
    offsets = []

    for i, part in enumerate(parts):
        # Skip module name (contains a dot or "exe")
        if "." in part or part.lower().endswith("exe"):
            continue
        try:
            val = int(part, 16)
            if base_offset is None:
                base_offset = val
            else:
                offsets.append(val)
        except ValueError:
            continue

    if base_offset is None:
        return None, []

    return base_offset, offsets


def fmt_hex(val: int) -> str:
    return f"0x{val:x}"


# ── CT XML parsing ────────────────────────────────────────────────────────────

def parse_cheat_table(root: ET.Element) -> list:
    """
    Walk the CheatTable XML and extract all enabled/named entries
    that have a usable address and type.
    Returns a list of dicts ready for FreeMod cheat format.
    """
    cheats = []

    # CT entries live in <CheatEntries><CheatEntry>...</CheatEntry></CheatEntries>
    # They can be nested (groups contain sub-entries)
    entries = root.findall(".//CheatEntry")

    for entry in entries:
        description_el = entry.find("Description")
        address_el     = entry.find("Address")
        vartype_el     = entry.find("VariableType")
        value_el       = entry.find("Value")

        if address_el is None or address_el.text is None:
            continue  # skip groups/folders with no address

        description = ""
        if description_el is not None and description_el.text:
            # CT descriptions are often quoted
            description = description_el.text.strip().strip('"')

        if not description:
            continue  # skip unnamed entries

        address_str = address_el.text.strip()
        base_offset, offsets = parse_address(address_str)

        if base_offset is None:
            print(f"  [skip] Could not parse address: {address_str!r} ({description!r})", file=sys.stderr)
            continue

        ct_type = ""
        if vartype_el is not None and vartype_el.text:
            ct_type = vartype_el.text.strip()

        fm_type = map_type(ct_type)

        # Try to extract a freeze value
        freeze_val = 0.0
        if value_el is not None and value_el.text:
            try:
                freeze_val = float(value_el.text.strip())
            except ValueError:
                pass

        cheat = {
            "name":        description,
            "description": f"Imported from Cheat Engine — original type: {ct_type or 'unknown'}",
            "type":        fm_type,
            "behavior":    "freeze",
            "base_offset": fmt_hex(base_offset),
            "value":       freeze_val,
        }

        if offsets:
            cheat["offsets"] = [fmt_hex(o) for o in offsets]

        cheats.append(cheat)

    return cheats


def detect_exe(root: ET.Element, fallback: str) -> str:
    """Try to detect the target process name from the CT file."""
    # Some CT files store it in <CheatTable><TargetApp>
    target = root.find("TargetApp")
    if target is not None and target.text:
        exe = target.text.strip()
        # Strip .exe suffix (FreeMod uses bare name on macOS)
        exe = re.sub(r'\.exe$', '', exe, flags=re.IGNORECASE)
        return exe
    return fallback


# ── Main ──────────────────────────────────────────────────────────────────────

def main():
    parser = argparse.ArgumentParser(
        description="Convert a Cheat Engine .ct file to a FreeMod trainer JSON."
    )
    parser.add_argument("input", help="Path to the .ct file")
    parser.add_argument("-o", "--output", help="Output JSON path (default: <input>.json)")
    parser.add_argument("--game",    default="", help="Game display name")
    parser.add_argument("--exe",     default="", help="Process name (without .exe)")
    parser.add_argument("--version", default="1.0", help="Game version string")
    args = parser.parse_args()

    if not os.path.exists(args.input):
        print(f"Error: file not found: {args.input}", file=sys.stderr)
        sys.exit(1)

    # CT files are XML — parse them
    try:
        tree = ET.parse(args.input)
        root = tree.getroot()
    except ET.ParseError as e:
        print(f"Error: failed to parse CT file as XML: {e}", file=sys.stderr)
        sys.exit(1)

    # Detect exe name
    exe = args.exe or detect_exe(root, os.path.splitext(os.path.basename(args.input))[0])
    game = args.game or exe

    print(f"Parsing: {args.input}")
    cheats = parse_cheat_table(root)

    if not cheats:
        print("Warning: no usable cheat entries found.", file=sys.stderr)

    trainer = {
        "game":    game,
        "exe":     exe,
        "version": args.version,
        "cheats":  cheats,
    }

    out_path = args.output or os.path.splitext(args.input)[0] + ".json"

    with open(out_path, "w") as f:
        json.dump(trainer, f, indent=2)

    print(f"Done: {len(cheats)} cheats → {out_path}")
    if cheats:
        print("\nEntries:")
        for c in cheats:
            offsets = f" + offsets {c['offsets']}" if c.get("offsets") else ""
            print(f"  [{c['type']:8}] {c['base_offset']}{offsets}  —  {c['name']}")


if __name__ == "__main__":
    main()
