package job

import (
	"strings"
	"testing"
)

func TestParseHN(t *testing.T) {
	jobs, err := parseHN(readFixture(t, "hn.json"))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(jobs) == 0 {
		t.Fatal("no postings parsed from the fixture")
	}

	first := jobs[0]
	if first.Company == "" || first.Title == "" {
		t.Fatalf("header not split: %+v", first)
	}
	if first.Title == "DevOps Engineer" && first.Company != "Simple Analytics" {
		t.Fatalf("unexpected company: %q", first.Company)
	}
	if !strings.HasPrefix(first.URL, "http") {
		t.Fatalf("url not extracted: %q", first.URL)
	}
	if first.Posted.IsZero() {
		t.Fatal("comment date not parsed")
	}
	if len(first.ID) != 16 {
		t.Fatalf("unexpected id %q", first.ID)
	}
	if len(first.Tags) == 0 {
		t.Fatal("header tags should keep the stack hints")
	}
}

func TestHNHeader(t *testing.T) {
	tests := []struct {
		name     string
		text     string
		company  string
		title    string
		location string
		ok       bool
		tag      string
	}{
		{
			name:     "standard header",
			text:     "Acme | Backend Engineer | REMOTE (EU) | Go, Kubernetes\nWe build things.",
			company:  "Acme",
			title:    "Backend Engineer",
			location: "Remote (EU)",
			ok:       true,
			tag:      "Go, Kubernetes",
		},
		{
			name:     "worldwide is remote",
			text:     "Acme | Platform Engineer | Worldwide",
			company:  "Acme",
			title:    "Platform Engineer",
			location: "Worldwide",
			ok:       true,
		},
		{
			name:     "city stays a city",
			text:     "Acme | SRE | Berlin, Germany",
			company:  "Acme",
			title:    "SRE",
			location: "Berlin, Germany",
			ok:       true,
		},
		{
			name:     "url in the role slot moves aside",
			text:     "LiveKit | https://livekit.io/ | Backend Engineer | Remote",
			company:  "LiveKit",
			title:    "Backend Engineer",
			location: "Remote",
			ok:       true,
		},
		{
			name: "prose without separators is skipped",
			text: "Location: India (currently travelling within India; IST) Remote",
			ok:   false,
		},
		{
			name: "overlong title is skipped",
			text: "Acme | " + strings.Repeat("word ", 30) + "| Remote",
			ok:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			company, title, location, tags, ok := hnHeader(tt.text)
			if ok != tt.ok {
				t.Fatalf("ok = %v, want %v (%q, %q, %q)", ok, tt.ok, company, title, location)
			}
			if !ok {
				return
			}
			if company != tt.company || title != tt.title || location != tt.location {
				t.Fatalf("got (%q, %q, %q), want (%q, %q, %q)",
					company, title, location, tt.company, tt.title, tt.location)
			}
			if tt.tag != "" {
				found := false
				for _, tag := range tags {
					if tag == tt.tag {
						found = true
					}
				}
				if !found {
					t.Fatalf("tag %q missing from %v", tt.tag, tags)
				}
			}
		})
	}
}

func TestHNURLExtraction(t *testing.T) {
	fallback := hnURL("no link here", 42)
	if fallback != "https://news.ycombinator.com/item?id=42" {
		t.Fatalf("fallback = %q", fallback)
	}
	if got := hnURL(`<a href="https://example.com/jobs">x</a>`, 42); got != "https://example.com/jobs" {
		t.Fatalf("href = %q", got)
	}
	if got := hnURL("see https://example.com/apply for details", 42); got != "https://example.com/apply" {
		t.Fatalf("bare url = %q", got)
	}
}

func TestStripHTML(t *testing.T) {
	in := "<p>Hello&nbsp;<strong>world</strong></p><p>second&nbsp;line</p>"
	got := stripHTML(in)
	if strings.ContainsAny(got, "<>") {
		t.Fatalf("tags left in %q", got)
	}
	if !strings.Contains(got, "second line") {
		t.Fatalf("text lost: %q", got)
	}
}
