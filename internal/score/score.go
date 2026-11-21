// Package score ranks postings against a profile.
package score

import (
	"regexp"
	"strings"
	"time"

	"github.com/codeafridi/jobwatch/internal/job"
)

// Profile describes what the reader wants to see.
type Profile struct {
	Include      []string
	Exclude      []string
	Locations    []string
	SkipExcluded bool
}

// restrictedRe catches postings that look remote but only hire in one region.
var restrictedRe = regexp.MustCompile(
	`(?i)\b(us only|usa only|united states|us-based|canada only|eu only|europe only|uk only|onsite|on-site)\b`)

type matcher struct {
	term []string
	res  []*regexp.Regexp
}

func newMatcher(terms []string) matcher {
	m := matcher{}
	for _, t := range terms {
		t = strings.TrimSpace(t)
		if t == "" {
			continue
		}
		// Word-ish boundaries so "go" does not match "Django" and "lead" does
		// not match "leading", while "c++" still works.
		m.term = append(m.term, t)
		m.res = append(m.res, regexp.MustCompile(
			`(?i)(^|[^a-z0-9])`+regexp.QuoteMeta(t)+`($|[^a-z0-9])`))
	}
	return m
}

func (m matcher) matchOne(i int, s string) bool {
	return i >= 0 && i < len(m.res) && m.res[i].MatchString(s)
}

func (m matcher) match(s string) bool {
	for _, re := range m.res {
		if re.MatchString(s) {
			return true
		}
	}
	return false
}

// Apply scores the posting in place and reports whether it should be shown at
// all. Reasons are filled in for --explain.
func Apply(p *job.Job, prof Profile, now time.Time) bool {
	include := newMatcher(prof.Include)
	exclude := newMatcher(prof.Exclude)
	locations := newMatcher(prof.Locations)

	title := strings.ToLower(p.Title)
	tags := strings.ToLower(strings.Join(p.Tags, " ") + " " + p.Description)
	place := strings.ToLower(p.Location)

	if prof.SkipExcluded && exclude.match(title) {
		return false
	}

	var s int

	if exclude.match(title) {
		s -= 35
		p.Reasons = append(p.Reasons, "excluded role word")
	}

	titleHits := 0
	for i, term := range include.term {
		if !include.matchOne(i, title) {
			continue
		}
		s += 20
		titleHits++
		p.Reasons = append(p.Reasons, "title: "+term)
		if titleHits == 3 {
			break
		}
	}

	// Without a positive signal the posting is just "remote and recent", which
	// is how a clinic in Florida ends up next to a Go job.
	positive := titleHits > 0
	if !positive {
		for i, term := range include.term {
			if include.matchOne(i, tags) {
				s += 6
				p.Reasons = append(p.Reasons, "tag: "+term)
				positive = true
				break
			}
		}
	}
	if !positive {
		return false
	}

	switch {
	case place == "":
		s += 4
	case locations.match(place):
		s += 16
		p.Reasons = append(p.Reasons, "open location")
	default:
		s -= 12
		p.Reasons = append(p.Reasons, "location may not apply")
	}
	if restrictedRe.MatchString(place) {
		s -= 14
		p.Reasons = append(p.Reasons, "region restricted")
	}

	if !p.Posted.IsZero() {
		age := now.Sub(p.Posted)
		switch {
		case age < 48*time.Hour:
			s += 12
			p.Reasons = append(p.Reasons, "fresh")
		case age < 7*24*time.Hour:
			s += 6
		case age > 30*24*time.Hour:
			s -= 6
		}
	}

	if s < 0 {
		s = 0
	}
	if s > 100 {
		s = 100
	}
	p.Score = s
	return true
}
