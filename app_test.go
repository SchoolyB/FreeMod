package main

import (
	"context"
	"os/exec"
	"testing"
	"time"
)

// TestAutoConnect starts the demo target, loads its trainer, and checks that
// the background watcher connects on its own (no ConnectTrainer call).
func TestAutoConnect(t *testing.T) {
	target := exec.Command("./build/bin/target")
	if err := target.Start(); err != nil {
		t.Fatalf("starting demo target: %v", err)
	}
	defer target.Process.Kill()
	time.Sleep(1500 * time.Millisecond) // let it boot and map memory

	app := NewApp()
	app.startup(context.Background())

	if _, err := app.LoadTrainer("target.json"); err != nil {
		t.Fatalf("LoadTrainer: %v", err)
	}

	deadline := time.Now().Add(12 * time.Second)
	for time.Now().Before(deadline) {
		s := app.GetTrainerStatus()
		if s.Connected {
			t.Logf("auto-connected: pid=%d", s.PID)
			return
		}
		time.Sleep(300 * time.Millisecond)
	}
	t.Fatal("watcher never auto-connected within 12s")
}
