// Package platform constructs the OBLIVRA service stack — WAL, hot store,
// search index, ingest pipeline, listeners, and the user-facing services that
// wrap them. Both the Wails desktop binary and the headless server bootstrap
// through here, so the layout stays identical across surfaces.
package platform

import (
	"context"
	"crypto/rand"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"sync"

	"github.com/kingknull/oblivra/internal/datapath"
	"github.com/kingknull/oblivra/internal/events"
	"github.com/kingknull/oblivra/internal/ingest"
	"github.com/kingknull/oblivra/internal/listeners"
	"github.com/kingknull/oblivra/internal/services"
	"github.com/kingknull/oblivra/internal/storage/hot"
	"github.com/kingknull/oblivra/internal/storage/search"
	"github.com/kingknull/oblivra/internal/wal"
)

type Stack struct {
	Log    *slog.Logger
	System *services.SystemService
	Siem   *services.SiemService
	Alerts *services.AlertService
	Intel  *services.ThreatIntelService
	Rules  *services.RulesService
	Audit  *services.AuditService
	Fleet  *services.FleetService
	Ueba   *services.UebaService
	Ndr    *services.NdrService
	Foren  *services.ForensicsService
	Bus    *events.Bus
	Syslog *listeners.SyslogUDP

	pipeline *ingest.Pipeline
	hot      *hot.Store
	wal      *wal.WAL
	search   *search.Index

	cancelFns []func()
	wg        sync.WaitGroup
}

type Options struct {
	Logger         *slog.Logger
	InMemory       bool
	SyslogAddr     string // "" disables; ":1514" enables
	StartListeners bool
}

func New(opts Options) (*Stack, error) {
	if opts.Logger == nil {
		opts.Logger = slog.Default()
	}

	dir, err := datapath.Resolve()
	if err != nil {
		return nil, fmt.Errorf("resolve data dir: %w", err)
	}

	walDir, err := datapath.Sub(dir, "wal")
	if err != nil {
		return nil, fmt.Errorf("wal dir: %w", err)
	}
	w, err := wal.Open(wal.Options{Dir: walDir, SyncOnAppend: true})
	if err != nil {
		return nil, fmt.Errorf("open wal: %w", err)
	}

	hotDir, err := datapath.Sub(dir, "siem_hot.badger")
	if err != nil {
		_ = w.Close()
		return nil, fmt.Errorf("hot dir: %w", err)
	}
	store, err := hot.Open(hot.Options{Dir: hotDir, InMemory: opts.InMemory})
	if err != nil {
		_ = w.Close()
		return nil, fmt.Errorf("open hot store: %w", err)
	}

	indexDir, err := datapath.Sub(dir, "bleve.idx")
	if err != nil {
		_ = store.Close()
		_ = w.Close()
		return nil, fmt.Errorf("index dir: %w", err)
	}
	idx, err := search.Open(search.Options{Dir: indexDir, InMemory: opts.InMemory})
	if err != nil {
		_ = store.Close()
		_ = w.Close()
		return nil, fmt.Errorf("open search index: %w", err)
	}

	bus := events.NewBus(512)
	pipeline := ingest.New(opts.Logger, w, store, idx, bus)

	system := services.NewSystemService(opts.Logger)
	siem := services.NewSiemService(opts.Logger, pipeline)
	alerts := services.NewAlertService(opts.Logger)
	intel := services.NewThreatIntelService(opts.Logger)
	audit := services.NewAuditService(opts.Logger, hmacKey())
	rules := services.NewRulesService(opts.Logger, alerts)
	fleet := services.NewFleetService(opts.Logger, pipeline)
	ueba := services.NewUebaService(opts.Logger, alerts)
	ndr := services.NewNdrService(opts.Logger)
	foren := services.NewForensicsService(store, audit)

	stack := &Stack{
		Log:      opts.Logger,
		System:   system,
		Siem:     siem,
		Alerts:   alerts,
		Intel:    intel,
		Rules:    rules,
		Audit:    audit,
		Fleet:    fleet,
		Ueba:     ueba,
		Ndr:      ndr,
		Foren:    foren,
		Bus:      bus,
		pipeline: pipeline,
		hot:      store,
		wal:      w,
		search:   idx,
	}

	// Subscribe processors to the event bus so detection / UEBA / forensics
	// run on every ingested event without the ingest path waiting on them.
	stack.startProcessors(context.Background())

	if opts.SyslogAddr != "" && opts.StartListeners {
		s := listeners.NewSyslogUDP(opts.Logger, pipeline, listeners.SyslogOptions{Addr: opts.SyslogAddr})
		ctx, cancel := context.WithCancel(context.Background())
		stack.cancelFns = append(stack.cancelFns, cancel)
		stack.wg.Add(1)
		go func() {
			defer stack.wg.Done()
			if err := s.Start(ctx); err != nil {
				opts.Logger.Error("syslog listener stopped", "err", err)
			}
		}()
		stack.Syslog = s
	}

	// Audit our own startup so the chain has a root.
	_ = audit.Append(context.Background(), "system", "platform.start", "default", map[string]string{
		"dataDir": dir,
	})

	opts.Logger.Info("platform ready", "dataDir", dir)
	return stack, nil
}

