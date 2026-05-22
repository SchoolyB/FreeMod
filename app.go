package main

import (
	"context"
	"embed"
	"encoding/base64"
	"encoding/binary"
	"fmt"
	"io/fs"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/freemod/freemod/core/memory"
	"github.com/freemod/freemod/core/process"
	"github.com/freemod/freemod/core/scanner"
	"github.com/freemod/freemod/trainers"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

//go:embed all:trainers
var trainersFS embed.FS

// ── JS-visible types ──────────────────────────────────────────────────────────

type ProcessInfo struct {
	PID  int    `json:"PID"`
	Name string `json:"Name"`
}

type ScanResult struct {
	Addresses []string `json:"Addresses"`
	Count     int      `json:"Count"`
}

type TrainerSummary struct {
	Filename string `json:"Filename"`
	Game     string `json:"Game"`
	Exe      string `json:"Exe"`
	Version  string `json:"Version"`
}

type CheatState struct {
	Name        string `json:"Name"`
	Description string `json:"Description"`
	Enabled     bool   `json:"Enabled"`
	Input       bool   `json:"Input"`
}

type TrainerStatus struct {
	Game      string       `json:"Game"`
	Exe       string       `json:"Exe"`
	Version   string       `json:"Version"`
	PID       int          `json:"PID"`
	Connected bool         `json:"Connected"`
	Cheats    []CheatState `json:"Cheats"`
}

// freezeInterval is how often a frozen cheat re-writes its value.
const freezeInterval = 100 * time.Millisecond

// ── internal cheat state ──────────────────────────────────────────────────────

type activeCheat struct {
	enabled      bool
	restoreBytes []byte // original bytes read before first enable
	writeBytes   []byte // bytes to write while frozen
	addr         uintptr
	cancelFreeze context.CancelFunc // non-nil while freeze goroutine is running
}

// ── App ───────────────────────────────────────────────────────────────────────

type App struct {
	ctx context.Context
	mem memory.Memory

	// dev-mode scanner state
	devMu        sync.Mutex
	devPID       int
	devScanAddrs []uintptr

	// trainer state
	trainerMu     sync.Mutex
	activeTrainer *trainers.TrainerFile
	activePID     int
	activeModBase uintptr
	cheatStates   []activeCheat

	// auto-connect watcher
	watchCancel context.CancelFunc
}

func NewApp() *App {
	return &App{mem: memory.New()}
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
}

// ── Trainer methods ───────────────────────────────────────────────────────────

// userTrainerDir returns (and creates) ~/Library/Application Support/FreeMod/trainers/
func userTrainerDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(home, "Library", "Application Support", "FreeMod", "trainers")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", err
	}
	return dir, nil
}

// ListTrainers returns built-in trainers merged with any user-added ones.
// User trainers with the same filename override built-in ones.
func (a *App) ListTrainers() ([]TrainerSummary, error) {
	// Built-in (embedded)
	sub, err := fs.Sub(trainersFS, "trainers")
	if err != nil {
		return nil, err
	}
	builtIn, err := trainers.ListFromFS(sub)
	if err != nil {
		return nil, err
	}

	// User trainers
	userDir, err := userTrainerDir()
	if err == nil {
		userFS := os.DirFS(userDir)
		userList, _ := trainers.ListFromFS(userFS)
		// Index built-in by filename for dedup
		byName := make(map[string]int, len(builtIn))
		for i, s := range builtIn {
			byName[s.Filename] = i
		}
		for _, s := range userList {
			plain := s.Filename          // matches built-in filenames for dedup
			s.Filename = "user:" + plain // prefix so LoadTrainer knows the source
			if idx, exists := byName[plain]; exists {
				builtIn[idx] = s // user trainer overrides the built-in of the same name
			} else {
				builtIn = append(builtIn, s)
			}
		}
	}

	out := make([]TrainerSummary, len(builtIn))
	for i, s := range builtIn {
		out[i] = TrainerSummary{Filename: s.Filename, Game: s.Game, Exe: s.Exe, Version: s.Version}
	}
	return out, nil
}

// UserTrainerDir exposes the user trainer directory path to the frontend.
func (a *App) UserTrainerDir() string {
	dir, _ := userTrainerDir()
	return dir
}

