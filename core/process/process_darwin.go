//go:build darwin

package process

import (
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"syscall"
	"unsafe"
)

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

	// Always prefer the ps-based list: sysctl p_comm is truncated at
	// MAXCOMLEN and can be overridden by setprogname(), giving misleading
	// names like "Application". Fall through to the full-path ps listing.
	return fallbackListProcesses()
}

// SystemPIDs returns PIDs owned by root (UID 0). Returns nil on error.
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

// ProcessExists returns true if the process with the given PID is still running.
func ProcessExists(pid int) bool {
	return syscall.Kill(pid, 0) != syscall.ESRCH
}

// fallbackListProcesses uses `ps` to get full executable paths, avoiding
// MAXCOMLEN truncation and setprogname() overrides.
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