// startProcessors fans every event through the asynchronous processors.
func (s *Stack) startProcessors(parent context.Context) {
	processors := []func(context.Context, events.Event){
		func(ctx context.Context, ev events.Event) { s.Rules.Evaluate(ctx, ev) },
		func(ctx context.Context, ev events.Event) { s.Ueba.Observe(ctx, ev) },
		func(_ context.Context, ev events.Event) { s.Foren.Observe(ev) },
		func(_ context.Context, ev events.Event) {
			// IOC enrichment — match every text field against the IOC table.
			candidates := []string{ev.HostID, ev.Message, ev.Raw}
			for k, v := range ev.Fields {
				if k == "src" || k == "dst" || k == "ip" {
					candidates = append(candidates, v)
				}
			}
			for _, c := range candidates {
				if c == "" {
					continue
				}
				for _, tok := range strings.Fields(c) {
					if r := s.Intel.Lookup(tok); r.Match {
						s.Alerts.Raise(parent, services.Alert{
							TenantID: ev.TenantID,
							RuleID:   "ioc-match",
							RuleName: "Threat-intel IOC matched",
							Severity: services.AlertSeverityHigh,
							HostID:   ev.HostID,
							Message:  "matched indicator " + r.Indicator.Value + " (" + string(r.Indicator.Type) + ")",
							MITRE:    []string{"T1071"},
							EventIDs: []string{ev.ID},
						})
					}
				}
			}
		},
	}

	for _, fn := range processors {
		ch, unsub := s.Bus.Subscribe()
		s.cancelFns = append(s.cancelFns, unsub)
		s.wg.Add(1)
		go func(fn func(context.Context, events.Event), ch <-chan events.Event) {
			defer s.wg.Done()
			for ev := range ch {
				fn(parent, ev)
			}
		}(fn, ch)
	}
}

// Close releases all underlying resources.
func (s *Stack) Close() error {
	for _, c := range s.cancelFns {
		c()
	}
	s.wg.Wait()

	var first error
	if s.search != nil {
		if err := s.search.Close(); err != nil && first == nil {
			first = err
		}
	}
	if s.hot != nil {
		if err := s.hot.Close(); err != nil && first == nil {
			first = err
		}
	}
	if s.wal != nil {
		if err := s.wal.Close(); err != nil && first == nil {
			first = err
		}
	}
	return first
}

// hmacKey returns the audit-log signing key. Generated random per-process if
// one isn't configured; durable storage of the signing key lands in Phase 5.
func hmacKey() []byte {
	if v := os.Getenv("OBLIVRA_AUDIT_KEY"); v != "" {
		return []byte(v)
	}
	var b [32]byte
	_, _ = rand.Read(b[:])
	return b[:]
}
