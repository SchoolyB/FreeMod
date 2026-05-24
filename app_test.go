package main

import (
	"context"
	"os/exec"
	"testing"
	"time"
)

// TestLoadTrainer_Parses loads the bundled demo trainer from the embedded FS
// and verifies the parsed fields land on the App. No external process or
// memory access required.
func TestLoadTrainer_Parses(t *testing.T) {
	app := NewApp()
	app.startup(context.Background())
	app.settings.AutoConnect = false // don't spin up the watcher goroutine

	status, err := app.LoadTrainer("target.json.example")
	if err != nil {
		t.Fatalf("LoadTrainer: %v", err)
	}

	if got, want := status.Game, "FreeMod Demo Target"; got != want {
		t.Errorf("Game = %q, want %q", got, want)
	}
	if got, want := status.Exe, "freemod-demo"; got != want {
		t.Errorf("Exe = %q, want %q", got, want)
	}
	if got, want := status.Version, "1.0"; got != want {
		t.Errorf("Version = %q, want %q", got, want)
	}
	if status.Connected {
		t.Errorf("Connected = true, want false (no process running)")
	}
	if got, want := len(status.Cheats), 2; got != want {
		t.Fatalf("len(Cheats) = %d, want %d", got, want)
	}
	if got, want := status.Cheats[0].Name, "Infinite Health"; got != want {
		t.Errorf("Cheats[0].Name = %q, want %q", got, want)
	}
	if got, want := status.Cheats[1].Name, "Set Health"; got != want {
		t.Errorf("Cheats[1].Name = %q, want %q", got, want)
	}
	if !status.Cheats[1].Input {
		t.Errorf("Cheats[1].Input = false, want true")
	}
}

func TestLoadTrainer_Missing(t *testing.T) {
	app := NewApp()
	app.startup(context.Background())
	app.settings.AutoConnect = false

	if _, err := app.LoadTrainer("nonexistent.json"); err == nil {
		t.Fatal("LoadTrainer(nonexistent.json) returned nil error, want error")
	}
}

// TestAutoConnect starts the demo target, loads its trainer, and checks that
// the background watcher connects on its own (no ConnectTrainer call).
// Integration test: requires `make build` (or the demo build steps) and a
// signed/entitled test runner to call task_for_pid on macOS.
func TestAutoConnect(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test: needs a built demo binary and macOS memory-access permissions")
	}
	target := exec.Command("./build/bin/freemod-demo")
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
