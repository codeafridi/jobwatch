package store

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestLoadMissingFile(t *testing.T) {
	s, err := Load(filepath.Join(t.TempDir(), "nope.json"))
	if err != nil {
		t.Fatalf("missing file must not error: %v", err)
	}
	if s == nil || s.Seen == nil {
		t.Fatal("state must be usable")
	}
	if !s.IsNew("anything") {
		t.Fatal("empty state must treat everything as new")
	}
}

func TestSaveLoadRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nested", "seen.json")
	now := time.Date(2026, 11, 21, 10, 0, 0, 0, time.UTC)

	s := &State{Seen: map[string]time.Time{}}
	s.Mark([]string{"a", "b"}, now)
	if err := s.Save(path); err != nil {
		t.Fatalf("save: %v", err)
	}

	got, err := Load(path)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if got.IsNew("a") || got.IsNew("b") {
		t.Fatal("marked ids must survive the round trip")
	}
	if !got.IsNew("c") {
		t.Fatal("unknown id must be new")
	}
	if !got.Seen["a"].Equal(now) {
		t.Fatalf("timestamp lost: %v", got.Seen["a"])
	}
}

func TestSaveIsAtomic(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "seen.json")

	first := &State{Seen: map[string]time.Time{"a": time.Now()}}
	if err := first.Save(path); err != nil {
		t.Fatal(err)
	}
	second := &State{Seen: map[string]time.Time{"a": time.Now(), "b": time.Now()}}
	if err := second.Save(path); err != nil {
		t.Fatal(err)
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		if e.Name() != "seen.json" {
			t.Fatalf("temporary file left behind: %s", e.Name())
		}
	}

	got, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Seen) != 2 {
		t.Fatalf("want 2 entries, got %d", len(got.Seen))
	}
}

func TestPrune(t *testing.T) {
	now := time.Now()
	s := &State{Seen: map[string]time.Time{
		"old": now.Add(-200 * 24 * time.Hour),
		"new": now.Add(-time.Hour),
	}}
	s.Prune(now, 120*24*time.Hour)

	if _, ok := s.Seen["old"]; ok {
		t.Fatal("old entry must be pruned")
	}
	if _, ok := s.Seen["new"]; !ok {
		t.Fatal("recent entry must stay")
	}
}
