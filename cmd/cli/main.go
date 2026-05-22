package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"

	"github.com/freemod/freemod/core/memory"
	"github.com/freemod/freemod/core/process"
	"github.com/freemod/freemod/core/scanner"
)

const scanFile = "/tmp/freetrainer_scan.json"

type scanState struct {
	PID       int      `json:"pid"`
	Value     int32    `json:"value"`
	Addresses []string `json:"addresses"`
}

func usage() {
	fmt.Fprintf(os.Stderr, `FreeTrainer CLI — Phase 2

Usage:
  freetrainer list
      List all running processes.

  freetrainer read <pid> <addr>
      Read the int32 value at the given address in the target process.

  freetrainer write <pid> <addr> <value>
      Write an int32 value to the given address in the target process.

  freetrainer scan <pid> <value>
      Scan all readable memory of <pid> for int32 <value>.
      Results saved to %s for use with 'narrow'.

  freetrainer narrow <pid> <value>
      Filter the previous scan results to addresses whose value is now <value>.

Examples:
  freetrainer list
  freetrainer scan   1234 100
  freetrainer narrow 1234 99
  freetrainer write  1234 0x1400010cef4 9999
`, scanFile)
	os.Exit(1)
}

func mustParsePID(s string) int {
	pid, err := strconv.Atoi(s)
	if err != nil || pid <= 0 {
		fmt.Fprintf(os.Stderr, "invalid pid: %s\n", s)
		os.Exit(1)
	}
	return pid
}

func mustParseAddr(s string) uintptr {
	var addr uint64
	var err error
	if len(s) > 2 && (s[:2] == "0x" || s[:2] == "0X") {
		addr, err = strconv.ParseUint(s[2:], 16, 64)
	} else {
		addr, err = strconv.ParseUint(s, 10, 64)
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "invalid address: %s\n", s)
		os.Exit(1)
	}
	return uintptr(addr)
}

func mustParseInt32(s string) int32 {
	v, err := strconv.ParseInt(s, 10, 32)
	if err != nil {
		fmt.Fprintf(os.Stderr, "invalid int32 value: %s\n", s)
		os.Exit(1)
	}
	return int32(v)
}

func saveScan(pid int, value int32, addrs []uintptr) error {
	state := scanState{
		PID:   pid,
		Value: value,
	}
	for _, a := range addrs {
		state.Addresses = append(state.Addresses, fmt.Sprintf("0x%x", a))
	}
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(scanFile, data, 0600)
}

func loadScan() (*scanState, error) {
	data, err := os.ReadFile(scanFile)
	if err != nil {
		return nil, fmt.Errorf("no scan results found (run 'scan' first): %w", err)
	}
	var state scanState
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, fmt.Errorf("corrupt scan file: %w", err)
	}
	return &state, nil
}

func parseAddrs(strs []string) []uintptr {
	addrs := make([]uintptr, 0, len(strs))
	for _, s := range strs {
		addrs = append(addrs, mustParseAddr(s))
	}
	return addrs
}

func printScanResults(addrs []uintptr, value int32) {
	fmt.Printf("Found %d match(es) for value %d:\n", len(addrs), value)
	limit := len(addrs)
	if limit > 20 {
		limit = 20
		fmt.Printf("(showing first 20 of %d — keep narrowing)\n", len(addrs))
	}
	for _, a := range addrs[:limit] {
		fmt.Printf("  0x%x\n", a)
	}
}

func main() {
	if len(os.Args) < 2 {
		usage()
	}

	mem := memory.New()

	switch os.Args[1] {
	case "list":
		procs, err := process.ListProcesses()
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("%-8s  %s\n", "PID", "NAME")
		for _, p := range procs {
			fmt.Printf("%-8d  %s\n", p.PID, p.Name)
		}

	case "read":
		if len(os.Args) < 4 {
			usage()
		}
		pid := mustParsePID(os.Args[2])
		addr := mustParseAddr(os.Args[3])

		if err := process.AttachByPID(pid); err != nil {
			fmt.Fprintf(os.Stderr, "attach failed: %v\n", err)
			os.Exit(1)
		}
		val, err := mem.ReadInt(pid, addr)
		if err != nil {
			fmt.Fprintf(os.Stderr, "read failed: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("int32 @ 0x%x = %d\n", addr, val)

	case "write":
		if len(os.Args) < 5 {
			usage()
		}
		pid := mustParsePID(os.Args[2])
		addr := mustParseAddr(os.Args[3])
		val := mustParseInt32(os.Args[4])

		if err := process.AttachByPID(pid); err != nil {
			fmt.Fprintf(os.Stderr, "attach failed: %v\n", err)
			os.Exit(1)
		}
		if err := mem.WriteInt(pid, addr, val); err != nil {
			fmt.Fprintf(os.Stderr, "write failed: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("wrote %d to 0x%x\n", val, addr)

	case "scan":
		if len(os.Args) < 4 {
			usage()
		}
		pid := mustParsePID(os.Args[2])
		value := mustParseInt32(os.Args[3])

		if err := process.AttachByPID(pid); err != nil {
			fmt.Fprintf(os.Stderr, "attach failed: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("Scanning pid %d for value %d...\n", pid, value)
		addrs, err := scanner.ScanForInt(mem, pid, value)
		if err != nil {
			fmt.Fprintf(os.Stderr, "scan failed: %v\n", err)
			os.Exit(1)
		}

		if err := saveScan(pid, value, addrs); err != nil {
			fmt.Fprintf(os.Stderr, "warning: could not save scan results: %v\n", err)
		} else {
			fmt.Printf("Results saved to %s\n", scanFile)
		}

		printScanResults(addrs, value)

	case "narrow":
		if len(os.Args) < 4 {
			usage()
		}
		pid := mustParsePID(os.Args[2])
		value := mustParseInt32(os.Args[3])

		state, err := loadScan()
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
		if state.PID != pid {
			fmt.Fprintf(os.Stderr, "warning: scan file is for pid %d, but you specified %d\n", state.PID, pid)
		}

		prevAddrs := parseAddrs(state.Addresses)
		fmt.Printf("Narrowing %d candidate(s) to value %d...\n", len(prevAddrs), value)

		addrs, err := scanner.NarrowScan(mem, pid, prevAddrs, value)
		if err != nil {
			fmt.Fprintf(os.Stderr, "narrow failed: %v\n", err)
			os.Exit(1)
		}

		if err := saveScan(pid, value, addrs); err != nil {
			fmt.Fprintf(os.Stderr, "warning: could not save scan results: %v\n", err)
		} else {
			fmt.Printf("Results saved to %s\n", scanFile)
		}

		printScanResults(addrs, value)

		if len(addrs) == 1 {
			fmt.Printf("\nFound unique address: 0x%x\n", addrs[0])
			fmt.Printf("Write with: freetrainer write %d 0x%x <value>\n", pid, addrs[0])
		}

	default:
		usage()
	}
}
