package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadMissingUsesDefaults(t *testing.T) {
	cfg, err := Load(filepath.Join(t.TempDir(), "nope.json"))
	if err != nil {
		t.Fatalf("missing config must not error: %v", err)
	}
	if cfg.MinScore != 20 {
		t.Fatalf("MinScore = %d, want 20", cfg.MinScore)
	}
	if !cfg.SkipExcluded {
		t.Fatal("SkipExcluded must default to true")
	}
	if len(cfg.Include) == 0 || len(cfg.Locations) == 0 || len(cfg.Sources) != 6 {
		t.Fatalf("defaults incomplete: %+v", cfg)
	}
}

func TestLoadOverridesOnlyWhatIsSet(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	if err := os.WriteFile(path, []byte(`{"min_score": 50, "include": ["rust"]}`), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.MinScore != 50 {
		t.Fatalf("MinScore = %d, want 50", cfg.MinScore)
	}
	if len(cfg.Include) != 1 || cfg.Include[0] != "rust" {
		t.Fatalf("Include = %v, want [rust]", cfg.Include)
	}
	if !cfg.SkipExcluded || len(cfg.Locations) == 0 {
		t.Fatalf("unset fields must keep defaults: %+v", cfg)
	}
}

func TestLoadBrokenFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	if err := os.WriteFile(path, []byte("{not json"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(path); err == nil {
		t.Fatal("broken config must be an error")
	}
}
