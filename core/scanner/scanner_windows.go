//go:build windows

package scanner

import (
	"fmt"
	"unsafe"

	"github.com/freemod/freemod/core/memory"
	"golang.org/x/sys/windows"
)

// FindModuleBase returns the base address of the main executable module for
// the given process. It uses EnumProcessModules — the first HMODULE returned
// is always the main executable, and on Windows an HMODULE value IS the
// module's base address.
func FindModuleBase(_ memory.Memory, pid int) (uintptr, error) {
	h, err := windows.OpenProcess(
		windows.PROCESS_QUERY_INFORMATION|windows.PROCESS_VM_READ,
		false,
		uint32(pid),
	)
	if err != nil {
		return 0, fmt.Errorf("OpenProcess(%d): %w", pid, err)
	}
	defer windows.CloseHandle(h)

	var module windows.Handle
	var needed uint32
	if err := windows.EnumProcessModules(h, &module, uint32(unsafe.Sizeof(module)), &needed); err != nil {
		return 0, fmt.Errorf("EnumProcessModules(%d): %w", pid, err)
	}
	if needed == 0 || module == 0 {
		return 0, fmt.Errorf("no modules found for pid %d", pid)
	}

	// HMODULE == base address on Windows.
	return uintptr(module), nil
}