// TrainerImage returns the trainer's cover art as a data: URI, or "" if the
// trainer declares no image or it can't be read. filename is the value from
// ListTrainers (a built-in name, or "user:"-prefixed for user trainers).
func (a *App) TrainerImage(filename string) string {
	var tf *trainers.TrainerFile
	var readImage func(name string) ([]byte, error)

	if strings.HasPrefix(filename, "user:") {
		userDir, err := userTrainerDir()
		if err != nil {
			return ""
		}
		tf, err = trainers.LoadFromFile(filepath.Join(userDir, strings.TrimPrefix(filename, "user:")))
		if err != nil {
			return ""
		}
		readImage = func(name string) ([]byte, error) { return os.ReadFile(filepath.Join(userDir, name)) }
	} else {
		sub, err := fs.Sub(trainersFS, "trainers")
		if err != nil {
			return ""
		}
		tf, err = trainers.LoadFromFS(sub, filename)
		if err != nil {
			return ""
		}
		readImage = func(name string) ([]byte, error) { return fs.ReadFile(sub, name) }
	}

	if tf.Image == "" {
		return ""
	}
	mime := imageMime(tf.Image)
	if mime == "" {
		return ""
	}
	// The image must sit next to the trainer JSON — basename only, no traversal.
	data, err := readImage(filepath.Base(tf.Image))
	if err != nil {
		return ""
	}
	return "data:" + mime + ";base64," + base64.StdEncoding.EncodeToString(data)
}

// imageMime maps a filename extension to an image MIME type, or "" if unknown.
func imageMime(name string) string {
	switch strings.ToLower(filepath.Ext(name)) {
	case ".png":
		return "image/png"
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".webp":
		return "image/webp"
	case ".gif":
		return "image/gif"
	default:
		return ""
	}
}

// LoadTrainer loads a trainer and returns its current status (not yet connected).
// Filenames prefixed with "user:" are loaded from the user trainer directory.
func (a *App) LoadTrainer(filename string) (TrainerStatus, error) {
	var tf *trainers.TrainerFile
	var err error

	if strings.HasPrefix(filename, "user:") {
		name := strings.TrimPrefix(filename, "user:")
		userDir, dirErr := userTrainerDir()
		if dirErr != nil {
			return TrainerStatus{}, dirErr
		}
		tf, err = trainers.LoadFromFile(filepath.Join(userDir, name))
	} else {
		sub, subErr := fs.Sub(trainersFS, "trainers")
		if subErr != nil {
			return TrainerStatus{}, subErr
		}
		tf, err = trainers.LoadFromFS(sub, filename)
	}
	if err != nil {
		return TrainerStatus{}, err
	}

	a.trainerMu.Lock()
	a.stopAllFreezesLocked()
	a.activeTrainer = tf
	a.activePID = 0
	a.activeModBase = 0
	a.cheatStates = make([]activeCheat, len(tf.Cheats))
	a.trainerMu.Unlock()

	// Start the auto-connect watcher, then connect right now if the game is
	// already running — so LoadTrainer returns an accurate (often already
	// connected) status instead of waiting for the first watcher tick.
	a.startWatcher(tf.Exe)
	a.watchTick(tf.Exe)

	return a.buildStatus(), nil
}

// ConnectTrainer finds the trainer's exe in the process list and attaches.
func (a *App) ConnectTrainer() (TrainerStatus, error) {
	a.trainerMu.Lock()
	tf := a.activeTrainer
	a.trainerMu.Unlock()

	if tf == nil {
		return TrainerStatus{}, fmt.Errorf("no trainer loaded")
	}

	procs, err := process.ListProcesses()
	if err != nil {
		return TrainerStatus{}, fmt.Errorf("listing processes: %w", err)
	}

	pid := 0
	for _, p := range procs {
		if strings.EqualFold(p.Name, tf.Exe) {
			pid = p.PID
			break
		}
	}
	if pid == 0 {
		return TrainerStatus{}, fmt.Errorf("%q is not running", tf.Exe)
	}

	base, err := scanner.FindModuleBase(a.mem, pid)
	if err != nil {
		return TrainerStatus{}, fmt.Errorf("finding module base: %w", err)
	}

	// Resolve addresses before taking the lock (pointer reads can block).
	addrs := make([]uintptr, len(tf.Cheats))
	for i, cheat := range tf.Cheats {
		addrs[i], _ = a.resolveCheatAddr(pid, base, cheat)
	}

	a.trainerMu.Lock()
	a.activePID = pid
	a.activeModBase = base
	for i := range tf.Cheats {
		a.cheatStates[i].addr = addrs[i]
	}
	a.trainerMu.Unlock()

	return a.buildStatus(), nil
}

