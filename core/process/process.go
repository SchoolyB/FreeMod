package process

import "fmt"

// ProcessInfo holds basic information about a running process.
type ProcessInfo struct {
	PID  int
	Name string
}

// AttachByPID verifies the process exists.
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

// DetachByPID is a no-op — no persistent handles are held on either platform.
func DetachByPID(pid int) error {
	return nil
}
