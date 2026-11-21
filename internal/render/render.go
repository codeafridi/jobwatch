// Package render prints the digest. Plain text first: it has to look right in a
// terminal, in a markdown note and in a chat message.
package render

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/codeafridi/jobwatch/internal/job"
)

// Options controls the terminal layout.
type Options struct {
	Color   bool
	Width   int
	Explain bool
	Now     time.Time
	Total   int // postings fetched before filtering
	Errors  map[string]error
	Sources []string
}

const (
	reset   = "\x1b[0m"
	bold    = "\x1b[1m"
	dim     = "\x1b[2m"
	amber   = "\x1b[38;5;214m"
	gray    = "\x1b[38;5;245m"
	faint   = "\x1b[38;5;240m"
	scarlet = "\x1b[38;5;203m"
)

func paint(on bool, code, s string) string {
	if !on {
		return s
	}
	return code + s + reset
}

// ANSI prints the digest as a two-line-per-posting list.
func ANSI(w io.Writer, jobs []job.Job, opts Options) error {
	width := opts.Width
	if width < 40 {
		width = 80
	}

	var b strings.Builder

	head := paint(opts.Color, bold, "jobwatch")
	when := opts.Now.Format("Mon 2 Jan 2006")
	if opts.Color {
		when = paint(opts.Color, dim, when)
	}
	fmt.Fprintf(&b, "  %s  %s\n", head, when)
	rule := strings.Repeat("─", min(width-4, 64))
	b.WriteString("  " + paint(opts.Color, faint, rule) + "\n\n")

	if len(jobs) == 0 {
		b.WriteString("  " + paint(opts.Color, dim, "nothing new above your filter") + "\n")
	} else {
		for _, j := range jobs {
			line := fmt.Sprintf("  %s  %s\n", scoreBadge(j.Score, opts.Color), titleText(j, opts.Color, width))
			b.WriteString(line)

			var meta []string
			if j.Company != "" {
				meta = append(meta, j.Company)
			}
			if j.Location != "" {
				meta = append(meta, j.Location)
			}
			if j.Salary != "" {
				meta = append(meta, j.Salary)
			}
			meta = append(meta, sourceName(j.Source), age(j.Posted, opts.Now))
			b.WriteString("      " + paint(opts.Color, gray, truncate(strings.Join(meta, " · "), width-8)) + "\n")

			if opts.Explain && len(j.Reasons) > 0 {
				b.WriteString("      " + paint(opts.Color, faint, strings.Join(j.Reasons, " · ")) + "\n")
			}
		}
	}

	b.WriteString("\n")
	fmt.Fprintf(&b, "  %s\n", paint(opts.Color, faint, summary(jobs, opts)))

	if len(opts.Errors) > 0 {
		b.WriteString("\n")
		for name, err := range opts.Errors {
			fmt.Fprintf(&b, "  %s %s: %v\n", paint(opts.Color, scarlet, "!"), name, err)
		}
	}

	_, err := io.WriteString(w, b.String())
	return err
}

func summary(jobs []job.Job, opts Options) string {
	n := len(jobs)
	noun := "new"
	if n == 1 {
		noun = "new"
	}
	s := fmt.Sprintf("%d %s of %d fetched", n, noun, opts.Total)
	if len(opts.Sources) > 0 {
		s += " · " + sourceNames(opts.Sources)
	}
	return s
}

func scoreBadge(score int, color bool) string {
	text := fmt.Sprintf("%3d", score)
	if !color {
		return text
	}
	code := gray
	switch {
	case score >= 75:
		code = amber
	case score >= 50:
		code = bold
	}
	return paint(true, code, text)
}

func titleText(j job.Job, color bool, width int) string {
	title := truncate(j.Title, width-8)
	if !color {
		return title
	}
	if j.URL != "" {
		title = "\x1b]8;;" + j.URL + "\x1b\\" + title + "\x1b]8;;\x1b\\"
	}
	return paint(true, bold, title)
}

var sourceLabels = map[string]string{
	"hn":        "Hacker News",
	"remotive":  "Remotive",
	"remoteok":  "Remote OK",
	"wwr":       "We Work Remotely",
	"arbeitnow": "Arbeitnow",
	"jobicy":    "Jobicy",
}

func sourceName(s string) string {
	if label, ok := sourceLabels[s]; ok {
		return label
	}
	return s
}

func sourceNames(names []string) string {
	labels := make([]string, 0, len(names))
	for _, n := range names {
		labels = append(labels, sourceName(n))
	}
	return strings.Join(labels, " · ")
}

func age(posted, now time.Time) string {
	if posted.IsZero() {
		return ""
	}
	d := now.Sub(posted)
	switch {
	case d < time.Hour:
		return fmt.Sprintf("%dm", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("%dh", int(d.Hours()))
	case d < 7*24*time.Hour:
		return fmt.Sprintf("%dd", int(d.Hours()/24))
	default:
		return fmt.Sprintf("%dw", int(d.Hours()/(24*7)))
	}
}

func truncate(s string, max int) string {
	if max <= 1 || utf8.RuneCountInString(s) <= max {
		return s
	}
	r := []rune(s)
	return string(r[:max-1]) + "…"
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// Markdown prints a digest that survives a paste into chat or a notes app.
func Markdown(w io.Writer, jobs []job.Job, opts Options) error {
	var b strings.Builder
	fmt.Fprintf(&b, "**jobwatch** · %s\n\n", opts.Now.Format("2 Jan 2006"))

	if len(jobs) == 0 {
		b.WriteString("nothing new above your filter\n")
	}
	for _, j := range jobs {
		fmt.Fprintf(&b, "- **%d** [%s](%s) — %s\n", j.Score, j.Title, j.URL, j.Company)
		var meta []string
		if j.Location != "" {
			meta = append(meta, j.Location)
		}
		if j.Salary != "" {
			meta = append(meta, j.Salary)
		}
		meta = append(meta, sourceName(j.Source), age(j.Posted, opts.Now))
		fmt.Fprintf(&b, "  %s\n", strings.Join(meta, " · "))
	}

	b.WriteString("\n")
	fmt.Fprintf(&b, "_%s_\n", summary(jobs, opts))
	_, err := io.WriteString(w, b.String())
	return err
}

// JSON prints the postings for other tools to consume.
func JSON(w io.Writer, jobs []job.Job) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(jobs)
}
