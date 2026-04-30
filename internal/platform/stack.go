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
	"time"

	"github.com/kingknull/oblivra/internal/datapath"
	"github.com/kingknull/oblivra/internal/events"
	"github.com/kingknull/oblivra/internal/ingest"
	"github.com/kingknull/oblivra/internal/listeners"
	"github.com/kingknull/oblivra/internal/scheduler"
	"github.com/kingknull/oblivra/internal/services"
	"github.com/kingknull/oblivra/internal/sigma"
	"github.com/kingknull/oblivra/internal/storage/hot"
	"github.com/kingknull/oblivra/internal/storage/search"
	"github.com/kingknull/oblivra/internal/storage/tiering"
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
	Foren   *services.ForensicsService
	Tier    *services.TieringService
	Lineage *services.LineageService
	Vault          *services.VaultService
	Timeline       *services.TimelineService
	Investigations *services.InvestigationsService
	Reconstruction *services.ReconstructionService
	TenantPolicy   *services.TenantPolicyService
	Trust          *services.TrustService
	Quality        *services.QualityService
	Graph          *services.EvidenceGraphService
	Import         *services.ImportService
	Report         *services.ReportService
	Bus            *events.Bus
	Syslog  *listeners.SyslogUDP
	NetFlow *listeners.NetFlowV5

	pipeline *ingest.Pipeline
	hot      *hot.Store
	wal      *wal.WAL
	search   *search.Index

	cancelFns []func()
	wg        sync.WaitGroup

	scheduler   *scheduler.Scheduler
	sigmaWatch  *sigma.Watcher
}

type Options struct {
	Logger         *slog.Logger
	InMemory       bool
	SyslogAddr     string // "" disables; ":1514" enables
	NetFlowAddr    string // "" disables; ":2055" enables
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
	audit, err := services.NewDurable(opts.Logger, dir, hmacKey())
	if err != nil {
		_ = idx.Close()
		_ = store.Close()
		_ = w.Close()
		return nil, fmt.Errorf("audit journal: %w", err)
	}
	rules := services.NewRulesService(opts.Logger, alerts)
	// Wire the Sigma directory if it exists alongside the binary or under
	// the data dir. The loader is best-effort; missing dir = no-op.
	sigmaDir := pickSigmaDir(dir)
	rules.AttachSigma(sigmaDir, func(d string) ([]services.Rule, []error) {
		return sigma.LoadDir(d)
	})
	if _, err := rules.Reload(); err != nil {
		opts.Logger.Warn("sigma reload", "err", err)
	}
	fleet := services.NewFleetService(opts.Logger, pipeline)
	ueba := services.NewUebaService(opts.Logger, alerts)
	ndr := services.NewNdrService(opts.Logger)
	foren := services.NewForensicsService(store, audit)

	warmDir, err := datapath.Sub(dir, "warm.parquet")
	if err != nil {
		_ = idx.Close()
		_ = store.Close()
		_ = w.Close()
		return nil, fmt.Errorf("warm dir: %w", err)
	}
	tenantPolicy, err := services.NewTenantPolicyService(opts.Logger, dir)
	if err != nil {
		_ = idx.Close()
		_ = store.Close()
		_ = w.Close()
		return nil, fmt.Errorf("tenant policy: %w", err)
	}
	migrator, err := tiering.New(opts.Logger, store, tiering.Options{
		WarmDir:    warmDir,
		ResolveAge: tenantPolicy.HotMaxAge,
	})
	if err != nil {
		_ = idx.Close()
		_ = store.Close()
		_ = w.Close()
		return nil, fmt.Errorf("tiering: %w", err)
	}
	tier := services.NewTieringService(opts.Logger, migrator)
	lineage := services.NewLineageService(opts.Logger)
	vaultSvc := services.NewVaultService(opts.Logger, dir)
	timeline := services.NewTimelineService(store, alerts, foren)
	investigations, err := services.NewInvestigationsService(opts.Logger, dir, store, alerts, foren, audit)
	if err != nil {
		_ = idx.Close()
		_ = store.Close()
		_ = w.Close()
		return nil, fmt.Errorf("investigations: %w", err)
	}
	recon := services.NewReconstructionService(opts.Logger, store)
	trustSvc := services.NewTrustService(opts.Logger)
	qualitySvc := services.NewQualityService(opts.Logger)
	graphSvc := services.NewEvidenceGraphService(opts.Logger)
	importSvc := services.NewImportService(opts.Logger, pipeline)
	reportSvc := services.NewReportService(opts.Logger, investigations, audit)

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
		Tier:     tier,
		Lineage:  lineage,
		Vault:    vaultSvc,
		Timeline:       timeline,
		Investigations: investigations,
		Reconstruction: recon,
		TenantPolicy:   tenantPolicy,
		Trust:          trustSvc,
		Quality:        qualitySvc,
		Graph:          graphSvc,
		Import:         importSvc,
		Report:         reportSvc,
		Bus:            bus,
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
	if opts.NetFlowAddr != "" && opts.StartListeners {
		nf := listeners.NewNetFlowV5(opts.Logger, ndr, listeners.NetFlowOptions{Addr: opts.NetFlowAddr})
		ctx, cancel := context.WithCancel(context.Background())
		stack.cancelFns = append(stack.cancelFns, cancel)
		stack.wg.Add(1)
		go func() {
			defer stack.wg.Done()
			if err := nf.Start(ctx); err != nil {
				opts.Logger.Error("netflow listener stopped", "err", err)
			}
		}()
		stack.NetFlow = nf
	}

	// Audit our own startup so the chain has a root.
	_ = audit.Append(context.Background(), "system", "platform.start", "default", map[string]string{
		"dataDir": dir,
	})

	// Background scheduler: run warm-tier migrations every 6h, audit health
	// every 5m. The intervals are deliberately conservative — overrides land
	// when we ship a config file.
	stack.scheduler = scheduler.New(opts.Logger)
	stack.scheduler.Add(scheduler.Job{
		Name:     "tiering.warm-migrate",
		Interval: 6 * time.Hour,
		Run: func(ctx context.Context) error {
			_, err := tier.Promote(ctx)
			return err
		},
	})
	stack.scheduler.Add(scheduler.Job{
		Name:     "audit.health",
		Interval: 5 * time.Minute,
		Run: func(_ context.Context) error {
			res := audit.Verify()
			if !res.OK {
				opts.Logger.Error("audit chain broken!", "brokenAt", res.BrokenAt, "entries", res.Entries)
			}
			return nil
		},
	})
	stack.scheduler.Add(scheduler.Job{
		Name:     "audit.daily-anchor",
		Interval: 1 * time.Hour, // fires hourly; AnchorYesterday is idempotent
		Run: func(ctx context.Context) error {
			return audit.AnchorYesterday(ctx)
		},
	})
	stack.scheduler.Start(context.Background())
	stack.cancelFns = append(stack.cancelFns, stack.scheduler.Stop)

	// Sigma hot-reload — recompile rules on any change in the sigma directory.
	if sigmaDir != "" {
		stack.sigmaWatch = sigma.NewWatcher(opts.Logger, sigmaDir)
		_ = stack.sigmaWatch.Start(context.Background(), func() {
			n, err := rules.Reload()
			if err != nil {
				opts.Logger.Warn("sigma hot-reload failed", "err", err)
				return
			}
			opts.Logger.Info("sigma hot-reload", "rules", n)
		})
		stack.cancelFns = append(stack.cancelFns, stack.sigmaWatch.Stop)
	}

	opts.Logger.Info("platform ready", "dataDir", dir)
	return stack, nil
}

