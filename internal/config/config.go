package config

import (
	"encoding/json"
	"os"
	"path/filepath"
)

type Config struct {
	DiffMaxLines     int      `json:"diffMaxLines,omitempty"`
	CommitStyle      string   `json:"commitStyle,omitempty"`
	RequiredTrailers []string `json:"requiredTrailers,omitempty"`
	OptionalTrailers []string `json:"optionalTrailers,omitempty"`
	BranchPrefixes   []string `json:"branchPrefixes,omitempty"`
	BaseBranch       string   `json:"baseBranch,omitempty"`
	PRTemplate       string   `json:"prTemplate,omitempty"`
	DocsPaths        []string `json:"docsPaths,omitempty"`
}

func Load() (Config, error) {
	cfg := defaults()

	globalPath, err := globalConfigPath()
	if err == nil {
		mergeFromFile(&cfg, globalPath)
	}

	mergeFromFile(&cfg, filepath.Join(".sg", "config.json"))
	mergeFromFile(&cfg, filepath.Join(".sg", "config.local.json"))

	return cfg, nil
}

func defaults() Config {
	return Config{
		DiffMaxLines: 2000,
		CommitStyle:  "conventional",
		BaseBranch:   "main",
	}
}

func globalConfigPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".config", "sg", "config.json"), nil
}

func mergeFromFile(cfg *Config, path string) {
	data, err := os.ReadFile(path)
	if err != nil {
		return
	}

	var layer Config
	if err := json.Unmarshal(data, &layer); err != nil {
		return
	}

	if layer.DiffMaxLines != 0 {
		cfg.DiffMaxLines = layer.DiffMaxLines
	}
	if layer.CommitStyle != "" {
		cfg.CommitStyle = layer.CommitStyle
	}
	if len(layer.RequiredTrailers) > 0 {
		cfg.RequiredTrailers = layer.RequiredTrailers
	}
	if len(layer.OptionalTrailers) > 0 {
		cfg.OptionalTrailers = layer.OptionalTrailers
	}
	if len(layer.BranchPrefixes) > 0 {
		cfg.BranchPrefixes = layer.BranchPrefixes
	}
	if layer.BaseBranch != "" {
		cfg.BaseBranch = layer.BaseBranch
	}
	if layer.PRTemplate != "" {
		cfg.PRTemplate = layer.PRTemplate
	}
	if len(layer.DocsPaths) > 0 {
		cfg.DocsPaths = layer.DocsPaths
	}
}