// ToggleCheat enables or disables cheat at index idx.
// For input cheats, userValue is the value the user typed; ignored otherwise.
func (a *App) ToggleCheat(idx int, enable bool, userValue float64) (TrainerStatus, error) {
	a.trainerMu.Lock()
	defer a.trainerMu.Unlock()

	if a.activeTrainer == nil {
		return TrainerStatus{}, fmt.Errorf("no trainer loaded")
	}
	if a.activePID == 0 {
		return TrainerStatus{}, fmt.Errorf("not connected to game")
	}
	if idx < 0 || idx >= len(a.cheatStates) {
		return TrainerStatus{}, fmt.Errorf("cheat index out of range")
	}

	state := &a.cheatStates[idx]
	cheat := a.activeTrainer.Cheats[idx]
	addr := state.addr

	if addr == 0 {
		return TrainerStatus{}, fmt.Errorf("address not resolved for %q", cheat.Name)
	}

	if enable && !state.enabled {
		size := cheatTypeSize(cheat.Type)
		// Read and save original bytes.
		orig, err := a.mem.ReadBytes(a.activePID, addr, size)
		if err != nil {
			return TrainerStatus{}, fmt.Errorf("reading original value: %w", err)
		}
		writeVal := cheat.Value
		if cheat.Input {
			writeVal = userValue
		}
		wb, err := encodeCheatValue(cheat.Type, writeVal)
		if err != nil {
			return TrainerStatus{}, err
		}
		state.restoreBytes = orig
		state.writeBytes = wb
		state.enabled = true

		// Write once immediately, then freeze on an interval.
		_ = a.mem.WriteBytes(a.activePID, addr, wb)

		pid := a.activePID
		mem := a.mem
		ctx, cancel := context.WithCancel(context.Background())
		state.cancelFreeze = cancel

		go func() {
			ticker := time.NewTicker(freezeInterval)
			defer ticker.Stop()
			for {
				select {
				case <-ctx.Done():
					return
				case <-ticker.C:
					_ = mem.WriteBytes(pid, addr, wb)
				}
			}
		}()

	} else if !enable && state.enabled {
		// Stop the freeze goroutine first, then restore.
		if state.cancelFreeze != nil {
			state.cancelFreeze()
			state.cancelFreeze = nil
		}
		if err := a.mem.WriteBytes(a.activePID, addr, state.restoreBytes); err != nil {
			return TrainerStatus{}, fmt.Errorf("restoring value: %w", err)
		}
		state.enabled = false
	}

	return a.buildStatusLocked(), nil
}

// GetTrainerStatus returns the current trainer status without changing anything.
func (a *App) GetTrainerStatus() TrainerStatus {
	return a.buildStatus()
}

// DisconnectTrainer restores all original values, stops freeze goroutines,
// and detaches from the game. The game goes back to its normal state.
func (a *App) DisconnectTrainer() TrainerStatus {
	a.stopWatcher()
	a.trainerMu.Lock()
	pid := a.activePID
	for i := range a.cheatStates {
		state := &a.cheatStates[i]
		if state.cancelFreeze != nil {
			state.cancelFreeze()
			state.cancelFreeze = nil
		}
		if state.enabled && len(state.restoreBytes) > 0 && state.addr != 0 {
			_ = a.mem.WriteBytes(pid, state.addr, state.restoreBytes)
		}
		state.enabled = false
	}
	a.activePID = 0
	a.activeModBase = 0
	a.trainerMu.Unlock()
	return a.buildStatus()
}

// KillApp disconnects (restoring all cheat values) then closes the app.
func (a *App) KillApp() {
	a.DisconnectTrainer()
	runtime.Quit(a.ctx)
}

func (a *App) buildStatus() TrainerStatus {
	a.trainerMu.Lock()
	defer a.trainerMu.Unlock()
	return a.buildStatusLocked()
}

