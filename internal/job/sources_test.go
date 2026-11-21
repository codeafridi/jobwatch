package job

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func readFixture(t *testing.T, name string) []byte {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("..", "..", "testdata", name))
	if err != nil {
		t.Fatalf("read fixture %s: %v", name, err)
	}
	return data
}

func TestParseRemotive(t *testing.T) {
	jobs, err := parseRemotive(readFixture(t, "remotive.json"))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(jobs) != 3 {
		t.Fatalf("want 3 jobs, got %d", len(jobs))
	}
	first := jobs[0]
	if first.Title == "" || first.Company == "" || first.URL == "" {
		t.Fatalf("missing fields: %+v", first)
	}
	if first.Source != "remotive" {
		t.Fatalf("source = %q", first.Source)
	}
	if first.Posted.IsZero() {
		t.Fatal("publication date not parsed")
	}
	if len(first.Tags) == 0 {
		t.Fatal("tags not parsed")
	}
	if first.ID == "" || len(first.ID) != 16 {
		t.Fatalf("unexpected id %q", first.ID)
	}
}

func TestParseRemoteOKSkipsLegalNotice(t *testing.T) {
	jobs, err := parseRemoteOK(readFixture(t, "remoteok.json"))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(jobs) != 3 {
		t.Fatalf("want 3 jobs, got %d (the legal notice must be skipped)", len(jobs))
	}
	for _, j := range jobs {
		if j.Title == "" {
			t.Fatalf("empty title in %+v", j)
		}
		if j.Posted.IsZero() {
			t.Fatalf("date not parsed: %+v", j)
		}
	}
}

func TestParseWWR(t *testing.T) {
	jobs, err := parseWWR(readFixture(t, "wwr.xml"))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(jobs) != 3 {
		t.Fatalf("want 3 jobs, got %d", len(jobs))
	}
	for _, j := range jobs {
		if j.Company == "" || j.Title == "" {
			t.Fatalf("company/title not split: %+v", j)
		}
		if strings.Contains(j.Title, ":") && j.Company == "" {
			t.Fatalf("title still has company prefix: %+v", j)
		}
	}
}

func TestParseArbeitnowKeepsOnlyRemote(t *testing.T) {
	jobs, err := parseArbeitnow(readFixture(t, "arbeitnow.json"))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(jobs) == 0 {
		t.Fatal("no remote jobs parsed")
	}
	for _, j := range jobs {
		if j.Posted.IsZero() {
			t.Fatalf("created_at not parsed: %+v", j)
		}
	}
}

func TestParseJobicy(t *testing.T) {
	jobs, err := parseJobicy(readFixture(t, "jobicy.json"))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(jobs) != 3 {
		t.Fatalf("want 3 jobs, got %d", len(jobs))
	}
	for _, j := range jobs {
		if j.Posted.IsZero() {
			t.Fatalf("pubDate not parsed: %+v", j)
		}
		if j.Title == "" || j.Company == "" {
			t.Fatalf("missing title/company: %+v", j)
		}
	}
}

// TestFetchAllUsesFixtures points every source at a local server and checks
// that a broken source is reported without losing the others.
func TestFetchAllUsesFixtures(t *testing.T) {
	hnBody := readFixture(t, "hn.json")
	var story struct {
		ID int64 `json:"id"`
	}
	if err := json.Unmarshal(hnBody, &story); err != nil {
		t.Fatal(err)
	}

	files := map[string]string{
		"/remotive":  "remotive.json",
		"/remoteok":  "remoteok.json",
		"/wwr":       "wwr.xml",
		"/arbeitnow": "arbeitnow.json",
		"/jobicy":    "jobicy.json",
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/hn-search" {
			_, _ = fmt.Fprintf(w, `{"hits":[{"objectID":"%d"}]}`, story.ID)
			return
		}
		if r.URL.Path == fmt.Sprintf("/hn/%d", story.ID) {
			_, _ = w.Write(hnBody)
			return
		}
		name, ok := files[r.URL.Path]
		if !ok {
			http.NotFound(w, r)
			return
		}
		_, _ = w.Write(readFixture(t, name))
	}))
	defer srv.Close()

	old := []string{hnSearchURL, hnItemURL, remotiveURL, remoteOKURL, wwrURL, arbeitnowURL, jobicyURL}
	hnSearchURL = srv.URL + "/hn-search"
	hnItemURL = srv.URL + "/hn/"
	remotiveURL = srv.URL + "/remotive"
	remoteOKURL = srv.URL + "/remoteok"
	wwrURL = srv.URL + "/wwr"
	arbeitnowURL = srv.URL + "/arbeitnow"
	jobicyURL = srv.URL + "/jobicy"
	t.Cleanup(func() {
		hnSearchURL, hnItemURL = old[0], old[1]
		remotiveURL, remoteOKURL, wwrURL = old[2], old[3], old[4]
		arbeitnowURL, jobicyURL = old[5], old[6]
	})

	client := &http.Client{}
	jobs, errs := FetchAll(t.Context(), client, nil)
	if len(errs) != 0 {
		t.Fatalf("unexpected source errors: %v", errs)
	}
	if len(jobs) < 13 {
		t.Fatalf("want at least 13 jobs from 6 fixtures, got %d", len(jobs))
	}

	// Drop one source and make another fail: the rest must survive.
	jobicyURL = srv.URL + "/nope"
	jobs, errs = FetchAll(t.Context(), client, []string{"remotive", "jobicy"})
	if _, ok := errs["jobicy"]; !ok {
		t.Fatalf("expected a jobicy error, got %v", errs)
	}
	if len(jobs) != 3 {
		t.Fatalf("want the 3 remotive jobs despite the failure, got %d", len(jobs))
	}

	_, errs = FetchAll(t.Context(), client, []string{"nope"})
	if _, ok := errs["sources"]; !ok {
		t.Fatalf("unknown source must be reported, got %v", errs)
	}
}
