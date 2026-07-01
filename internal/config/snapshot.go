package config

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/ntk148v/knit/internal/skills"
)

func Path() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "knit", "knit.json"), nil
}

func Load() (skills.Snapshot, error) {
	path, err := Path()
	if err != nil {
		return skills.Snapshot{}, err
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return skills.Snapshot{}, err
	}
	var snap skills.Snapshot
	return snap, json.Unmarshal(b, &snap)
}

func Save(s skills.Snapshot) (string, error) {
	path, err := Path()
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return "", err
	}
	b, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return "", err
	}
	return path, os.WriteFile(path, b, 0o644)
}
