// Command jobwatch builds a small digest of remote engineering jobs that match
// a reader's profile.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/codeafridi/jobwatch/internal/config"
	"github.com/codeafridi/jobwatch/internal/job"
	"github.com/codeafridi/jobwatch/internal/render"
	"github.com/codeafridi/jobwatch/internal/score"
	"github.com/codeafridi/jobwatch/internal/store"
)

// version is overridden at build time with -ldflags "-X main.version=...".
var version = "0.1.0"

func main() {
	if err := run(os.Args[1:], os.Stdout, os.Stderr); err != nil {
		fmt.Fprintln(os.Stderr, "jobwatch:", err)
		os.Exit(1)
	}
}

func run(args []string, stdout, stderr io.Writer) error {
	fs := flag.NewFlagSet("jobwatch", flag.ContinueOnError)
	fs.SetOutput(stderr)
	fs.Usage = func() {
		_, _ = fmt.Fprintf(stderr, "usage: jobwatch [flags]\n\nflags:\n")
		fs.PrintDefaults()
		_, _ = fmt.Fprintf(stderr, "\nsources: %s\n", strings.Join(job.Names(), ", "))
	}

	var (
		all         = fs.Bool("all", false, "show postings already reported")
		explain     = fs.Bool("explain", false, "say why each posting scored the way it did")
		format      = fs.String("format", "ansi", "output format: ansi, markdown or json")
		limit       = fs.Int("limit", 25, "maximum postings to print, 0 for all")
		minScore    = fs.Int("min-score", -1, "drop postings below this score (default from config)")
		since       = fs.Duration("since", 14*24*time.Hour, "only postings newer than this")
		sourcesFlag = fs.String("sources", "", "comma separated sources, default all")
		statePath   = fs.String("state", "", "state file (default XDG state dir)")
		configPath  = fs.String("config", "", "config file (default XDG config dir)")
		noColor     = fs.Bool("no-color", false, "disable colour")
		quiet       = fs.Bool("quiet", false, "print nothing when there is nothing new")
		listSources = fs.Bool("list-sources", false, "print the available sources and exit")
		showVersion = fs.Bool("version", false, "print the version and exit")
	)

	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}

	if *showVersion {
		_, _ = fmt.Fprintln(stdout, "jobwatch", version)
		return nil
	}
	if *listSources {
		for _, f := range job.All() {
			_, _ = fmt.Fprintln(stdout, f.Name)
		}
		return nil
	}

	cfgPath := *configPath
	if cfgPath == "" {
		if p, err := config.DefaultPath(); err == nil {
			cfgPath = p
		}
	}
	cfg, err := config.Load(cfgPath)
	if err != nil {
		return fmt.Errorf("read config %s: %w", cfgPath, err)
	}
	if *minScore >= 0 {
		cfg.MinScore = *minScore
	}
	if *sourcesFlag != "" {
		cfg.Sources = splitList(*sourcesFlag)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 40*time.Second)
	defer cancel()

	client := &http.Client{Timeout: 25 * time.Second}
	postings, fetchErrs := job.FetchAll(ctx, client, cfg.Sources)
	if len(postings) == 0 && len(fetchErrs) > 0 {
		return fmt.Errorf("no source returned postings: %s", joinErrors(fetchErrs))
	}

	now := time.Now()
	profile := score.Profile{
		Include:      cfg.Include,
		Exclude:      cfg.Exclude,
		Locations:    cfg.Locations,
		SkipExcluded: cfg.SkipExcluded,
	}

	scored := make([]job.Job, 0, len(postings))
	for _, p := range postings {
		if !p.Posted.IsZero() && p.Posted.Before(now.Add(-*since)) {
			continue
		}
		if !score.Apply(&p, profile, now) {
			continue
		}
		if p.Score < cfg.MinScore {
			continue
		}
		scored = append(scored, p)
	}

	scored = dedupe(scored)
	sort.SliceStable(scored, func(i, j int) bool {
		if scored[i].Score != scored[j].Score {
			return scored[i].Score > scored[j].Score
		}
		if !scored[i].Posted.Equal(scored[j].Posted) {
			return scored[i].Posted.After(scored[j].Posted)
		}
		return scored[i].Title < scored[j].Title
	})

	stateFile := *statePath
	if stateFile == "" {
		p, err := store.DefaultPath()
		if err != nil {
			return err
		}
		stateFile = p
	}
	state, err := store.Load(stateFile)
	if err != nil {
		return fmt.Errorf("read state %s: %w", stateFile, err)
	}

	shown := scored
	if !*all {
		shown = make([]job.Job, 0, len(scored))
		for _, p := range scored {
			if state.IsNew(p.ID) {
				shown = append(shown, p)
			}
		}
	}
	if *limit > 0 && len(shown) > *limit {
		shown = shown[:*limit]
	}

	if *quiet && len(shown) == 0 && len(fetchErrs) == 0 {
		return nil
	}

	opts := render.Options{
		Color:   colorEnabled(*noColor, stdout),
		Width:   terminalWidth(),
		Explain: *explain,
		Now:     now,
		Total:   len(postings),
		Errors:  fetchErrs,
		Sources: cfg.Sources,
	}

	switch *format {
	case "ansi":
		err = render.ANSI(stdout, shown, opts)
	case "markdown", "md":
		err = render.Markdown(stdout, shown, opts)
	case "json":
		err = render.JSON(stdout, shown)
	default:
		return fmt.Errorf("unknown format %q, use ansi, markdown or json", *format)
	}
	if err != nil {
		return err
	}

	if !*all {
		ids := make([]string, 0, len(shown))
		for _, p := range shown {
			ids = append(ids, p.ID)
		}
		state.Mark(ids, now)
		state.Prune(now, 120*24*time.Hour)
		if err := state.Save(stateFile); err != nil {
			return fmt.Errorf("save state %s: %w", stateFile, err)
		}
	}
	return nil
}

// dedupe keeps one posting per company+title, preferring the higher score.
func dedupe(jobs []job.Job) []job.Job {
	best := make(map[string]int, len(jobs))
	out := make([]job.Job, 0, len(jobs))
	for _, p := range jobs {
		key := p.DedupeKey()
		if idx, ok := best[key]; ok {
			if p.Score > out[idx].Score {
				out[idx] = p
			}
			continue
		}
		best[key] = len(out)
		out = append(out, p)
	}
	return out
}

func colorEnabled(noColor bool, w io.Writer) bool {
	if noColor || os.Getenv("NO_COLOR") != "" || os.Getenv("TERM") == "dumb" {
		return false
	}
	f, ok := w.(*os.File)
	if !ok {
		return false
	}
	info, err := f.Stat()
	return err == nil && info.Mode()&os.ModeCharDevice != 0
}

func terminalWidth() int {
	if cols := os.Getenv("COLUMNS"); cols != "" {
		if n, err := strconv.Atoi(cols); err == nil && n > 20 {
			return n
		}
	}
	return 80
}

func splitList(s string) []string {
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}

func joinErrors(errs map[string]error) string {
	names := make([]string, 0, len(errs))
	for name := range errs {
		names = append(names, name)
	}
	sort.Strings(names)
	parts := make([]string, 0, len(names))
	for _, name := range names {
		parts = append(parts, name+": "+errs[name].Error())
	}
	return strings.Join(parts, "; ")
}
