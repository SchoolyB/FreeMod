package process

import (
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"syscall"
	"unsafe"
)

// ProcessInfo holds basic information about a running process.
type ProcessInfo struct {
	PID  int
	Name string
}

// ListProcesses returns all currently running processes.
func ListProcesses() ([]ProcessInfo, error) {
	// Use sysctl kern.proc.all to list processes
	// We'll parse `ps` output as a portable fallback using sysctl underneath
	mib := []int32{1, 14, 0, 0} // CTL_KERN, KERN_PROC, KERN_PROC_ALL

	// Get required buffer size
	n := uintptr(0)
	_, _, errno := syscall.Syscall6(syscall.SYS___SYSCTL,
		uintptr(unsafe.Pointer(&mib[0])),
		uintptr(len(mib)),
		0, uintptr(unsafe.Pointer(&n)),
		0, 0)
	if errno != 0 {
		return fallbackListProcesses()
	}

	// Allocate and fill buffer
	buf := make([]byte, n)
	_, _, errno = syscall.Syscall6(syscall.SYS___SYSCTL,
		uintptr(unsafe.Pointer(&mib[0])),
		uintptr(len(mib)),
		uintptr(unsafe.Pointer(&buf[0])),
		uintptr(unsafe.Pointer(&n)),
		0, 0)
	if errno != 0 {
		return fallbackListProcesses()
	}

	// kinfo_proc size on macOS/arm64 and amd64 is 648 bytes
	const kinfoSize = 648
	var procs []ProcessInfo
	for i := 0; i+kinfoSize <= int(n); i += kinfoSize {
		// PID is at offset 40 (p_pid in extern_proc inside kinfo_proc)
		pid := int(int32(buf[i+40]) | int32(buf[i+41])<<8 | int32(buf[i+42])<<16 | int32(buf[i+43])<<24)
		// comm (process name) is at offset 56, 17 bytes (MAXCOMLEN+1)
		nameBytes := buf[i+56 : i+56+17]
		end := 0
		for end < len(nameBytes) && nameBytes[end] != 0 {
			end++
		}
		name := string(nameBytes[:end])
		if pid > 0 && name != "" {
			procs = append(procs, ProcessInfo{PID: pid, Name: name})
		}
	}

	if len(procs) == 0 {
		return fallbackListProcesses()
	}

	return procs, nil
}

// fallbackListProcesses uses `ps` when sysctl parsing fails.
func fallbackListProcesses() ([]ProcessInfo, error) {
	out, err := exec.Command("ps", "-axo", "pid,comm").Output()
	if err != nil {
		return nil, fmt.Errorf("ps failed: %w", err)
	}

	var procs []ProcessInfo
	lines := strings.Split(string(out), "\n")
	for _, line := range lines[1:] { // skip header
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, " ", 2)
		if len(parts) < 2 {
			continue
		}
		pid, err := strconv.Atoi(strings.TrimSpace(parts[0]))
		if err != nil {
			continue
		}
		name := strings.TrimSpace(parts[1])
		// ps -o comm gives full path; trim to basename
		if idx := strings.LastIndex(name, "/"); idx >= 0 {
			name = name[idx+1:]
		}
		procs = append(procs, ProcessInfo{PID: pid, Name: name})
	}
	return procs, nil
}

// AttachByPID verifies the process exists. On macOS, actual memory access
// is granted via task_for_pid at read/write time (requires root).
func AttachByPID(pid int) error {
	procs, err := ListProcesses()
	if err != nil {
		return fmt.Errorf("listing processes: %w", err)
	}
	for _, p := range procs {
		if p.PID == pid {
			return nil
		}
	}
	return fmt.Errorf("process %d not found", pid)
}

// DetachByPID is a no-op on macOS since task ports are not held open.
func DetachByPID(pid int) error {
	return nil
}
