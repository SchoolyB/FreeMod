package main

import (
	"fmt"
	"os"
	"time"
	"unsafe"
)

var health int32 = 100

func main() {
	fmt.Printf("Target PID: %d\n", os.Getpid())
	fmt.Printf("Health address: 0x%x\n", uintptr(unsafe.Pointer(&health)))
	fmt.Println("Printing health every second. Modify it with freetrainer CLI.")
	fmt.Println()

	for {
		fmt.Printf("health = %d\n", health)
		time.Sleep(1 * time.Second)
	}
}
