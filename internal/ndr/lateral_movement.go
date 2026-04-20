package ndr

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/kingknull/oblivrashell/internal/eventbus"
	"github.com/kingknull/oblivrashell/internal/logger"
)

// HopChain represents a sequence of internal hops from a single origin.
type HopChain struct {
	OriginIP    string    `json:"origin_ip"`
	Hops        []string  `json:"hops"`
	FirstSeen   string    `json:"first_seen"`
	LastSeen    string    `json:"last_seen"`
	Ports       []int     `json:"ports"`
	Protocols   []string  `json:"protocols"`
	RiskScore   float64   `json:"risk_score"`
	Technique   string    `json:"technique"` // MITRE ATT&CK technique ID
}

// LateralMovementEvent is published on the event bus when a multi-hop pattern is confirmed.
type LateralMovementEvent struct {
	Chain       HopChain  `json:"chain"`
	DetectedAt  string    `json:"detected_at"`
	Severity    string    `json:"severity"`
	Description string    `json:"description"`
}

// connectionKey is the (src, dst) pair used to track per-flow state.
type connectionKey struct {
	src string
	dst string
}

type flowRecord struct {
	ports     map[int]bool
	protocols map[string]bool
	seen      string
}

// LateralMovementEngine correlates multi-hop internal connections to detect
// lateral movement. Implements MITRE ATT&CK T1021, T1210, T1570.
type LateralMovementEngine struct {
	mu           sync.RWMutex
	connections  map[connectionKey]*flowRecord
	window       time.Duration
	hopThreshold int
	bus          *eventbus.Bus
	log          *logger.Logger
}

// NewLateralMovementEngine returns a ready-to-use engine.
func NewLateralMovementEngine(bus *eventbus.Bus, log *logger.Logger) *LateralMovementEngine {
	return &LateralMovementEngine{
		connections:  make(map[connectionKey]*flowRecord),
		window:       2 * time.Minute,
		hopThreshold: 5,
		bus:          bus,
		log:          log,
	}
}

// Start begins the background GC loop.
func (e *LateralMovementEngine) Start(ctx context.Context) {
	go e.runGC(ctx)
}

// ProcessFlow ingests a network flow and checks for lateral movement patterns.
func (e *LateralMovementEngine) ProcessFlow(flow NetworkFlow) {
	// Only track flows originating from internal space
	if !isInternalIP(flow.SourceIP) {
		return
	}

	e.mu.Lock()
	k := connectionKey{src: flow.SourceIP, dst: flow.DestIP}
	rec, ok := e.connections[k]
	if !ok {
		rec = &flowRecord{
			ports:     make(map[int]bool),
			protocols: make(map[string]bool),
			seen:      flow.Timestamp,
		}
		e.connections[k] = rec
	}
	rec.ports[flow.DestPort] = true
	rec.protocols[flow.Protocol] = true
	rec.seen = flow.Timestamp
	e.mu.Unlock()

	e.evaluate(flow.SourceIP)
}

// evaluate checks if sourceIP has reached the hop threshold in the correlation window.
func (e *LateralMovementEngine) evaluate(sourceIP string) {
	cutoff := time.Now().Add(-e.window)

	e.mu.RLock()
	var hops []string
	portSet := make(map[int]bool)
	protocolSet := make(map[string]bool)

	for k, rec := range e.connections {
		if k.src != sourceIP || parseTime(rec.seen).Before(cutoff) {
			continue
		}
		if isInternalIP(k.dst) {
			hops = append(hops, k.dst)
			for p := range rec.ports {
				portSet[p] = true
			}
			for pr := range rec.protocols {
				protocolSet[pr] = true
			}
		}
	}
	e.mu.RUnlock()

	if len(hops) < e.hopThreshold {
		return
	}

	var ports []int
	var protocols []string
	for p := range portSet {
		ports = append(ports, p)
	}
	for pr := range protocolSet {
		protocols = append(protocols, pr)
	}

	// Score and map to the most-specific MITRE technique by observed ports.
	score := float64(len(hops)) / 10.0
	technique := "T1210" // Exploitation of Remote Services (default)
	for p := range portSet {
		switch p {
		case 445, 139:
			score += 0.3
			technique = "T1021.002" // SMB/Windows Admin Shares
		case 3389:
			score += 0.4
			technique = "T1021.001" // Remote Desktop Protocol
		case 22:
			score += 0.2
			technique = "T1021.004" // SSH
		case 5985, 5986:
			score += 0.3
			technique = "T1021.006" // Windows Remote Management (WinRM)
		case 135, 593:
			score += 0.25
			technique = "T1021.003" // DCOM
		}
	}
	if score > 1.0 {
		score = 1.0
	}

	severity := "HIGH"
	if score >= 0.7 {
		severity = "CRITICAL"
	}

	chain := HopChain{
		OriginIP:  sourceIP,
		Hops:      hops,
		FirstSeen: time.Now().Add(-e.window).Format(time.RFC3339),
		LastSeen:  time.Now().Format(time.RFC3339),
		Ports:     ports,
		Protocols: protocols,
		RiskScore: score,
		Technique: technique,
	}

	evt := LateralMovementEvent{
		Chain:      chain,
		DetectedAt: time.Now().Format(time.RFC3339),
		Severity:   severity,
		Description: fmt.Sprintf(
			"Source %s connected to %d internal hosts via %s in %s. MITRE %s. Risk: %.2f",
			sourceIP, len(hops), strings.Join(protocols, "/"), e.window, technique, score,
		),
	}

	e.log.Warn("[NDR:LateralMovement] %s", evt.Description)
	e.bus.Publish("ndr.lateral_movement", evt)
	e.bus.Publish("siem.alert_fired", map[string]interface{}{
		"type":        "NDR_LATERAL_MOVEMENT",
		"severity":    severity,
		"source_ip":   sourceIP,
		"description": evt.Description,
		"risk_score":  score,
		"technique":   technique,
		"hop_count":   len(hops),
	})
}

// runGC periodically evicts stale connection records outside the correlation window.
func (e *LateralMovementEngine) runGC(ctx context.Context) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			cutoff := time.Now().Add(-e.window)
			e.mu.Lock()
			for k, rec := range e.connections {
				if parseTime(rec.seen).Before(cutoff) {
					delete(e.connections, k)
				}
			}
			e.mu.Unlock()
		}
	}
}

// isInternalIP returns true if ip is in RFC-1918 private address space.
func isInternalIP(ip string) bool {
	if strings.HasPrefix(ip, "10.") || strings.HasPrefix(ip, "192.168.") {
		return true
	}
	// 172.16.0.0/12 covers 172.16.x.x – 172.31.x.x
	for i := 16; i <= 31; i++ {
		if strings.HasPrefix(ip, fmt.Sprintf("172.%d.", i)) {
			return true
		}
	}
	return false
}

func parseTime(ts string) time.Time {
	t, _ := time.Parse(time.RFC3339, ts)
	return t
}
