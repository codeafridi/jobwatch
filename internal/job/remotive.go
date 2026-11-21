package job

import (
	"context"
	"net/http"
	"strings"
	"time"
)

var remotiveURL = "https://remotive.com/api/remote-jobs?category=software-dev"

type remotiveResponse struct {
	Jobs []struct {
		ID             jsonNumber `json:"id"`
		URL            string     `json:"url"`
		Title          string     `json:"title"`
		Company        string     `json:"company_name"`
		Category       string     `json:"category"`
		Tags           []string   `json:"tags"`
		JobType        string     `json:"job_type"`
		Publication    string     `json:"publication_date"`
		RequiredPlace  string     `json:"candidate_required_location"`
		Salary         string     `json:"salary"`
		CompanyLogoURL string     `json:"company_logo_url"`
	} `json:"jobs"`
}

func fetchRemotive(ctx context.Context, c *http.Client) ([]Job, error) {
	body, err := get(ctx, c, remotiveURL)
	if err != nil {
		return nil, err
	}
	return parseRemotive(body)
}

func parseRemotive(body []byte) ([]Job, error) {
	var resp remotiveResponse
	if err := decodeJSON(body, &resp); err != nil {
		return nil, err
	}

	jobs := make([]Job, 0, len(resp.Jobs))
	for _, r := range resp.Jobs {
		posted, _ := time.Parse("2006-01-02T15:04:05", r.Publication)
		tags := append([]string{}, r.Tags...)
		if r.Category != "" {
			tags = append(tags, r.Category)
		}
		jobs = append(jobs, Job{
			ID:       NewID("remotive", r.ID.String(), r.URL),
			Source:   "remotive",
			Title:    CleanTitle(r.Title),
			Company:  strings.TrimSpace(r.Company),
			Location: strings.TrimSpace(r.RequiredPlace),
			Salary:   strings.TrimSpace(r.Salary),
			URL:      r.URL,
			Tags:     tags,
			Posted:   posted.UTC(),
		})
	}
	return jobs, nil
}

// jsonNumber accepts both string and numeric ids: boards are inconsistent and
// this keeps the decoder from failing on one of them.
type jsonNumber string

func (n *jsonNumber) UnmarshalJSON(b []byte) error {
	s := strings.Trim(string(b), `"`)
	if s == "null" {
		s = ""
	}
	*n = jsonNumber(s)
	return nil
}

func (n jsonNumber) String() string { return string(n) }