func (a *App) buildStatusLocked() TrainerStatus {
	if a.activeTrainer == nil {
		return TrainerStatus{}
	}
	cheats := make([]CheatState, len(a.activeTrainer.Cheats))
	for i, c := range a.activeTrainer.Cheats {
		cheats[i] = CheatState{
			Name:        c.Name,
			Description: c.Description,
			Enabled:     a.cheatStates[i].enabled,
			Input:       c.Input,
		}
	}
	return TrainerStatus{
		Game:      a.activeTrainer.Game,
		Exe:       a.activeTrainer.Exe,
		Version:   a.activeTrainer.Version,
		PID:       a.activePID,
		Connected: a.activePID > 0,
		Cheats:    cheats,
	}
}

// ── Dev-mode methods ──────────────────────────────────────────────────────────

func (a *App) ListProcesses() ([]ProcessInfo, error) {
	procs, err := process.ListProcesses()
	if err != nil {
		return nil, err
	}
	result := make([]ProcessInfo, len(procs))
	for i, p := range procs {
		result[i] = ProcessInfo{PID: p.PID, Name: p.Name}
	}
	return result, nil
}

func (a *App) GetModuleBase(pid int) string {
	base, err := scanner.FindModuleBase(a.mem, pid)
	if err != nil {
		return ""
	}
	return fmt.Sprintf("0x%x", base)
}

func (a *App) AttachProcess(pid int) error {
	if err := process.AttachByPID(pid); err != nil {
		return err
	}
	a.devMu.Lock()
	a.devPID = pid
	a.devScanAddrs = nil
	a.devMu.Unlock()
	return nil
}

func (a *App) AttachedPID() int {
	a.devMu.Lock()
	defer a.devMu.Unlock()
	return a.devPID
}

func (a *App) ScanValue(val int32) (ScanResult, error) {
	a.devMu.Lock()
	pid := a.devPID
	a.devMu.Unlock()

	if pid == 0 {
		return ScanResult{}, fmt.Errorf("no process attached")
	}
	addrs, err := scanner.ScanForInt(a.mem, pid, val)
	if err != nil {
		return ScanResult{}, err
	}
	a.devMu.Lock()
	a.devScanAddrs = addrs
	a.devMu.Unlock()
	return toScanResult(addrs), nil
}

func (a *App) NarrowValue(val int32) (ScanResult, error) {
	a.devMu.Lock()
	pid := a.devPID
	prev := a.devScanAddrs
	a.devMu.Unlock()

	if pid == 0 {
		return ScanResult{}, fmt.Errorf("no process attached")
	}
	if prev == nil {
		return ScanResult{}, fmt.Errorf("no previous scan")
	}
	addrs, err := scanner.NarrowScan(a.mem, pid, prev, val)
	if err != nil {
		return ScanResult{}, err
	}
	a.devMu.Lock()
	a.devScanAddrs = addrs
	a.devMu.Unlock()
	return toScanResult(addrs), nil
}

func (a *App) WriteValue(addrHex string, val int32) error {
	a.devMu.Lock()
	pid := a.devPID
	a.devMu.Unlock()

	if pid == 0 {
		return fmt.Errorf("no process attached")
	}
	addr, err := parseHex(addrHex)
	if err != nil {
		return fmt.Errorf("invalid address %q: %w", addrHex, err)
	}
	return a.mem.WriteInt(pid, addr, val)
}

func (a *App) ReadValue(addrHex string) (int32, error) {
	a.devMu.Lock()
	pid := a.devPID
	a.devMu.Unlock()

	if pid == 0 {
		return 0, fmt.Errorf("no process attached")
	}
	addr, err := parseHex(addrHex)
	if err != nil {
		return 0, fmt.Errorf("invalid address %q: %w", addrHex, err)
	}
	return a.mem.ReadInt(pid, addr)
}

// ── helpers ───────────────────────────────────────────────────────────────────

// startWatcher launches a goroutine that auto-connects/disconnects as the game
// starts/stops. It checks once immediately, then every second.
func (a *App) startWatcher(exe string) {
	a.stopWatcher()
	ctx, cancel := context.WithCancel(context.Background())
	a.watchCancel = cancel

	go func() {
		ticker := time.NewTicker(time.Second)
		defer ticker.Stop()
		for {
			a.watchTick(exe)
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
			}
		}
	}()
}

