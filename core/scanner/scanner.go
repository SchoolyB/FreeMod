package scanner

import (
	"encoding/binary"
	"fmt"

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
