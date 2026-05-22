package scanner

import (
	"encoding/binary"
	"fmt"
	"math"

	"github.com/freemod/freemod/core/memory"
)

// ScanForInt scans all readable memory regions of the process for the given
// int32 value and returns the addresses where it is found.
func ScanForInt(mem memory.Memory, pid int, value int32) ([]uintptr, error) {
	regions, err := mem.ReadableRegions(pid)
	if err != nil {
		return nil, fmt.Errorf("enumerating regions: %w", err)
	}

	target := make([]byte, 4)
	binary.LittleEndian.PutUint32(target, uint32(value))

	var matches []uintptr

	for _, region := range regions {
		if region.Size < 4 {
			continue
		}

		data, err := mem.ReadBytes(pid, region.Start, int(region.Size))
		if err != nil {
			// Silently skip unreadable regions (common for guard pages, etc.)
			continue
		}

		for i := 0; i <= len(data)-4; i++ {
			if data[i] == target[0] &&
				data[i+1] == target[1] &&
				data[i+2] == target[2] &&
				data[i+3] == target[3] {
				matches = append(matches, region.Start+uintptr(i))
			}
		}
	}

	return matches, nil
}

// NarrowScan filters a previous address list to only those whose current
// int32 value matches newValue.
func NarrowScan(mem memory.Memory, pid int, addrs []uintptr, newValue int32) ([]uintptr, error) {
	var matches []uintptr
	for _, addr := range addrs {
		val, err := mem.ReadInt(pid, addr)
		if err != nil {
			// Address may have become invalid; drop it.
			continue
		}
		if val == newValue {
			matches = append(matches, addr)
		}
	}
	return matches, nil
}

// ScanForFloat32 scans all readable memory regions for the given float32 value.
func ScanForFloat32(mem memory.Memory, pid int, value float32) ([]uintptr, error) {
	regions, err := mem.ReadableRegions(pid)
	if err != nil {
		return nil, fmt.Errorf("enumerating regions: %w", err)
	}

	target := make([]byte, 4)
	binary.LittleEndian.PutUint32(target, math.Float32bits(value))

	var matches []uintptr
	for _, region := range regions {
		if region.Size < 4 {
			continue
		}
		data, err := mem.ReadBytes(pid, region.Start, int(region.Size))
		if err != nil {
			continue
		}
		for i := 0; i <= len(data)-4; i++ {
			if data[i] == target[0] && data[i+1] == target[1] &&
				data[i+2] == target[2] && data[i+3] == target[3] {
				matches = append(matches, region.Start+uintptr(i))
			}
		}
	}
	return matches, nil
}

// NarrowScanFloat32 filters a previous address list to those whose current
// float32 value (by exact bit pattern) matches newValue.
func NarrowScanFloat32(mem memory.Memory, pid int, addrs []uintptr, newValue float32) ([]uintptr, error) {
	bits := math.Float32bits(newValue)
	var matches []uintptr
	for _, addr := range addrs {
		data, err := mem.ReadBytes(pid, addr, 4)
		if err != nil {
			continue
		}
		if binary.LittleEndian.Uint32(data) == bits {
			matches = append(matches, addr)
		}
	}
	return matches, nil
}

// ReadFloat32AtAddrs reads the current float32 value at each address, silently
// dropping unreadable ones. Returns parallel slices of surviving addresses and values.
func ReadFloat32AtAddrs(mem memory.Memory, pid int, addrs []uintptr) ([]uintptr, []float32) {
	out := make([]uintptr, 0, len(addrs))
	vals := make([]float32, 0, len(addrs))
	for _, addr := range addrs {
		data, err := mem.ReadBytes(pid, addr, 4)
		if err != nil {
			continue
		}
		out = append(out, addr)
		vals = append(vals, math.Float32frombits(binary.LittleEndian.Uint32(data)))
	}
	return out, vals
}

