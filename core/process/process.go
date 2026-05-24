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

	// Always prefer the ps-based list: sysctl p_comm is truncated at
	// MAXCOMLEN and can be overridden by setprogname(), giving misleading
	// names like "Application". Use sysctl only to verify the process count;
	// fall through to the full-path ps listing for actual display.
	return fallbackListProcesses()
}

// SystemPIDs returns PIDs owned by root (UID 0). Returns nil on error or timeout.
func SystemPIDs() []int {
	out, err := exec.Command("ps", "-axo", "pid=,uid=").Output()
	if err != nil {
		return nil
	}
	var pids []int
	for _, line := range strings.Split(string(out), "\n") {
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		pid, err1 := strconv.Atoi(fields[0])
		uid, err2 := strconv.Atoi(fields[1])
		if err1 == nil && err2 == nil && uid == 0 {
			pids = append(pids, pid)
		}
	}
	return pids
}

// fallbackListProcesses uses `ps` when sysctl parsing fails.
// Uses command= (full path) instead of comm= to avoid MAXCOMLEN truncation
// and to get the real executable name even when a process has overridden
// its argv[0] (e.g. games that call setprogname("Application")).
func fallbackListProcesses() ([]ProcessInfo, error) {
	out, err := exec.Command("ps", "-axo", "pid=,command=").Output()
	if err != nil {
		return nil, fmt.Errorf("ps failed: %w", err)
	}

	var procs []ProcessInfo
	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(line)
		idx := strings.Index(line, " ")
		if idx < 0 {
			continue
		}
		pid, err := strconv.Atoi(line[:idx])
		if err != nil {
			continue
		}
		cmd := strings.TrimSpace(line[idx+1:])
		// Strip any flags/args after the executable path
		if i := strings.Index(cmd, " -"); i > 0 {
			cmd = cmd[:i]
		}
		name := cmd
		if i := strings.LastIndex(cmd, "/"); i >= 0 {
			name = cmd[i+1:]
		}
		if name == "" {
			continue
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
