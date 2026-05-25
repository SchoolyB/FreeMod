//go:build darwin

package scanner

import (
	"encoding/binary"
	"fmt"

	"github.com/freemod/freemod/core/memory"
)

// FindModuleBase scans the process's readable regions for a Mach-O 64-bit
// header (magic 0xFEEDFACF) at an address >= 0x100000000, which is the
// default load address of the main executable on macOS arm64/amd64.
// Returns the actual (ASLR-adjusted) base address of the main module.
func FindModuleBase(mem memory.Memory, pid int) (uintptr, error) {
	const (
		machoMagic64 = uint32(0xFEEDFACF)
		defaultBase  = uintptr(0x100000000)
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
