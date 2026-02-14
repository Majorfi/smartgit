package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadDefaults(t *testing.T) {
	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	if cfg.DiffMaxLines != 2000 {
		t.Errorf("expected DiffMaxLines 2000, got %d", cfg.DiffMaxLines)
	}
	if cfg.BaseBranch != "" {
		t.Errorf("expected BaseBranch '', got %q", cfg.BaseBranch)
	}
	if cfg.CommitStyle != "conventional" {
		t.Errorf("expected CommitStyle 'conventional', got %q", cfg.CommitStyle)
	}
}

func TestLoadFromFile(t *testing.T) {
	dir := t.TempDir()
	sgDir := filepath.Join(dir, ".sg")
	os.MkdirAll(sgDir, 0755)

	configData := `{"commitStyle":"angular","baseBranch":"develop"}`
	os.WriteFile(filepath.Join(sgDir, "config.json"), []byte(configData), 0644)

	originalDir, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(originalDir)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	if cfg.CommitStyle != "angular" {
		t.Errorf("expected commitStyle 'angular', got %q", cfg.CommitStyle)
	}
	if cfg.BaseBranch != "develop" {
		t.Errorf("expected baseBranch 'develop', got %q", cfg.BaseBranch)
	}
	if cfg.DiffMaxLines != 2000 {
		t.Errorf("default DiffMaxLines should be preserved, got %d", cfg.DiffMaxLines)
	}
}

func TestLoadLayerOverride(t *testing.T) {
	dir := t.TempDir()
	sgDir := filepath.Join(dir, ".sg")
	os.MkdirAll(sgDir, 0755)

	projectConfig := `{"baseBranch":"develop","commitStyle":"angular"}`
	os.WriteFile(filepath.Join(sgDir, "config.json"), []byte(projectConfig), 0644)

	localConfig := `{"baseBranch":"staging"}`
	os.WriteFile(filepath.Join(sgDir, "config.local.json"), []byte(localConfig), 0644)

	originalDir, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(originalDir)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	if cfg.BaseBranch != "staging" {
		t.Errorf("local should override project: expected 'staging', got %q", cfg.BaseBranch)
	}
	if cfg.CommitStyle != "angular" {
		t.Errorf("project value should survive: expected 'angular', got %q", cfg.CommitStyle)
	}
}
