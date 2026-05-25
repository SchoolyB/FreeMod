//go:build windows

package process

import (
	"fmt"
	"unsafe"

	"golang.org/x/sys/windows"
)

// ListProcesses returns all currently running processes using the Windows
// Toolhelp32 snapshot API.
func ListProcesses() ([]ProcessInfo, error) {
	snap, err := windows.CreateToolhelp32Snapshot(windows.TH32CS_SNAPPROCESS, 0)
	if err != nil {
		return nil, fmt.Errorf("CreateToolhelp32Snapshot: %w", err)
	}
	defer windows.CloseHandle(snap)

	var entry windows.ProcessEntry32
	entry.Size = uint32(unsafe.Sizeof(entry))

	if err := windows.Process32First(snap, &entry); err != nil {
		return nil, fmt.Errorf("Process32First: %w", err)
	}

	var procs []ProcessInfo
	for {
		name := windows.UTF16ToString(entry.ExeFile[:])
		if name != "" && entry.ProcessID > 0 {
			procs = append(procs, ProcessInfo{
				PID:  int(entry.ProcessID),
				Name: name,
			})
		}
		if err := windows.Process32Next(snap, &entry); err != nil {
			break
		}
	}
	return procs, nil
}

// SystemPIDs returns an empty slice on Windows. The concept of root-owned
// processes doesn't map cleanly — the system-process warning is Darwin-specific.
func SystemPIDs() []int {
	return []int{}
}

// ProcessExists returns true if a process with the given PID is still running.
func ProcessExists(pid int) bool {
	h, err := windows.OpenProcess(windows.PROCESS_QUERY_LIMITED_INFORMATION, false, uint32(pid))
	if err != nil {
		return false
	}
	defer windows.CloseHandle(h)
	var code uint32
	if err := windows.GetExitCodeProcess(h, &code); err != nil {
		return false
	}
	return code == 259 // STILL_ACTIVE
}
