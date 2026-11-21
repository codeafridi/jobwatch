package main

import (
	"bytes"
	"strings"
	"testing"

	"github.com/codeafridi/jobwatch/internal/job"
)

func runCLI(t *testing.T, args ...string) (string, error) {
	t.Helper()
	var out, errOut bytes.Buffer
	err := run(args, &out, &errOut)
	return out.String(), err
}

func TestVersionAndSources(t *testing.T) {
	out, err := runCLI(t, "--version")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(out, "jobwatch ") {
		t.Fatalf("unexpected version output %q", out)
	}

	out, err = runCLI(t, "--list-sources")
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"hn", "remotive", "remoteok", "wwr", "arbeitnow", "jobicy"} {
		if !strings.Contains(out, name) {
			t.Fatalf("source %q missing from %q", name, out)
		}
	}
}

func TestUnknownFormatIsRejectedBeforeFetching(t *testing.T) {
	_, err := runCLI(t, "--format", "yaml", "--sources", "nope")
	if err == nil {
		t.Fatal("unknown format must error")
	}
}

func TestDedupeKeepsBestScore(t *testing.T) {
	jobs := []job.Job{
		{Company: "Acme", Title: "Backend Engineer", Score: 30, Source: "wwr"},
		{Company: "acme", Title: "backend  engineer", Score: 70, Source: "remotive"},
		{Company: "Acme", Title: "Platform Engineer", Score: 40, Source: "wwr"},
	}
	got := dedupe(jobs)
	if len(got) != 2 {
		t.Fatalf("want 2 unique postings, got %d", len(got))
	}
	if got[0].Score != 70 || got[0].Source != "remotive" {
		t.Fatalf("dedupe kept the wrong posting: %+v", got[0])
	}
}

func TestSplitList(t *testing.T) {
	got := splitList(" remotive, ,jobicy ,")
	want := []string{"remotive", "jobicy"}
	if len(got) != len(want) || got[0] != want[0] || got[1] != want[1] {
		t.Fatalf("splitList = %v, want %v", got, want)
	}
}

func TestColorDisabledForPipes(t *testing.T) {
	t.Setenv("NO_COLOR", "")
	t.Setenv("TERM", "xterm-256color")
	var b bytes.Buffer
	if colorEnabled(false, &b) {
		t.Fatal("a pipe must not get colour")
	}
	if colorEnabled(true, &b) {
		t.Fatal("--no-color must win")
	}
	t.Setenv("NO_COLOR", "1")
	if colorEnabled(false, &b) {
		t.Fatal("NO_COLOR must win")
	}
}