// startProcessors fans every event through the asynchronous processors.
func (s *Stack) startProcessors(parent context.Context) {
	processors := []func(context.Context, events.Event){
		func(ctx context.Context, ev events.Event) { s.Rules.Evaluate(ctx, ev) },
		func(ctx context.Context, ev events.Event) { s.Ueba.Observe(ctx, ev) },
		func(_ context.Context, ev events.Event) { s.Foren.Observe(ev) },
		func(_ context.Context, ev events.Event) { s.Lineage.Observe(ev) },
		func(ctx context.Context, ev events.Event) { s.Reconstruction.Observe(ctx, ev) },
		func(ctx context.Context, ev events.Event) { s.Trust.Observe(ctx, ev) },
		func(ctx context.Context, ev events.Event) { s.Quality.Observe(ctx, ev) },
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
	if s.Investigations != nil {
		if err := s.Investigations.Close(); err != nil && first == nil {
			first = err
		}
	}
	if s.Audit != nil {
		if err := s.Audit.Close(); err != nil && first == nil {
			first = err
		}
	}
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

// pickSigmaDir picks the first existing candidate among:
//   $OBLIVRA_SIGMA_DIR, ./sigma, {dataDir}/sigma
func pickSigmaDir(dataDir string) string {
	candidates := []string{
		os.Getenv("OBLIVRA_SIGMA_DIR"),
		"sigma",
	}
	if dataDir != "" {
		candidates = append(candidates, dataDir+"/sigma")
	}
	for _, c := range candidates {
		if c == "" {
			continue
		}
		if info, err := os.Stat(c); err == nil && info.IsDir() {
			return c
		}
	}
	return ""
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
