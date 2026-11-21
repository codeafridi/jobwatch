// Package config loads the reader's profile.
package config

import (
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/codeafridi/jobwatch/internal/job"
)

// Config is the filter profile. Every field has a usable default, so the tool
// works with no config file at all.
type Config struct {
	Include      []string `json:"include"`
	Exclude      []string `json:"exclude"`
	Locations    []string `json:"locations"`
	Sources      []string `json:"sources"`
	MinScore     int      `json:"min_score"`
	SkipExcluded bool     `json:"skip_excluded"`
}

// Default is a backend/infrastructure profile that can be seen from India.
// It is deliberately conservative: better to miss a sales role than to read
// forty of them every morning.
func Default() Config {
	return Config{
		Include: []string{
			"go", "golang", "backend", "back-end", "platform", "infrastructure",
			"kubernetes", "docker", "devops", "sre", "rust", "c++", "distributed",
			"systems", "api", "database", "linux",
		},
		Exclude: []string{
			"senior", "staff", "principal", "lead", "manager", "director",
			"head of", "vp", "vice president", "chief", "cto", "recruiter",
			"sales", "account executive",
			"customer support", "solutions architect", "intern", "graduate",
		},
		Locations: []string{
			"worldwide", "anywhere", "global", "remote", "india", "asia",
			"apac", "emea", "async",
		},
		Sources:      job.Names(),
		MinScore:     20,
		SkipExcluded: true,
	}
}

// DefaultPath follows XDG: $XDG_CONFIG_HOME/jobwatch/config.json.
func DefaultPath() (string, error) {
	if dir := os.Getenv("XDG_CONFIG_HOME"); dir != "" {
		return filepath.Join(dir, "jobwatch", "config.json"), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".config", "jobwatch", "config.json"), nil
}

// Load starts from Default and applies the file on top. A missing file is not
// an error: the tool is meant to run out of the box.
func Load(path string) (Config, error) {
	cfg := Default()
	data, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return cfg, nil
	}
	if err != nil {
		return cfg, err
	}
	if err := json.Unmarshal(data, &cfg); err != nil {
		return cfg, err
	}
	return cfg, nil
}
