# jobwatch

A small digest of remote engineering jobs, for people who read their terminal
more often than five job boards.

```
  jobwatch  Sun 22 Nov 2026
  ────────────────────────────────────────────────────────────────

   62  Full Stack Engineer (mostly backend + API)
      New Product · Remote · Hacker News · 6d
   36  Software Engineer / Backend Engineer
      Hemant Singh · Remote · Hacker News · 1w
   20  Infrastructure Software Engineer, Orchestration
      Dropbox · Poland · Jobicy · 4h

  7 new of 412 fetched · Hacker News · Remotive · Remote OK · We Work Remotely · Arbeitnow · Jobicy
```

It fetches six boards, drops the noise, ranks what is left against a profile
you control, remembers what it has already shown, and prints the rest. No
account, no server, no tracking, one binary.

## Why another job tool

Job boards are optimised for posting, not for reading. The same role appears on
three boards, half the feed is sales and support, and the seniority you asked to
avoid comes back every morning. `jobwatch` does the boring part:

- **Rank, don't flood.** Postings need a real signal to appear at all: a matching
  role word in the title, or a matching tag. "Remote and recent" is not enough.
- **Remember.** A posting is printed once. `--all` when you want the full board.
- **Stay small.** Standard library only, one binary, no database, state in
  `~/.local/state/jobwatch/seen.json`.

## Install

```sh
go install github.com/codeafridi/jobwatch@latest
```

Or from a checkout: `go build -o ~/.local/bin/jobwatch .`

## Use

```sh
jobwatch                      # what is new, best first
jobwatch --explain            # why each posting scored the way it did
jobwatch --all --limit 50     # everything, including what you have seen
jobwatch --format markdown    # paste into notes or a chat
jobwatch --format json        # pipe into something else
jobwatch --sources hn,jobicy  # just the boards you trust
jobwatch --min-score 35       # stricter
```

Flags:

```
--all            show postings already reported
--explain        say why each posting scored the way it did
--format         ansi, markdown or json (default ansi)
--limit          maximum postings to print, 0 for all (default 25)
--min-score      drop postings below this score (default from config)
--since          only postings newer than this (default 336h)
--sources        comma separated sources, default all
--state          state file (default XDG state dir)
--config         config file (default XDG config dir)
--no-color       disable colour
--list-sources   print the available sources and exit
```

Colour is used only when stdout is a terminal, and `NO_COLOR` is respected.

## Profile

Defaults are a backend/infrastructure profile that can be seen from India. To
change them, write `~/.config/jobwatch/config.json` (every field is optional):

```json
{
  "include": ["go", "kubernetes", "platform", "rust", "c++"],
  "exclude": ["senior", "staff", "manager", "sales"],
  "locations": ["worldwide", "anywhere", "india", "asia", "emea"],
  "sources": ["hn", "remotive", "remoteok", "wwr", "arbeitnow", "jobicy"],
  "min_score": 25,
  "skip_excluded": true
}
```

Matching is word-aware: `go` does not match "Django", `lead` does not match
"leading", and `c++` still works.

## Scoring

Postings are scored 0-100 and sorted, so the order is the product:

| signal | points |
| --- | --- |
| role word in the title | +20 each, up to three |
| role word in a tag | +6, only when the title has none |
| location open to you | +16 |
| posted in the last 48 hours | +12 |
| region-locked ("US only", "EU only", onsite) | −14 |
| excluded word in the title | −35, or dropped entirely |

A posting without a title or tag match is never shown, however fresh it is.

## Sources

Thank you to the boards that keep public APIs:

- [Hacker News "Who is hiring?"](https://news.ycombinator.com/user?id=whoishiring) via [Algolia](https://hn.algolia.com/api)
- [Remotive](https://remotive.com/api/remote-jobs)
- [Remote OK](https://remoteok.com/api) — please support them; their API asks for attribution and a followed link back
- [We Work Remotely](https://weworkremotely.com/categories/remote-programming-jobs.rss)
- [Arbeitnow](https://www.arbeitnow.com/api/job-board-api)
- [Jobicy](https://jobicy.com/api/v2/remote-jobs)

Each posting links to the original ad. Attribution is printed in the digest
footer. Some feeds ask not to be redistributed to aggregators; this tool shows
them to one person and links back, which is the intended use.

## Daily digest

`jobwatch --format markdown` is made for a scheduler. With `cron`:

```cron
30 8 * * *  /home/you/.local/bin/jobwatch --format markdown > /tmp/jobs.md
```

and a small script that sends `/tmp/jobs.md` wherever you read messages.

## License

MIT.
