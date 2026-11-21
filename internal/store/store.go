// Package store remembers which postings were already shown.
package store

import (
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"time"
)

// State is the on-disk set of posting ids already reported.
type State struct {
	Seen map[string]time.Time `json:"seen"`
}

// DefaultPath follows XDG: $XDG_STATE_HOME/jobwatch/seen.json.
func DefaultPath() (string, error) {
	if dir := os.Getenv("XDG_STATE_HOME"); dir != "" {
		return filepath.Join(dir, "jobwatch", "seen.json"), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".local", "state", "jobwatch", "seen.json"), nil
}

// Load reads the state file. A missing file is an empty state, not an error.
func Load(path string) (*State, error) {
	s := &State{Seen: map[string]time.Time{}}
	data, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return s, nil
	}
	if err != nil {
		return nil, err
	}
	if len(data) == 0 {
		return s, nil
	}
	if err := json.Unmarshal(data, s); err != nil {
		return nil, err
	}
	if s.Seen == nil {
		s.Seen = map[string]time.Time{}
	}
	return s, nil
}

// Save writes the state atomically so an interrupted run cannot truncate it.
func (s *State) Save(path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), ".seen-*.json")
	if err != nil {
		return err
	}
	defer func() { _ = os.Remove(tmp.Name()) }()
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmp.Name(), path)
}

// IsNew reports whether the posting has not been reported before.
func (s *State) IsNew(id string) bool {
	_, ok := s.Seen[id]
	return !ok
}

// Mark records the postings as reported.
func (s *State) Mark(ids []string, now time.Time) {
	for _, id := range ids {
		s.Seen[id] = now
	}
}

// Prune drops records older than maxAge so the file does not grow forever.
func (s *State) Prune(now time.Time, maxAge time.Duration) {
	for id, seen := range s.Seen {
		if now.Sub(seen) > maxAge {
			delete(s.Seen, id)
		}
	}
}
