package job

import (
	"context"
	"encoding/xml"
	"net/http"
	"strings"
	"time"
)

var wwrURL = "https://weworkremotely.com/categories/remote-programming-jobs.rss"

func fetchWWR(ctx context.Context, c *http.Client) ([]Job, error) {
	body, err := get(ctx, c, wwrURL)
	if err != nil {
		return nil, err
	}
	return parseWWR(body)
}

type wwrFeed struct {
	Channel struct {
		Items []struct {
			Title       string `xml:"title"`
			Region      string `xml:"region"`
			Category    string `xml:"category"`
			Link        string `xml:"link"`
			PubDate     string `xml:"pubDate"`
			Description string `xml:"description"`
		} `xml:"item"`
	} `xml:"channel"`
}

func parseWWR(body []byte) ([]Job, error) {
	var feed wwrFeed
	if err := xml.Unmarshal(body, &feed); err != nil {
		return nil, err
	}

	jobs := make([]Job, 0, len(feed.Channel.Items))
	for _, it := range feed.Channel.Items {
		company, title := SplitCompanyTitle(it.Title)
		if title == "" {
			continue
		}
		posted, _ := time.Parse(time.RFC1123Z, strings.TrimSpace(it.PubDate))
		tags := []string{}
		if it.Category != "" {
			tags = append(tags, it.Category)
		}
		jobs = append(jobs, Job{
			ID:       NewID("wwr", it.Link, it.Link),
			Source:   "wwr",
			Title:    CleanTitle(title),
			Company:  company,
			Location: strings.TrimSpace(it.Region),
			URL:      strings.TrimSpace(it.Link),
			Tags:     tags,
			Posted:   posted.UTC(),
		})
	}
	return jobs, nil
}
