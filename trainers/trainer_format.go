package trainers

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// Cheat describes a single moddable value in a game.
type Cheat struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Type        string   `json:"type"`        // "int32" | "int64" | "float32"
	BaseOffset  string   `json:"base_offset"` // hex offset from main module base, e.g. "0x15c344"
	Offsets     []string `json:"offsets"`     // optional pointer chain offsets, e.g. ["0x10","0x4c"]
	Value       float64  `json:"value"`       // written when cheat is enabled (ignored if Input is true)
	Input       bool     `json:"input"`       // true = user provides the value at runtime
}

// TrainerFile is the root struct for a trainer JSON file.
type TrainerFile struct {
	Game    string  `json:"game"`
	Exe     string  `json:"exe"`     // process name to match (without .exe on macOS)
	Version string  `json:"version"` // game version this trainer targets
	Image   string  `json:"image"`   // optional cover-art filename, alongside this JSON
	Cheats  []Cheat `json:"cheats"`
}

// Summary is a lightweight descriptor used in the UI list.
type Summary struct {
	Filename string
	Game     string
	Exe      string
	Version  string
}

// LoadFromFile reads and parses a trainer JSON file.
func LoadFromFile(path string) (*TrainerFile, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", path, err)
	}
	var tf TrainerFile
	if err := json.Unmarshal(data, &tf); err != nil {
		return nil, fmt.Errorf("parsing %s: %w", path, err)
	}
	return &tf, nil
}

// LoadFromFS reads and parses a trainer JSON from an fs.FS (e.g. embed.FS).
func LoadFromFS(fsys fs.FS, name string) (*TrainerFile, error) {
	data, err := fs.ReadFile(fsys, name)
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", name, err)
	}
	var tf TrainerFile
	if err := json.Unmarshal(data, &tf); err != nil {
		return nil, fmt.Errorf("parsing %s: %w", name, err)
	}
	return &tf, nil
}

// ListFromFS returns Summary descriptors for all *.json files in an fs.FS.
func ListFromFS(fsys fs.FS) ([]Summary, error) {
	var out []Summary
	err := fs.WalkDir(fsys, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(path, ".json") {
			return err
		}
		tf, err := LoadFromFS(fsys, path)
		if err != nil {
			return nil // skip malformed files
		}
		out = append(out, Summary{
			Filename: filepath.Base(path),
			Game:     tf.Game,
			Exe:      tf.Exe,
			Version:  tf.Version,
		})
		return nil
	})
	return out, err
}
