// Package job holds the job posting model shared by every source.
package job

import (
	"crypto/sha256"
	"encoding/hex"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"
)

// Job is a single posting, normalised across sources.
type Job struct {
	ID       string
	Source   string
	Title    string
	Company  string
	Location string
	Salary   string
	URL      string
	Tags     []string
	Posted   time.Time

	// Description is extra text used for matching only; it is not printed.
	Description string `json:"-"`

	// Filled in by the scorer.
	Score   int
	Reasons []string
}

// NewID builds a stable identifier from the source and the source's own key.
// Falls back to the URL for feeds that have no id.
func NewID(source, key, url string) string {
	if key == "" {
		key = url
	}
	sum := sha256.Sum256([]byte(source + "\x00" + key))
	return hex.EncodeToString(sum[:8])
}

// DedupeKey groups the same posting seen on several boards. Company and title
// are enough in practice: boards do not describe different roles identically.
func (j Job) DedupeKey() string {
	norm := func(s string) string {
		return strings.Join(strings.Fields(strings.ToLower(s)), " ")
	}
	return norm(j.Company) + "\x00" + norm(j.Title)
}

// CleanTitle tidies a title without changing its meaning: collapses whitespace,
// drops trailing location suffixes that boards append ("(Remote)", "- US only")
// and the "1234 | " prefixes some feeds add.
func CleanTitle(title string) string {
	title = strings.Join(strings.Fields(title), " ")
	title = numericPrefix.ReplaceAllString(title, "")
	for _, suffix := range []string{" (remote)", " [remote]", " - remote"} {
		if strings.HasSuffix(strings.ToLower(title), suffix) {
			title = title[:len(title)-len(suffix)]
			break
		}
	}
	return strings.TrimSpace(title)
}

var numericPrefix = regexp.MustCompile(`^\d{3,}\s*\|\s*`)

// cp1252 maps the runes Windows-1252 puts in 0x80-0x9F back to their bytes.
// Without it "â€“" cannot be turned back into an en dash: the middle rune is
// U+20AC, not 0x80.
var cp1252 = map[rune]byte{
	'€': 0x80, '‚': 0x82, 'ƒ': 0x83, '„': 0x84, '…': 0x85, '†': 0x86,
	'‡': 0x87, 'ˆ': 0x88, '‰': 0x89, 'Š': 0x8A, '‹': 0x8B, 'Œ': 0x8C,
	'Ž': 0x8E, '‘': 0x91, '’': 0x92, '“': 0x93, '”': 0x94, '•': 0x95,
	'–': 0x96, '—': 0x97, '˜': 0x98, '™': 0x99, 'š': 0x9A, '›': 0x9B,
	'œ': 0x9C, 'ž': 0x9E, 'Ÿ': 0x9F,
}

// FixMojibake repairs text that was UTF-8 but got decoded as Windows-1252 by
// a board, e.g. "â€“" for an en dash.
func FixMojibake(s string) string {
	if !strings.ContainsAny(s, "ÂÃâ€") {
		return s
	}
	b := make([]byte, 0, len(s))
	for _, r := range s {
		switch {
		case r <= 0xFF:
			b = append(b, byte(r))
		default:
			c, ok := cp1252[r]
			if !ok {
				return s
			}
			b = append(b, c)
		}
	}
	if utf8.Valid(b) {
		return string(b)
	}
	return s
}

// SplitCompanyTitle splits the "Company: Role" convention used by RSS feeds.
func SplitCompanyTitle(s string) (company, title string) {
	if i := strings.Index(s, ":"); i > 0 {
		return strings.TrimSpace(s[:i]), strings.TrimSpace(s[i+1:])
	}
	return "", strings.TrimSpace(s)
}
