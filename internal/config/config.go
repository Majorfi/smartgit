package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type Config struct {
	DiffMaxLines     int      `json:"diffMaxLines"`
	CommitStyle      string   `json:"commitStyle,omitempty"`
	RequiredTrailers []string `json:"requiredTrailers,omitempty"`
	OptionalTrailers []string `json:"optionalTrailers,omitempty"`
	BranchPrefixes   []string `json:"branchPrefixes,omitempty"`
	BaseBranch       string   `json:"baseBranch,omitempty"`
	PRTemplate       string   `json:"prTemplate,omitempty"`
	DocsPaths        []string `json:"docsPaths,omitempty"`
}

type configLayer struct {
	DiffMaxLines     *int     `json:"diffMaxLines,omitempty"`
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
	var errs []string

	globalPath, err := globalConfigPath()
	if err == nil {
		if err := mergeFromFile(&cfg, globalPath); err != nil {
			errs = append(errs, fmt.Sprintf("%s: %v", globalPath, err))
		}
	}

	projectPath := filepath.Join(".sg", "config.json")
	if err := mergeFromFile(&cfg, projectPath); err != nil {
		errs = append(errs, fmt.Sprintf("%s: %v", projectPath, err))
	}

	localPath := filepath.Join(".sg", "config.local.json")
	if err := mergeFromFile(&cfg, localPath); err != nil {
		errs = append(errs, fmt.Sprintf("%s: %v", localPath, err))
	}

	if len(errs) > 0 {
		return cfg, fmt.Errorf("%s", strings.Join(errs, "; "))
	}
	return cfg, nil
}

func defaults() Config {
	return Config{
		DiffMaxLines: 2000,
		CommitStyle:  "conventional",
	}
}

func globalConfigPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".config", "sg", "config.json"), nil
}

func mergeFromFile(cfg *Config, path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return fmt.Errorf("read error: %w", err)
	}

	var layer configLayer
	if err := json.Unmarshal(data, &layer); err != nil {
		return fmt.Errorf("invalid JSON: %w", err)
	}

	if layer.DiffMaxLines != nil {
		cfg.DiffMaxLines = *layer.DiffMaxLines
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
	return nil
}
