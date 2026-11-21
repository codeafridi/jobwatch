package job

import (
	"context"
	"net/http"
	"strconv"
	"strings"
	"time"
)

var jobicyURL = "https://jobicy.com/api/v2/remote-jobs?count=50"

func fetchJobicy(ctx context.Context, c *http.Client) ([]Job, error) {
	body, err := get(ctx, c, jobicyURL)
	if err != nil {
		return nil, err
	}
	return parseJobicy(body)
}

func parseJobicy(body []byte) ([]Job, error) {
	var resp struct {
		Jobs []struct {
			ID        jsonNumber `json:"id"`
			URL       string     `json:"url"`
			Title     string     `json:"jobTitle"`
			Company   string     `json:"companyName"`
			Industry  []string   `json:"jobIndustry"`
			JobType   []string   `json:"jobType"`
			Geo       string     `json:"jobGeo"`
			Level     string     `json:"jobLevel"`
			PubDate   string     `json:"pubDate"`
			SalaryMin *float64   `json:"annualSalaryMin"`
			SalaryMax *float64   `json:"annualSalaryMax"`
			SalaryCur string     `json:"salaryCurrency"`
		} `json:"jobs"`
	}
	if err := decodeJSON(body, &resp); err != nil {
		return nil, err
	}

	jobs := make([]Job, 0, len(resp.Jobs))
	for _, r := range resp.Jobs {
		posted, _ := time.Parse(time.RFC3339, r.PubDate)
		tags := append(append([]string{}, r.Industry...), r.JobType...)
		if r.Level != "" && !strings.EqualFold(r.Level, "any") {
			tags = append(tags, r.Level)
		}
		jobs = append(jobs, Job{
			ID:       NewID("jobicy", r.ID.String(), r.URL),
			Source:   "jobicy",
			Title:    CleanTitle(r.Title),
			Company:  strings.TrimSpace(r.Company),
			Location: strings.TrimSpace(r.Geo),
			Salary:   annualRange(r.SalaryMin, r.SalaryMax, r.SalaryCur),
			URL:      r.URL,
			Tags:     tags,
			Posted:   posted.UTC(),
		})
	}
	return jobs, nil
}

func annualRange(min, max *float64, currency string) string {
	if min == nil && max == nil {
		return ""
	}
	cur := strings.ToUpper(currency)
	format := func(v float64) string {
		s := strconv.FormatFloat(v/1000, 'f', 0, 64)
		return s + "k"
	}
	switch {
	case min != nil && max != nil:
		return cur + " " + format(*min) + "–" + format(*max)
	case min != nil:
		return cur + " from " + format(*min)
	case max != nil:
		return cur + " up to " + format(*max)
	}
	return ""
}