// watchTick runs one auto-connect/disconnect check: it connects when the game
// appears (or restarts under a new PID) and disconnects when it stops. It is
// idempotent — a no-op when already in the right state — so it is safe to call
// directly (e.g. from LoadTrainer) as well as from the watcher goroutine.
func (a *App) watchTick(exe string) {
	a.trainerMu.Lock()
	currentPID := a.activePID
	a.trainerMu.Unlock()

	procs, err := process.ListProcesses()
	if err != nil {
		return
	}

	found := 0
	for _, p := range procs {
		if strings.EqualFold(p.Name, exe) {
			found = p.PID
			break
		}
	}

	switch {
	case found != 0 && found != currentPID:
		// Game appeared or restarted — connect.
		base, err := scanner.FindModuleBase(a.mem, found)
		if err != nil {
			return
		}
		a.trainerMu.Lock()
		tf := a.activeTrainer
		a.trainerMu.Unlock()
		if tf == nil || !strings.EqualFold(tf.Exe, exe) {
			return
		}
		// Resolve addresses outside the lock (pointer reads can block).
		addrs := make([]uintptr, len(tf.Cheats))
		for i, cheat := range tf.Cheats {
			addrs[i], _ = a.resolveCheatAddr(found, base, cheat)
		}
		a.trainerMu.Lock()
		a.stopAllFreezesLocked()
		a.activePID = found
		a.activeModBase = base
		for i := range tf.Cheats {
			a.cheatStates[i].addr = addrs[i]
			a.cheatStates[i].enabled = false
			a.cheatStates[i].cancelFreeze = nil
		}
		a.trainerMu.Unlock()

	case found == 0 && currentPID != 0:
		// Game stopped — disconnect.
		a.trainerMu.Lock()
		a.stopAllFreezesLocked()
		a.activePID = 0
		a.activeModBase = 0
		for i := range a.cheatStates {
			a.cheatStates[i].enabled = false
		}
		a.trainerMu.Unlock()
	}
}

func (a *App) stopWatcher() {
	if a.watchCancel != nil {
		a.watchCancel()
		a.watchCancel = nil
	}
}

// stopAllFreezesLocked cancels every active freeze goroutine.
// Must be called with trainerMu held.
func (a *App) stopAllFreezesLocked() {
	for i := range a.cheatStates {
		if a.cheatStates[i].cancelFreeze != nil {
			a.cheatStates[i].cancelFreeze()
			a.cheatStates[i].cancelFreeze = nil
		}
	}
}

// resolveCheatAddr resolves the final memory address for a cheat.
// If the cheat has offsets, it walks the pointer chain from base+baseOffset.
// Otherwise it returns base+baseOffset directly.
func (a *App) resolveCheatAddr(pid int, base uintptr, cheat trainers.Cheat) (uintptr, error) {
	baseOffset, err := parseHex(cheat.BaseOffset)
	if err != nil {
		return 0, fmt.Errorf("invalid base_offset %q: %w", cheat.BaseOffset, err)
	}
	addr := base + baseOffset

	if len(cheat.Offsets) == 0 {
		return addr, nil
	}

	offsets := make([]uintptr, len(cheat.Offsets))
	for i, s := range cheat.Offsets {
		off, err := parseHex(s)
		if err != nil {
			return 0, fmt.Errorf("invalid offset[%d] %q: %w", i, s, err)
		}
		offsets[i] = off
	}
	return scanner.ResolvePointer(a.mem, pid, addr, offsets)
}

func cheatTypeSize(t string) int {
	switch t {
	case "int64":
		return 8
	default: // "int32", "float32"
		return 4
	}
}

func encodeCheatValue(t string, value float64) ([]byte, error) {
	buf := make([]byte, cheatTypeSize(t))
	switch t {
	case "float32":
		binary.LittleEndian.PutUint32(buf, math.Float32bits(float32(value)))
	case "int64":
		binary.LittleEndian.PutUint64(buf, uint64(int64(value)))
	default: // "int32"
		binary.LittleEndian.PutUint32(buf, uint32(int32(value)))
	}
	return buf, nil
}

func parseHex(s string) (uintptr, error) {
	clean := strings.TrimPrefix(strings.TrimPrefix(s, "0x"), "0X")
	v, err := strconv.ParseUint(clean, 16, 64)
	return uintptr(v), err
}

func toScanResult(addrs []uintptr) ScanResult {
	strs := make([]string, len(addrs))
	for i, a := range addrs {
		strs[i] = fmt.Sprintf("0x%x", a)
	}
	return ScanResult{Addresses: strs, Count: len(addrs)}
}
