package job

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
)

// UserAgent identifies the tool to the job boards. Several of them ask for a
// descriptive agent and a link back, see the README's sources section.
const UserAgent = "jobwatch/0.1 (+https://github.com/codeafridi/jobwatch)"

const maxBodyBytes = 16 << 20

// Fetcher pulls postings from one board.
type Fetcher struct {
	Name  string
	Fetch func(ctx context.Context, c *http.Client) ([]Job, error)
}

// All returns the known sources in the order they are reported.
func All() []Fetcher {
	return []Fetcher{
		{"hn", fetchHN},
		{"remotive", fetchRemotive},
		{"remoteok", fetchRemoteOK},
		{"wwr", fetchWWR},
		{"arbeitnow", fetchArbeitnow},
		{"jobicy", fetchJobicy},
	}
}

// Names lists the available source names.
func Names() []string {
	names := make([]string, 0, len(All()))
	for _, f := range All() {
		names = append(names, f.Name)
	}
	return names
}

// FetchAll fetches the selected sources concurrently. A failing board does not
// sink the digest: its error is returned alongside whatever the others found.
func FetchAll(ctx context.Context, c *http.Client, names []string) ([]Job, map[string]error) {
	selected := map[string]bool{}
	for _, n := range names {
		selected[n] = true
	}

	var (
		mu    sync.Mutex
		wg    sync.WaitGroup
		jobs  []Job
		errs  = map[string]error{}
		found int
	)
	for _, f := range All() {
		if len(selected) > 0 && !selected[f.Name] {
			continue
		}
		found++
		wg.Add(1)
		go func(f Fetcher) {
			defer wg.Done()
			got, err := f.Fetch(ctx, c)
			mu.Lock()
			defer mu.Unlock()
			if err != nil {
				errs[f.Name] = err
				return
			}
			for i := range got {
				got[i].Title = FixMojibake(got[i].Title)
				got[i].Company = FixMojibake(got[i].Company)
			}
			jobs = append(jobs, got...)
		}(f)
	}
	wg.Wait()

	if found == 0 {
		errs["sources"] = fmt.Errorf("no matching sources, available: %v", Names())
	}
	return jobs, errs
}

// get reads a whole response body, refusing anything unreasonable.
func get(ctx context.Context, c *http.Client, url string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", UserAgent)
	req.Header.Set("Accept", "application/json, application/xml, text/xml;q=0.9, */*;q=0.8")

	resp, err := c.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%s: %s", url, resp.Status)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxBodyBytes))
	if err != nil {
		return nil, err
	}
	if len(body) == int(maxBodyBytes) {
		return nil, fmt.Errorf("%s: response larger than %d bytes", url, maxBodyBytes)
	}
	return body, nil
}

func decodeJSON(body []byte, v any) error {
	if err := json.Unmarshal(body, v); err != nil {
		return fmt.Errorf("decode json: %w", err)
	}
	return nil
}
