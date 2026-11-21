package job

import "testing"

func TestNewIDIsStable(t *testing.T) {
	a := NewID("remotive", "42", "https://example.com/1")
	b := NewID("remotive", "42", "https://example.com/2")
	if a != b {
		t.Fatal("same source+key must produce the same id")
	}
	c := NewID("remoteok", "42", "https://example.com/1")
	if a == c {
		t.Fatal("different sources must not collide")
	}
	if len(a) != 16 {
		t.Fatalf("want 16 hex chars, got %d (%q)", len(a), a)
	}
}

func TestNewIDFallsBackToURL(t *testing.T) {
	a := NewID("wwr", "", "https://example.com/job")
	b := NewID("wwr", "", "https://example.com/job")
	if a != b || a == "" {
		t.Fatalf("url fallback ids differ: %q %q", a, b)
	}
}

func TestDedupeKeyIgnoresCaseAndSpace(t *testing.T) {
	a := Job{Company: "Acme  Inc", Title: "Backend  Engineer"}
	b := Job{Company: "acme inc", Title: "backend engineer"}
	if a.DedupeKey() != b.DedupeKey() {
		t.Fatalf("keys differ: %q vs %q", a.DedupeKey(), b.DedupeKey())
	}
	c := Job{Company: "Acme Inc", Title: "Frontend Engineer"}
	if a.DedupeKey() == c.DedupeKey() {
		t.Fatal("different titles must not share a key")
	}
}

func TestCleanTitle(t *testing.T) {
	tests := map[string]string{
		"Backend  Engineer":            "Backend Engineer",
		" Backend Engineer ":           "Backend Engineer",
		"Backend Engineer (Remote)":    "Backend Engineer",
		"Backend Engineer - Remote":    "Backend Engineer",
		"Senior Go Engineer [Remote]":  "Senior Go Engineer",
		"Engineer (Remote) - Contract": "Engineer (Remote) - Contract",
		"1227 | Back-end Developer":    "Back-end Developer",
	}
	for in, want := range tests {
		if got := CleanTitle(in); got != want {
			t.Errorf("CleanTitle(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestFixMojibake(t *testing.T) {
	tests := map[string]string{
		"Virtual IOP Clinician \u00e2\u20ac\u201c Florida": "Virtual IOP Clinician – Florida",
		"San Jos\u00c3\u00a9":                              "San José",
		"plain ASCII":                                      "plain ASCII",
		"already correct – dash":                           "already correct – dash",
	}
	for in, want := range tests {
		if got := FixMojibake(in); got != want {
			t.Errorf("FixMojibake(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestSplitCompanyTitle(t *testing.T) {
	tests := []struct{ in, company, title string }{
		{"Acme: Backend Engineer", "Acme", "Backend Engineer"},
		{"A.Team: Senior Developer ($90/hr)", "A.Team", "Senior Developer ($90/hr)"},
		{"Backend Engineer", "", "Backend Engineer"},
		{": broken", "", ": broken"},
	}
	for _, tt := range tests {
		company, title := SplitCompanyTitle(tt.in)
		if company != tt.company || title != tt.title {
			t.Errorf("SplitCompanyTitle(%q) = (%q, %q), want (%q, %q)",
				tt.in, company, title, tt.company, tt.title)
		}
	}
}
