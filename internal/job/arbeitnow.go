package job

import (
	"context"
	"net/http"
	"strings"
	"time"
)

var arbeitnowURL = "https://www.arbeitnow.com/api/job-board-api"

func fetchArbeitnow(ctx context.Context, c *http.Client) ([]Job, error) {
	body, err := get(ctx, c, arbeitnowURL)
	if err != nil {
		return nil, err
	}
	return parseArbeitnow(body)
}

func parseArbeitnow(body []byte) ([]Job, error) {
	var resp struct {
		Data []struct {
			Slug      string   `json:"slug"`
			Company   string   `json:"company_name"`
			Title     string   `json:"title"`
			Remote    bool     `json:"remote"`
			URL       string   `json:"url"`
			Tags      []string `json:"tags"`
			JobTypes  []string `json:"job_types"`
			Location  string   `json:"location"`
			CreatedAt int64    `json:"created_at"`
		} `json:"data"`
	}
	if err := decodeJSON(body, &resp); err != nil {
		return nil, err
	}

	jobs := make([]Job, 0, len(resp.Data))
	for _, r := range resp.Data {
		if !r.Remote {
			continue
		}
		posted := time.Unix(r.CreatedAt, 0).UTC()
		tags := append(append([]string{}, r.Tags...), r.JobTypes...)
		jobs = append(jobs, Job{
			ID:       NewID("arbeitnow", r.Slug, r.URL),
			Source:   "arbeitnow",
			Title:    CleanTitle(r.Title),
			Company:  strings.TrimSpace(r.Company),
			Location: strings.TrimSpace(r.Location),
			URL:      r.URL,
			Tags:     tags,
			Posted:   posted,
		})
	}
	return jobs, nil
}
