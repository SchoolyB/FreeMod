// Command target is a demo process for exercising FreeMod end-to-end.
//
// It holds a single global int32 `health` and prints it once a second.
// On startup it inspects its own memory to report the static offset of
// `health` from the main module base — that offset is what a trainer JSON
// puts in "base_offset" so FreeMod can locate the value after ASLR.
package main

import (
	"fmt"
	"os"
	"time"
	"unsafe"

	"github.com/freemod/freemod/core/memory"
	"github.com/freemod/freemod/core/scanner"
)

var health int32 = 100

func main() {
	pid := os.Getpid()
	healthAddr := uintptr(unsafe.Pointer(&health))

	fmt.Printf("Target PID:       %d\n", pid)
	fmt.Printf("Health address:   0x%x\n", healthAddr)

	// task_for_pid on our own pid succeeds without root, so we can locate
	// our own Mach-O base and compute the ASLR-independent static offset.
	if base, err := scanner.FindModuleBase(memory.New(), pid); err == nil {
		fmt.Printf("Module base:      0x%x\n", base)
		fmt.Printf("Static offset:    0x%x  <- put this in target.json \"base_offset\"\n", healthAddr-base)
	} else {
		fmt.Printf("Module base:      (unavailable: %v)\n", err)
	}

	fmt.Println()
	fmt.Println("Printing health every second. Modify it with FreeMod or the CLI.")
	fmt.Println()

	for {
		fmt.Printf("health = %d\n", health)
		time.Sleep(1 * time.Second)
	}
}
