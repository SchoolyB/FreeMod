//go:build windows

package memory

import (
	"encoding/binary"
	"fmt"
	"math"
	"unsafe"

	"golang.org/x/sys/windows"
)

// WindowsMemory implements Memory using Windows process memory APIs.
// No CGO required — all calls go through golang.org/x/sys/windows.
type WindowsMemory struct{}

func New() Memory {
	return &WindowsMemory{}
}

// openProcess opens a handle to the target process with the access rights
// needed for memory read/write operations.
func openProcess(pid int) (windows.Handle, error) {
	h, err := windows.OpenProcess(
		windows.PROCESS_VM_READ|windows.PROCESS_VM_WRITE|windows.PROCESS_VM_OPERATION,
		false,
		uint32(pid),
	)
	if err != nil {
		return 0, fmt.Errorf("OpenProcess(%d): %w", pid, err)
	}
	return h, nil
}

func (m *WindowsMemory) ReadBytes(pid int, addr uintptr, length int) ([]byte, error) {
	h, err := openProcess(pid)
	if err != nil {
		return nil, err
	}
	defer windows.CloseHandle(h)

	buf := make([]byte, length)
	var nRead uintptr
	err = windows.ReadProcessMemory(h, addr, &buf[0], uintptr(length), &nRead)
	if err != nil {
		return nil, fmt.Errorf("ReadProcessMemory at 0x%x: %w", addr, err)
	}
	return buf[:nRead], nil
}

func (m *WindowsMemory) ReadInt(pid int, addr uintptr) (int32, error) {
	b, err := m.ReadBytes(pid, addr, 4)
	if err != nil {
		return 0, err
	}
	return int32(binary.LittleEndian.Uint32(b)), nil
}

func (m *WindowsMemory) ReadInt64(pid int, addr uintptr) (int64, error) {
	b, err := m.ReadBytes(pid, addr, 8)
	if err != nil {
		return 0, err
	}
	return int64(binary.LittleEndian.Uint64(b)), nil
}

func (m *WindowsMemory) ReadFloat(pid int, addr uintptr) (float32, error) {
	b, err := m.ReadBytes(pid, addr, 4)
	if err != nil {
		return 0, err
	}
	return math.Float32frombits(binary.LittleEndian.Uint32(b)), nil
}

func (m *WindowsMemory) WriteBytes(pid int, addr uintptr, data []byte) error {
	h, err := openProcess(pid)
	if err != nil {
		return err
	}
	defer windows.CloseHandle(h)

	var nWritten uintptr
	err = windows.WriteProcessMemory(h, addr, &data[0], uintptr(len(data)), &nWritten)
	if err != nil {
		// Page may be read-only (e.g. PAGE_EXECUTE_READ). Try making it writable.
		var oldProtect uint32
		if vpErr := windows.VirtualProtectEx(h, addr, uintptr(len(data)), windows.PAGE_EXECUTE_READWRITE, &oldProtect); vpErr == nil {
			err = windows.WriteProcessMemory(h, addr, &data[0], uintptr(len(data)), &nWritten)
			// Restore original protection regardless of write outcome.
			windows.VirtualProtectEx(h, addr, uintptr(len(data)), oldProtect, &oldProtect) //nolint:errcheck
		}
	}
	if err != nil {
		return fmt.Errorf("WriteProcessMemory at 0x%x: %w", addr, err)
	}
	return nil
}

func (m *WindowsMemory) WriteInt(pid int, addr uintptr, val int32) error {
	buf := make([]byte, 4)
	binary.LittleEndian.PutUint32(buf, uint32(val))
	return m.WriteBytes(pid, addr, buf)
}

func (m *WindowsMemory) WriteInt64(pid int, addr uintptr, val int64) error {
	buf := make([]byte, 8)
	binary.LittleEndian.PutUint64(buf, uint64(val))
	return m.WriteBytes(pid, addr, buf)
}

func (m *WindowsMemory) WriteFloat(pid int, addr uintptr, val float32) error {
	buf := make([]byte, 4)
	binary.LittleEndian.PutUint32(buf, math.Float32bits(val))
	return m.WriteBytes(pid, addr, buf)
}

// ReadableRegions enumerates committed, readable memory regions in the process
// using VirtualQueryEx.
func (m *WindowsMemory) ReadableRegions(pid int) ([]MemRegion, error) {
	h, err := openProcess(pid)
	if err != nil {
		return nil, err
	}
	defer windows.CloseHandle(h)

	var regions []MemRegion
	var addr uintptr

	for {
		var mbi windows.MemoryBasicInformation
		ret, err := windows.VirtualQueryEx(h, addr, &mbi, unsafe.Sizeof(mbi))
		if ret == 0 || err != nil {
			break
		}

		// Only include committed pages that are readable.
		if mbi.State == windows.MEM_COMMIT && isReadable(mbi.Protect) {
			regions = append(regions, MemRegion{
				Start: uintptr(mbi.BaseAddress),
				Size:  uintptr(mbi.RegionSize),
				Perms: protectString(mbi.Protect),
			})
		}

		next := uintptr(mbi.BaseAddress) + uintptr(mbi.RegionSize)
		if next <= addr {
			break // overflow guard
		}
		addr = next
	}

	return regions, nil
}

// isReadable returns true for page protection values that allow reading.
func isReadable(protect uint32) bool {
	const noAccess = windows.PAGE_NOACCESS
	const guard = windows.PAGE_GUARD
	if protect&noAccess != 0 || protect&guard != 0 {
		return false
	}
	readable := uint32(windows.PAGE_READONLY | windows.PAGE_READWRITE |
		windows.PAGE_EXECUTE_READ | windows.PAGE_EXECUTE_READWRITE |
		windows.PAGE_EXECUTE_WRITECOPY | windows.PAGE_WRITECOPY)
	return protect&readable != 0
}

// protectString converts a Windows page protection constant to a short
// human-readable string similar to the Darwin "rwx" format.
func protectString(protect uint32) string {
	switch protect &^ (windows.PAGE_GUARD | windows.PAGE_NOCACHE | windows.PAGE_WRITECOMBINE) {
	case windows.PAGE_READONLY:
		return "r--"
	case windows.PAGE_READWRITE:
		return "rw-"
	case windows.PAGE_WRITECOPY:
		return "rw-"
	case windows.PAGE_EXECUTE:
		return "--x"
	case windows.PAGE_EXECUTE_READ:
		return "r-x"
	case windows.PAGE_EXECUTE_READWRITE:
		return "rwx"
	case windows.PAGE_EXECUTE_WRITECOPY:
		return "rwx"
	default:
		return "---"
	}
}