// NarrowFloat32ByMode filters a previous float32 address list by comparing the
// current value against the previously recorded value.
// mode: "increased", "decreased", "changed", or "unchanged".
func NarrowFloat32ByMode(mem memory.Memory, pid int, addrs []uintptr, prevVals []float32, mode string) ([]uintptr, []float32, error) {
	if len(addrs) != len(prevVals) {
		return nil, nil, fmt.Errorf("address/value slice length mismatch")
	}
	var survivors []uintptr
	var newVals []float32
	for i, addr := range addrs {
		data, err := mem.ReadBytes(pid, addr, 4)
		if err != nil {
			continue
		}
		cur := math.Float32frombits(binary.LittleEndian.Uint32(data))
		var keep bool
		switch mode {
		case "increased":
			keep = cur > prevVals[i]
		case "decreased":
			keep = cur < prevVals[i]
		case "changed":
			keep = cur != prevVals[i]
		case "unchanged":
			keep = cur == prevVals[i]
		default:
			return nil, nil, fmt.Errorf("unknown scan mode %q", mode)
		}
		if keep {
			survivors = append(survivors, addr)
			newVals = append(newVals, cur)
		}
	}
	return survivors, newVals, nil
}

// ReadValuesAtAddrs reads the current int32 value at each address, silently
// dropping any that have become unreadable. Returns parallel slices of the
// surviving addresses and their values.
func ReadValuesAtAddrs(mem memory.Memory, pid int, addrs []uintptr) ([]uintptr, []int32) {
	out := make([]uintptr, 0, len(addrs))
	vals := make([]int32, 0, len(addrs))
	for _, addr := range addrs {
		v, err := mem.ReadInt(pid, addr)
		if err != nil {
			continue
		}
		out = append(out, addr)
		vals = append(vals, v)
	}
	return out, vals
}

// NarrowByMode filters a previous address list by comparing each address's
// current int32 value against the previously recorded value. mode must be one
// of: "increased", "decreased", "changed", "unchanged".
// Returns the surviving addresses and their current values.
func NarrowByMode(mem memory.Memory, pid int, addrs []uintptr, prevVals []int32, mode string) ([]uintptr, []int32, error) {
	if len(addrs) != len(prevVals) {
		return nil, nil, fmt.Errorf("address/value slice length mismatch")
	}
	var survivors []uintptr
	var newVals []int32
	for i, addr := range addrs {
		cur, err := mem.ReadInt(pid, addr)
		if err != nil {
			continue
		}
		var keep bool
		switch mode {
		case "increased":
			keep = cur > prevVals[i]
		case "decreased":
			keep = cur < prevVals[i]
		case "changed":
			keep = cur != prevVals[i]
		case "unchanged":
			keep = cur == prevVals[i]
		default:
			return nil, nil, fmt.Errorf("unknown scan mode %q", mode)
		}
		if keep {
			survivors = append(survivors, addr)
			newVals = append(newVals, cur)
		}
	}
	return survivors, newVals, nil
}

// FindModuleBase scans the process's readable regions for a Mach-O 64-bit
// header (magic 0xFEEDFACF) at an address >= 0x100000000, which is the
// default load address of the main executable on macOS arm64/amd64.
// Returns the actual (ASLR-adjusted) base address of the main module.
func FindModuleBase(mem memory.Memory, pid int) (uintptr, error) {
	const (
		machoMagic64  = uint32(0xFEEDFACF)
		defaultBase   = uintptr(0x100000000)
	)

	regions, err := mem.ReadableRegions(pid)
	if err != nil {
		return 0, fmt.Errorf("enumerating regions: %w", err)
	}

	for _, r := range regions {
		if r.Start < defaultBase || r.Size < 4 {
			continue
		}
		data, err := mem.ReadBytes(pid, r.Start, 4)
		if err != nil {
			continue
		}
		if binary.LittleEndian.Uint32(data) == machoMagic64 {
			return r.Start, nil
		}
	}
	return 0, fmt.Errorf("main module base not found for pid %d", pid)
}

// ResolvePointer walks a pointer chain starting at base, applying each offset
// in turn: it reads the pointer at (current + offset) and follows it.
// Returns the final resolved address.
func ResolvePointer(mem memory.Memory, pid int, base uintptr, offsets []uintptr) (uintptr, error) {
	addr := base
	for i, offset := range offsets {
		addr += offset
		val, err := mem.ReadInt64(pid, addr)
		if err != nil {
			return 0, fmt.Errorf("resolving pointer at step %d (0x%x): %w", i, addr, err)
		}
		addr = uintptr(val)
	}
	return addr, nil
}
