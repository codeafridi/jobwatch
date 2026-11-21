package job

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

var remoteOKURL = "https://remoteok.com/api"

func fetchRemoteOK(ctx context.Context, c *http.Client) ([]Job, error) {
	body, err := get(ctx, c, remoteOKURL)
	if err != nil {
		return nil, err
	}
	return parseRemoteOK(body)
}

func parseRemoteOK(body []byte) ([]Job, error) {
	// The first element of the array is a legal notice, not a posting.
	var raw []json.RawMessage
	if err := decodeJSON(body, &raw); err != nil {
		return nil, err
	}

	type posting struct {
		ID        string   `json:"id"`
		Position  string   `json:"position"`
		Company   string   `json:"company"`
		Location  string   `json:"location"`
		Tags      []string `json:"tags"`
		Date      string   `json:"date"`
		URL       string   `json:"url"`
		SalaryMin int      `json:"salary_min"`
		SalaryMax int      `json:"salary_max"`
	}

	jobs := make([]Job, 0, len(raw))
	for _, item := range raw {
		var p posting
		if err := json.Unmarshal(item, &p); err != nil || p.Position == "" {
			continue
		}
		posted, _ := time.Parse(time.RFC3339, p.Date)
		jobs = append(jobs, Job{
			ID:       NewID("remoteok", p.ID, p.URL),
			Source:   "remoteok",
			Title:    CleanTitle(p.Position),
			Company:  p.Company,
			Location: p.Location,
			Salary:   salaryRange(p.SalaryMin, p.SalaryMax),
			URL:      p.URL,
			Tags:     p.Tags,
			Posted:   posted.UTC(),
		})
	}
	return jobs, nil
}

func salaryRange(min, max int) string {
	switch {
	case min > 0 && max > 0:
		return fmt.Sprintf("$%dk–$%dk", min/1000, max/1000)
	case min > 0:
		return fmt.Sprintf("from $%dk", min/1000)
	case max > 0:
		return fmt.Sprintf("up to $%dk", max/1000)
	}
	return ""
}
