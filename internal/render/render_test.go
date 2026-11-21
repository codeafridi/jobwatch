package render

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/codeafridi/jobwatch/internal/job"
)

func fixture() ([]job.Job, Options) {
	now := time.Date(2026, 11, 21, 12, 0, 0, 0, time.UTC)
	jobs := []job.Job{
		{
			ID: "1", Source: "remotive", Title: "Backend Engineer (Go)",
			Company: "Acme", Location: "Worldwide", Salary: "$90k–$120k",
			URL: "https://example.com/1", Posted: now.Add(-2 * time.Hour), Score: 92,
			Reasons: []string{"title: go", "fresh"},
		},
		{
			ID: "2", Source: "wwr", Title: "Platform Engineer",
			Company: "Northflank", Location: "EU",
			URL: "https://example.com/2", Posted: now.Add(-3 * 24 * time.Hour), Score: 41,
		},
	}
	opts := Options{Now: now, Width: 80, Total: 240, Sources: []string{"remotive", "wwr"}}
	return jobs, opts
}

func TestANSIPlain(t *testing.T) {
	jobs, opts := fixture()
	var b bytes.Buffer
	if err := ANSI(&b, jobs, opts); err != nil {
		t.Fatal(err)
	}

	want := "  jobwatch  Sat 21 Nov 2026\n" +
		"  " + strings.Repeat("─", 64) + "\n\n" +
		"   92  Backend Engineer (Go)\n" +
		"      Acme · Worldwide · $90k–$120k · Remotive · 2h\n" +
		"   41  Platform Engineer\n" +
		"      Northflank · EU · We Work Remotely · 3d\n" +
		"\n" +
		"  2 new of 240 fetched · Remotive · We Work Remotely\n"

	if b.String() != want {
		t.Fatalf("unexpected output:\n--- got ---\n%s\n--- want ---\n%s", b.String(), want)
	}
}

func TestANSIExplain(t *testing.T) {
	jobs, opts := fixture()
	opts.Explain = true
	var b bytes.Buffer
	if err := ANSI(&b, jobs, opts); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(b.String(), "title: go · fresh") {
		t.Fatalf("reasons missing:\n%s", b.String())
	}
}

func TestANSIEmpty(t *testing.T) {
	_, opts := fixture()
	var b bytes.Buffer
	if err := ANSI(&b, nil, opts); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(b.String(), "nothing new above your filter") {
		t.Fatalf("empty state must say so:\n%s", b.String())
	}
}

func TestANSITruncatesLongTitles(t *testing.T) {
	now := time.Now()
	jobs := []job.Job{{
		Title: strings.Repeat("very long title ", 10), Score: 50,
		Source: "wwr", URL: "https://example.com/x",
	}}
	opts := Options{Now: now, Width: 40}
	var b bytes.Buffer
	if err := ANSI(&b, jobs, opts); err != nil {
		t.Fatal(err)
	}
	for _, line := range strings.Split(b.String(), "\n") {
		if len([]rune(line)) > 40 {
			t.Fatalf("line wider than terminal: %q", line)
		}
	}
	if !strings.Contains(b.String(), "…") {
		t.Fatal("long title must be elided")
	}
}

func TestMarkdown(t *testing.T) {
	jobs, opts := fixture()
	var b bytes.Buffer
	if err := Markdown(&b, jobs, opts); err != nil {
		t.Fatal(err)
	}

	want := "**jobwatch** · 21 Nov 2026\n\n" +
		"- **92** [Backend Engineer (Go)](https://example.com/1) — Acme\n" +
		"  Worldwide · $90k–$120k · Remotive · 2h\n" +
		"- **41** [Platform Engineer](https://example.com/2) — Northflank\n" +
		"  EU · We Work Remotely · 3d\n" +
		"\n" +
		"_2 new of 240 fetched · Remotive · We Work Remotely_\n"

	if b.String() != want {
		t.Fatalf("unexpected markdown:\n--- got ---\n%s\n--- want ---\n%s", b.String(), want)
	}
}

func TestJSON(t *testing.T) {
	jobs, _ := fixture()
	var b bytes.Buffer
	if err := JSON(&b, jobs); err != nil {
		t.Fatal(err)
	}
	var got []job.Job
	if err := json.Unmarshal(b.Bytes(), &got); err != nil {
		t.Fatalf("output is not json: %v", err)
	}
	if len(got) != 2 || got[0].Score != 92 || got[1].Title != "Platform Engineer" {
		t.Fatalf("unexpected json payload: %+v", got)
	}
}

func TestErrorsAreReported(t *testing.T) {
	_, opts := fixture()
	opts.Errors = map[string]error{"jobicy": errTest}
	var b bytes.Buffer
	if err := ANSI(&b, nil, opts); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(b.String(), "jobicy") || !strings.Contains(b.String(), "timeout") {
		t.Fatalf("source error missing:\n%s", b.String())
	}
}

type testErr struct{}

func (testErr) Error() string { return "timeout" }

var errTest = testErr{}
