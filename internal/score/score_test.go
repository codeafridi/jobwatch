package score

import (
	"testing"
	"time"

	"github.com/codeafridi/jobwatch/internal/job"
)

func TestApply(t *testing.T) {
	now := time.Date(2026, 11, 21, 12, 0, 0, 0, time.UTC)

	profile := Profile{
		Include:      []string{"go", "backend", "platform"},
		Exclude:      []string{"senior", "lead"},
		Locations:    []string{"worldwide", "india", "anywhere"},
		SkipExcluded: true,
	}

	tests := []struct {
		name     string
		job      job.Job
		wantKeep bool
		wantMin  int
		wantMax  int
	}{
		{
			name:     "title match, open location, fresh",
			job:      job.Job{Title: "Backend Engineer", Location: "Worldwide", Posted: now.Add(-time.Hour)},
			wantKeep: true,
			wantMin:  48,
			wantMax:  48,
		},
		{
			name:     "two title terms stack",
			job:      job.Job{Title: "Go Backend Engineer", Location: "Worldwide", Posted: now.Add(-time.Hour)},
			wantKeep: true,
			wantMin:  68,
			wantMax:  68,
		},
		{
			name:     "go must not match django, and no signal means no show",
			job:      job.Job{Title: "Django Developer", Location: "Worldwide", Posted: now.Add(-time.Hour)},
			wantKeep: false,
		},
		{
			name:     "tag match alone is enough",
			job:      job.Job{Title: "Software Engineer", Tags: []string{"platform"}, Location: "Worldwide", Posted: now.Add(-time.Hour)},
			wantKeep: true,
			wantMin:  34,
			wantMax:  34,
		},
		{
			name:     "excluded role is dropped",
			job:      job.Job{Title: "Senior Backend Engineer", Location: "Worldwide", Posted: now},
			wantKeep: false,
		},
		{
			name:     "leading does not match lead, and has no signal",
			job:      job.Job{Title: "Leading Hand", Location: "Worldwide", Posted: now},
			wantKeep: false,
		},
		{
			name:     "non-technical role with no signal is dropped",
			job:      job.Job{Title: "Compliance Officer/MLRO", Location: "Asia", Posted: now},
			wantKeep: false,
		},
		{
			name:     "restricted location is penalised",
			job:      job.Job{Title: "Backend Engineer", Location: "United States", Posted: now.Add(-40 * 24 * time.Hour)},
			wantKeep: true,
			wantMin:  0,
			wantMax:  0,
		},
		{
			name:     "unknown location and date still score",
			job:      job.Job{Title: "Platform Engineer"},
			wantKeep: true,
			wantMin:  24,
			wantMax:  24,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := tt.job
			keep := Apply(&p, profile, now)
			if keep != tt.wantKeep {
				t.Fatalf("keep = %v, want %v (score %d)", keep, tt.wantKeep, p.Score)
			}
			if !keep {
				return
			}
			if p.Score < tt.wantMin || p.Score > tt.wantMax {
				t.Fatalf("score = %d, want %d..%d (reasons %v)", p.Score, tt.wantMin, tt.wantMax, p.Reasons)
			}
		})
	}
}

func TestApplyCPlusPlus(t *testing.T) {
	now := time.Now()
	p := job.Job{Title: "C++ Engineer", Location: "Anywhere", Posted: now}
	profile := Profile{Include: []string{"c++"}, Locations: []string{"anywhere"}, SkipExcluded: true}
	if !Apply(&p, profile, now) {
		t.Fatal("c++ posting must be kept")
	}
	if p.Score < 40 {
		t.Fatalf("c++ should match its title, score = %d", p.Score)
	}
}

func TestApplyNoSkipExcluded(t *testing.T) {
	now := time.Now()
	p := job.Job{Title: "Senior Backend Engineer", Location: "Worldwide", Posted: now}
	profile := Profile{
		Include:      []string{"backend"},
		Exclude:      []string{"senior"},
		Locations:    []string{"worldwide"},
		SkipExcluded: false,
	}
	if !Apply(&p, profile, now) {
		t.Fatal("with SkipExcluded off the posting should be kept")
	}
	if p.Score >= 48 {
		t.Fatalf("exclusion penalty missing, score = %d", p.Score)
	}
}

func TestScoreIsClamped(t *testing.T) {
	now := time.Now()
	p := job.Job{Title: "Go Backend Platform Engineer", Location: "Worldwide", Posted: now}
	profile := Profile{
		Include:   []string{"go", "backend", "platform"},
		Locations: []string{"worldwide"},
	}
	Apply(&p, profile, now)
	if p.Score > 100 || p.Score < 0 {
		t.Fatalf("score out of range: %d", p.Score)
	}
}
