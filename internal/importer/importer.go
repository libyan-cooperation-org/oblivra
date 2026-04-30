// Package importer ingests historical event data from external sources —
// JSONL exports from another SIEM, raw syslog dumps, etc. Every imported
// event carries `Provenance.IngestPath="import"` so it's distinguishable
// from live ingest and the analyst can tell at a glance which events
// arrived from a backfill vs the wire.
//
// The importer is also the entry point for the static health summary the
// task tracker calls out: time-range coverage, host count, format mix,
// parse-failure ratio. We compute it as we stream, so a 10GB import comes
// back with a useful summary even before the analyst runs a query.
package importer

import (
	"bufio"
	"context"
	"errors"
	"io"
	"log/slog"
	"sort"
	"strings"
	"time"

	"github.com/kingknull/oblivra/internal/events"
	"github.com/kingknull/oblivra/internal/ingest"
	"github.com/kingknull/oblivra/internal/parsers"
)

// Summary is the static health report produced by the importer.
type Summary struct {
	Lines           int            `json:"lines"`
	Imported        int            `json:"imported"`
	ParseFailures   int            `json:"parseFailures"`
	HostCount       int            `json:"hostCount"`
	From            time.Time      `json:"from"`
	To              time.Time      `json:"to"`
	FormatMix       map[string]int `json:"formatMix"`
	SampleHosts     []string       `json:"sampleHosts"`
	StartedAt       time.Time      `json:"startedAt"`
	FinishedAt      time.Time      `json:"finishedAt"`
}

type Options struct {
	TenantID string
	Source   string             // free-form label, e.g. "splunk-export-2026-Q1"
	Format   parsers.Format     // FormatAuto unless caller knows
	Logger   *slog.Logger
}

type Importer struct {
	pipeline *ingest.Pipeline
	opts     Options
}

func New(p *ingest.Pipeline, opts Options) *Importer {
	if opts.Format == "" {
		opts.Format = parsers.FormatAuto
	}
	if opts.TenantID == "" {
		opts.TenantID = "default"
	}
	if opts.Source == "" {
		opts.Source = "import"
	}
	if opts.Logger == nil {
		opts.Logger = slog.Default()
	}
	return &Importer{pipeline: p, opts: opts}
}

// Run reads line-delimited input. Lines that look like JSON-encoded events
// are unmarshalled directly; everything else goes through the format-aware
// parser. Returns a summary so the UI can show "what just landed" without
// running a search.
func (i *Importer) Run(ctx context.Context, r io.Reader) (Summary, error) {
	if r == nil {
		return Summary{}, errors.New("importer: nil reader")
	}
	br := bufio.NewReaderSize(r, 1<<20)
	hostSet := map[string]struct{}{}
	formatMix := map[string]int{}
	s := Summary{
		StartedAt: time.Now().UTC(),
		FormatMix: formatMix,
	}
	for {
		if err := ctx.Err(); err != nil {
			return s, err
		}
		line, err := br.ReadString('\n')
		line = strings.TrimRight(line, "\r\n")
		if line != "" {
			s.Lines++
			ev, perr := i.parseOne(line)
			if perr != nil || ev == nil {
				s.ParseFailures++
				if err == nil {
					continue
				}
			} else {
				ev.TenantID = i.opts.TenantID
				ev.Provenance.IngestPath = "import"
				ev.Provenance.Format = string(i.opts.Format)
				if ev.Provenance.Parser == "" {
					ev.Provenance.Parser = string(i.opts.Format)
				}
				if ferr := i.pipeline.Submit(ctx, ev); ferr == nil {
					s.Imported++
					if ev.HostID != "" {
						hostSet[ev.HostID] = struct{}{}
					}
					if s.From.IsZero() || ev.Timestamp.Before(s.From) {
						s.From = ev.Timestamp
					}
					if ev.Timestamp.After(s.To) {
						s.To = ev.Timestamp
					}
					formatMix[orFallback(ev.EventType, "raw")]++
				} else {
					s.ParseFailures++
				}
			}
		}
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return s, err
		}
	}
	s.FinishedAt = time.Now().UTC()
	s.HostCount = len(hostSet)
	for h := range hostSet {
		s.SampleHosts = append(s.SampleHosts, h)
	}
	sort.Strings(s.SampleHosts)
	if len(s.SampleHosts) > 20 {
		s.SampleHosts = s.SampleHosts[:20]
	}
	return s, nil
}

func (i *Importer) parseOne(line string) (*events.Event, error) {
	// Fast path: line is already a marshalled Event.
	if strings.HasPrefix(strings.TrimLeft(line, " \t"), "{") &&
		strings.Contains(line, `"schemaVersion"`) {
		ev := &events.Event{}
		if err := unmarshalEvent(line, ev); err == nil {
			if err := ev.Validate(); err != nil {
				return nil, err
			}
			return ev, nil
		}
	}
	return parsers.Parse(line, i.opts.Format)
}

func unmarshalEvent(line string, ev *events.Event) error {
	dec := jsonDecoder(line)
	dec.UseNumber()
	return dec.Decode(ev)
}

func orFallback(s, fallback string) string {
	if s == "" {
		return fallback
	}
	return s
}
