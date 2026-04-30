package services

import (
	"context"
	"log/slog"
	"time"

	"github.com/kingknull/oblivra/internal/events"
	"github.com/kingknull/oblivra/internal/reconstruction"
	"github.com/kingknull/oblivra/internal/storage/hot"
)

// ReconstructionService bundles session grouping + state-at-time-T behind a
// single Wails-facing surface.
type ReconstructionService struct {
	log      *slog.Logger
	sessions *reconstruction.SessionEngine
	state    *reconstruction.StateService
	network  *reconstruction.NetworkStitcher
	entity   *reconstruction.EntityIndex
	cmdline  *reconstruction.CmdLineEngine
}

func NewReconstructionService(log *slog.Logger, h *hot.Store) *ReconstructionService {
	return &ReconstructionService{
		log:      log,
		sessions: reconstruction.NewSessionEngine(),
		state:    reconstruction.NewStateService(h),
		network:  reconstruction.NewNetworkStitcher(),
		entity:   reconstruction.NewEntityIndex(),
		cmdline:  reconstruction.NewCmdLineEngine(),
	}
}

func (r *ReconstructionService) ServiceName() string { return "ReconstructionService" }

// Observe is called from the bus fan-out; it routes the event to whichever
// reconstructor cares.
func (r *ReconstructionService) Observe(ctx context.Context, ev events.Event) {
	r.sessions.Observe(ctx, ev)
	r.network.Observe(ctx, ev)
	r.entity.Observe(ctx, ev)
	r.cmdline.Observe(ctx, ev)
}

// EntityProfile returns the rolled-up profile for a (kind, id) pair.
func (r *ReconstructionService) EntityProfile(kind, id string) *reconstruction.EntityProfile {
	return r.entity.Profile(kind, id)
}

// EntityList lists profiles per kind.
func (r *ReconstructionService) EntityList(kind string, limit int) []reconstruction.EntityProfile {
	return r.entity.List(kind, limit)
}

// CmdLines returns the most recent command-line invocations.
func (r *ReconstructionService) CmdLines(host string, limit int) []reconstruction.CmdLine {
	return r.cmdline.ByHost(host, limit)
}

// SuspiciousCmdLines surfaces the flagged ones.
func (r *ReconstructionService) SuspiciousCmdLines(limit int) []reconstruction.CmdLine {
	return r.cmdline.Suspicious(limit)
}

func (r *ReconstructionService) Sessions(host string) []reconstruction.Session {
	return r.sessions.Sessions(host)
}

func (r *ReconstructionService) Session(id string) (*reconstruction.Session, bool) {
	return r.sessions.Get(id)
}

func (r *ReconstructionService) StateAt(ctx context.Context, tenantID, hostID string, at time.Time) (*reconstruction.ProcessSnapshot, error) {
	return r.state.Reconstruct(ctx, tenantID, hostID, at)
}

// FlowsByHost returns the network 5-tuple stories the stitcher remembers.
func (r *ReconstructionService) FlowsByHost(host string) []reconstruction.Flow {
	return r.network.FlowsByHost(host)
}

// DNSByQuery returns DNS resolutions seen for a given hostname.
func (r *ReconstructionService) DNSByQuery(query string) []reconstruction.DNSAnswer {
	return r.network.DNSByQuery(query)
}
