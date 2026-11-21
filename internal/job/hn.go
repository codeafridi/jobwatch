package job

import (
	"context"
	"fmt"
	"html"
	"net/http"
	"regexp"
	"strings"
	"time"
)

// Hacker News' monthly "Who is hiring?" thread is still the best single source
// of small-team infrastructure jobs, and it has no API of its own: the story is
// found through Algolia and read from the item tree.
var (
	hnSearchURL = "https://hn.algolia.com/api/v1/search_by_date?tags=story,author_whoishiring&hitsPerPage=1"
	hnItemURL   = "https://hn.algolia.com/api/v1/items/"
)

func fetchHN(ctx context.Context, c *http.Client) ([]Job, error) {
	searchBody, err := get(ctx, c, hnSearchURL)
	if err != nil {
		return nil, err
	}
	var search struct {
		Hits []struct {
			ObjectID string `json:"objectID"`
		} `json:"hits"`
	}
	if err := decodeJSON(searchBody, &search); err != nil {
		return nil, err
	}
	if len(search.Hits) == 0 {
		return nil, fmt.Errorf("no who-is-hiring story found")
	}

	body, err := get(ctx, c, hnItemURL+search.Hits[0].ObjectID)
	if err != nil {
		return nil, err
	}
	return parseHN(body)
}

type hnItem struct {
	ID        int64       `json:"id"`
	Title     string      `json:"title"`
	CreatedAt time.Time   `json:"created_at"`
	Children  []hnComment `json:"children"`
}

type hnComment struct {
	ID        int64     `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	Text      string    `json:"text"`
}

var (
	htmlTagRe = regexp.MustCompile(`<[^>]*>`)
	hrefRe    = regexp.MustCompile(`href="([^"]+)"`)
	urlRe     = regexp.MustCompile(`https?://[^\s"<>]+`)
	spaceRe   = regexp.MustCompile(`[ \t]+`)
)

func parseHN(body []byte) ([]Job, error) {
	var item hnItem
	if err := decodeJSON(body, &item); err != nil {
		return nil, err
	}

	jobs := make([]Job, 0, len(item.Children))
	for _, c := range item.Children {
		if strings.TrimSpace(c.Text) == "" {
			continue
		}
		raw := html.UnescapeString(c.Text)

		company, title, location, tags, ok := hnHeader(stripHTML(raw))
		if !ok {
			continue
		}
		url := hnURL(raw, c.ID)

		jobs = append(jobs, Job{
			ID:       NewID("hn", fmt.Sprint(c.ID), url),
			Source:   "hn",
			Title:    CleanTitle(title),
			Company:  strings.TrimSpace(company),
			Location: location,
			URL:      url,
			Tags:     tags,
			Posted:   c.CreatedAt.UTC(),
		})
	}
	return jobs, nil
}

// hnHeader reads the "Company | Role | Location | ..." first line that the
// thread's convention asks for. Posts without that shape are skipped: they are
// conversations, not job openings, and guessing turns them into noise.
func hnHeader(text string) (company, title, location string, tags []string, ok bool) {
	line := text
	if i := strings.IndexByte(line, '\n'); i >= 0 {
		line = line[:i]
	}
	parts := strings.Split(line, "|")
	if len(parts) < 2 {
		return "", "", "", nil, false
	}
	for i := range parts {
		parts[i] = strings.TrimSpace(parts[i])
	}
	if parts[0] == "" || parts[1] == "" {
		return "", "", "", nil, false
	}

	company, title = parts[0], parts[1]
	rest := parts[2:]

	// Some posters put the link where the role goes.
	if isURL(title) && len(rest) > 0 {
		title, rest = rest[0], rest[1:]
	}
	if len(title) > 90 || title == "" {
		return "", "", "", nil, false
	}

	for _, p := range rest {
		if looksRemote(p) && len(p) <= 40 {
			location = normalizeRemote(p)
			break
		}
	}
	if location == "" {
		for _, p := range rest {
			if looksRemote(p) {
				location = "Remote"
				break
			}
		}
	}
	if location == "" && len(rest) > 0 && len(rest[0]) <= 40 && !isURL(rest[0]) {
		location = rest[0]
	}

	for _, p := range rest {
		if p == "" || isURL(p) || len(p) > 40 || looksRemote(p) || p == location {
			continue
		}
		tags = append(tags, p)
		if len(tags) == 6 {
			break
		}
	}
	return company, title, location, tags, true
}

// normalizeRemote turns the thread's SHOUTED location headers into something
// worth printing: "REMOTE (EU)" becomes "Remote (EU)".
func normalizeRemote(s string) string {
	if strings.HasPrefix(strings.ToUpper(s), "REMOTE") {
		return "Remote" + s[len("REMOTE"):]
	}
	return s
}

func isURL(s string) bool {
	return strings.HasPrefix(s, "http://") || strings.HasPrefix(s, "https://")
}

func looksRemote(s string) bool {
	l := strings.ToLower(s)
	return strings.Contains(l, "remote") || strings.Contains(l, "anywhere") ||
		strings.Contains(l, "worldwide") || strings.Contains(l, "distributed")
}

func hnURL(raw string, id int64) string {
	if m := hrefRe.FindStringSubmatch(raw); len(m) == 2 {
		return strings.TrimSpace(m[1])
	}
	if m := urlRe.FindString(raw); m != "" {
		return m
	}
	return fmt.Sprintf("https://news.ycombinator.com/item?id=%d", id)
}

func stripHTML(s string) string {
	s = html.UnescapeString(s)
	s = htmlTagRe.ReplaceAllString(s, " ")
	// Entities can hide inside text nodes too ("&amp;nbsp;"), and NBSP is not
	// matched by the space class.
	s = html.UnescapeString(s)
	s = strings.ReplaceAll(s, "\u00a0", " ")
	s = spaceRe.ReplaceAllString(s, " ")
	s = strings.ReplaceAll(s, "\r", "")
	return strings.TrimSpace(s)
}
