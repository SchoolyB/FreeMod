package main

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// Settings holds user-configurable preferences persisted to disk.
type Settings struct {
	AutoConnect      bool   `json:"AutoConnect"`
	FreezeIntervalMs int    `json:"FreezeIntervalMs"`
	TrainerDir       string `json:"TrainerDir"`
}

func defaultSettings() Settings {
	return Settings{
		AutoConnect:      true,
		FreezeIntervalMs: 100,
		TrainerDir:       "",
	}
}

func settingsPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(home, "Library", "Application Support", "FreeMod")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", err
	}
	return filepath.Join(dir, "settings.json"), nil
}

func loadSettings() Settings {
	s := defaultSettings()
	path, err := settingsPath()
	if err != nil {
		return s
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return s
	}
	_ = json.Unmarshal(data, &s)
	s = clampSettings(s)
	return s
}

func clampSettings(s Settings) Settings {
	if s.FreezeIntervalMs < 50 {
		s.FreezeIntervalMs = 50
	}
	if s.FreezeIntervalMs > 1000 {
		s.FreezeIntervalMs = 1000
	}
	return s
}

func (s Settings) save() error {
	path, err := settingsPath()
	if err != nil {
		return err
	}
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}
