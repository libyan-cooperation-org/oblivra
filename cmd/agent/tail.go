package main

import (
	"bufio"
	"compress/gzip"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync/atomic"
	"time"
)

// Tailer watches one file (or stdin) and emits each event into the queue.
// Multiline events are stitched if a startPattern is configured. Position
// is checkpointed on every flush so a restart resumes where we left off.
type Tailer struct {
	in       Input
	hostname string
	tenant   string
	tags     []string
	stateDir string
	redact   bool
	pos      *PositionStore
	queue    chan<- string
	hiQueue  chan<- string // optional priority queue for high-severity events
	mlOpts   MultilineOpts

	includeRE *regexp.Regexp
	excludeRE *regexp.Regexp
	extracts  []compiledExtract

	signer *Signer
	rules  []LocalRule
}

type compiledExtract struct {
	name string
	re   *regexp.Regexp
}

// TailerDeps bundles the optional pieces (signer, priority queue, local rules).
type TailerDeps struct {
	Queue     chan<- string
	HiQueue   chan<- string
	Signer    *Signer
	Rules     []LocalRule
}

func NewTailer(c *Config, in Input, deps TailerDeps, p *PositionStore) (*Tailer, error) {
	t := &Tailer{
		in: in, hostname: c.Hostname, tenant: c.Tenant, tags: c.Tags,
		stateDir: c.StateDir,
		redact:   c.Redact,
		pos:      p, queue: deps.Queue, hiQueue: deps.HiQueue,
		signer: deps.Signer, rules: deps.Rules,
	}
	if in.Multiline != nil {
		t.mlOpts = *in.Multiline
	} else {
		t.mlOpts = c.Multiline
	}
	if in.IncludeOnly != "" {
		re, err := regexp.Compile(in.IncludeOnly)
		if err != nil {
			return nil, err
		}
		t.includeRE = re
	}
	if in.Exclude != "" {
		re, err := regexp.Compile(in.Exclude)
		if err != nil {
			return nil, err
		}
		t.excludeRE = re
	}
	for _, ex := range in.Extract {
		re, err := regexp.Compile(ex.Regex)
		if err != nil {
			return nil, fmt.Errorf("extract %q: %w", ex.Name, err)
		}
		t.extracts = append(t.extracts, compiledExtract{name: ex.Name, re: re})
	}
	return t, nil
}

func (t *Tailer) Run(ctx context.Context) error {
	switch t.in.Type {
	case "file":
		return t.runFile(ctx)
	case "stdin":
		return t.runStdin(ctx)
	case "journald":
		return t.runJournald(ctx)
	default:
		return errors.New("unsupported input type: " + t.in.Type)
	}
}

func (t *Tailer) runStdin(ctx context.Context) error {
	br := bufio.NewReader(os.Stdin)
	t.feed(ctx, br, "stdin")
	return nil
}

func (t *Tailer) runFile(ctx context.Context) error {
	// Glob the path so a single config entry can cover rolled files like
	// `/var/log/nginx/access*.log`.
	candidates, err := filepath.Glob(t.in.Path)
	if err != nil || len(candidates) == 0 {
		// Glob returns no error when nothing matches — fall back to literal.
		candidates = []string{t.in.Path}
	}

	// Day-zero backfill: when StartFrom == "beginning", also scoop up
	// the rotated archive set (auth.log.1, auth.log.2.gz, …) — we only
	// do this on the first run, then drop down to live tailing.
	if t.in.StartFrom == "beginning" {
		archives := discoverRotatedArchives(t.in.Path)
		// Read archives oldest-first so the resulting event stream is
		// chronological. Archives are read once then forgotten — they
		// don't move, so we don't need a tail loop.
		for _, a := range archives {
			if ctx.Err() != nil {
				return nil
			}
			t.replayArchive(ctx, a)
		}
	}

	for _, path := range candidates {
		go t.runOneFile(ctx, path)
	}
	<-ctx.Done()
	return nil
}

// discoverRotatedArchives finds older copies of the live file: the
// .1 / .2 / .gz / .bz2 / .zst variants logrotate produces. Returned
// list is sorted oldest-first so the replay event stream is in time
// order. Best-effort — no error on missing archives.
func discoverRotatedArchives(livePath string) []string {
	dir, base := filepath.Split(livePath)
	if dir == "" {
		dir = "."
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	var matches []string
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if name == base {
			continue // the live file is handled by the tailer loop
		}
		// Match: <base>.<N>, <base>.<N>.gz, <base>.<N>.bz2, <base>.<N>.zst
		// Also: <base>.gz / .bz2 / .zst (single-archive rotation).
		// We accept anything that starts with base + "."
		if !strings.HasPrefix(name, base+".") {
			continue
		}
		matches = append(matches, filepath.Join(dir, name))
	}
	// Sort by mtime ascending so older content is replayed first.
	type withTime struct {
		path string
		mod  time.Time
	}
	withTimes := make([]withTime, 0, len(matches))
	for _, m := range matches {
		st, err := os.Stat(m)
		if err != nil {
			continue
		}
		withTimes = append(withTimes, withTime{m, st.ModTime()})
	}
	sort.Slice(withTimes, func(i, j int) bool { return withTimes[i].mod.Before(withTimes[j].mod) })
	out := make([]string, len(withTimes))
	for i, w := range withTimes {
		out[i] = w.path
	}
	return out
}

// replayArchive reads a (possibly compressed) rotated log file once
// through the standard feed() pipeline. Compression: .gz only — bz2
// and zst would need extra deps; an operator who needs them can
// gunzip first or extend this function.
func (t *Tailer) replayArchive(ctx context.Context, path string) {
	f, err := os.Open(path)
	if err != nil {
		log.Printf("backfill open %s: %v", path, err)
		return
	}
	defer f.Close()

	var r io.Reader = f
	if strings.HasSuffix(path, ".gz") {
		gz, err := gzip.NewReader(f)
		if err != nil {
			log.Printf("backfill gunzip %s: %v", path, err)
			return
		}
		defer gz.Close()
		r = gz
	} else if strings.HasSuffix(path, ".bz2") || strings.HasSuffix(path, ".zst") {
		log.Printf("backfill skip %s: %s archives unsupported (operator: decompress first)",
			path, strings.TrimPrefix(filepath.Ext(path), "."))
		return
	}

	log.Printf("backfill: replaying %s", path)
	t.feed(ctx, bufio.NewReader(r), path)
}

func (t *Tailer) runOneFile(ctx context.Context, path string) {
	for ctx.Err() == nil {
		f, err := os.Open(path)
		if err != nil {
			log.Printf("tail %s: %v", path, err)
			sleep(ctx, 2*time.Second)
			continue
		}

		// Determine where to start: stored offset, or tail/beginning per config.
		startAt := int64(0)
		info, _ := f.Stat()
		if pos, ok := t.pos.Get(path); ok && pos.Size > 0 {
			if info != nil && info.Size() < pos.Size {
				// File shrunk — assume rotation, restart from 0.
				log.Printf("tail %s: rotation detected (%d → %d bytes), re-tailing from start",
					path, pos.Size, info.Size())
				startAt = 0
			} else {
				startAt = pos.Off
			}
		} else if t.in.StartFrom != "beginning" && info != nil {
			startAt = info.Size()
		}
		if _, err := f.Seek(startAt, io.SeekStart); err != nil {
			log.Printf("tail %s seek: %v", path, err)
			f.Close()
			sleep(ctx, 2*time.Second)
			continue
		}

		t.feed(ctx, bufio.NewReader(f), path)
		offset, _ := f.Seek(0, io.SeekCurrent)
		fi, _ := f.Stat()
		var size int64
		if fi != nil {
			size = fi.Size()
		}
		_ = t.pos.Set(Position{Path: path, Off: offset, Size: size})
		f.Close()
	}
}

// feed reads from r and emits one event per logical line (or per multiline
// block). Returns when ctx is cancelled or r returns a non-recoverable error.
func (t *Tailer) feed(ctx context.Context, r *bufio.Reader, source string) {
	var (
		buffer        []string
		bufferStartTs time.Time
	)

	emit := func() {
		if len(buffer) == 0 {
			return
		}
		full := strings.Join(buffer, "\n")
		buffer = buffer[:0]
		t.enqueue(source, full)
	}

	flushTimer := time.NewTimer(t.mlOpts.Timeout)
	defer flushTimer.Stop()
	if !flushTimer.Stop() {
		<-flushTimer.C
	}

	for {
		if ctx.Err() != nil {
			emit()
			return
		}
		line, err := r.ReadString('\n')
		line = strings.TrimRight(line, "\r\n")
		if line != "" {
			if t.mlOpts.StartPattern != "" {
				re, _ := cachedMLRegex(t.mlOpts.StartPattern)
				if re != nil && re.MatchString(line) {
					emit()
					bufferStartTs = time.Now()
				}
				buffer = append(buffer, line)
				if len(buffer) >= t.mlOpts.MaxLines {
					emit()
				}
				if !flushTimer.Stop() {
					select {
					case <-flushTimer.C:
					default:
					}
				}
				flushTimer.Reset(t.mlOpts.Timeout)
				_ = bufferStartTs
			} else {
				t.enqueue(source, line)
			}
		}
		if err != nil {
			if errors.Is(err, io.EOF) {
				select {
				case <-ctx.Done():
					emit()
					return
				case <-time.After(200 * time.Millisecond):
					continue
				case <-flushTimer.C:
					emit()
					continue
				}
			}
			return
		}
	}
}

func (t *Tailer) enqueue(source, raw string) {
	if t.includeRE != nil && !t.includeRE.MatchString(raw) {
		return
	}
	if t.excludeRE != nil && t.excludeRE.MatchString(raw) {
		return
	}

	// Edge DLP — apply BEFORE rule scoring, sigma matching, signing,
	// or wire egress. Conservative: if a known canary survives the
	// regex pass (e.g. an unusual private-key PEM block format) we
	// drop the line entirely rather than risk a leak.
	var dlpHits []string
	if t.redact {
		raw, dlpHits = redactLine(raw)
		if stillHasSecrets(raw) {
			droppedEvents.Add(1)
			log.Printf("dlp drop: leaked-canary-detected from %s; line redacted to nothing", source)
			return
		}
	}
	fields := map[string]string{
		"agentSource": source,
		"agentInput":  t.in.Label,
	}
	if len(dlpHits) > 0 {
		fields["agentRedacted"] = strings.Join(dlpHits, ",")
	}
	for k, v := range t.in.Fields {
		fields[k] = v
	}
	if t.in.SourceType != "" {
		fields["sourceType"] = t.in.SourceType
	}
	if len(t.tags) > 0 {
		fields["tags"] = strings.Join(t.tags, ",")
	}
	if t.signer != nil {
		fields["agentKeyId"] = t.signer.FingerprintShort()
	}

	// Edge-side regex extraction — first matching rule wins; named groups
	// become event fields. Saves the platform a re-extraction pass and
	// makes downstream OQL queries cheaper.
	for _, ex := range t.extracts {
		m := ex.re.FindStringSubmatch(raw)
		if m == nil {
			continue
		}
		names := ex.re.SubexpNames()
		for i, val := range m {
			if i == 0 || i >= len(names) || names[i] == "" {
				continue
			}
			if _, exists := fields[names[i]]; !exists {
				fields[names[i]] = val
			}
		}
		fields["agentExtract"] = ex.name
		break
	}

	// Local pre-detection — events that match high-severity rules go to the
	// priority queue so they ship first under backpressure.
	score := 0
	if len(t.rules) > 0 {
		score = ScoreLine(raw, t.rules)
		if score > 0 {
			fields["localRuleSeverity"] = severityLabel(score)
		}
	}

	doc := map[string]any{
		"source":    "agent",
		"tenantId":  t.tenant,
		"hostId":    t.in.HostID,
		"message":   raw,
		"raw":       raw,
		"eventType": orFallback(t.in.SourceType, "tail"),
		"fields":    fields,
	}
	b, _ := json.Marshal(doc)
	out := string(b)

	// Sign at the edge — appends agentSig + agentKeyId to the JSON.
	if t.signer != nil {
		signed, err := t.signer.SignEvent(out)
		if err == nil {
			out = signed
		} else {
			log.Printf("tailer sign: %v", err)
		}
	}

	target := t.queue
	if score >= 3 && t.hiQueue != nil {
		target = t.hiQueue
	}
	select {
	case target <- out:
	default:
		droppedEvents.Add(1)
		log.Printf("tailer queue full; dropping event from %s", source)
	}
}

// droppedEvents counts events the tailer had to drop because both the
// primary and high-severity queues were full. Surfaced in the
// heartbeat so the operator can spot agents that need a bigger queue
// or a faster downstream.
var droppedEvents atomic.Int64

func severityLabel(s int) string {
	switch s {
	case 4:
		return "critical"
	case 3:
		return "high"
	case 2:
		return "medium"
	default:
		return "low"
	}
}

func orFallback(s, def string) string {
	if s == "" {
		return def
	}
	return s
}

// cachedMLRegex avoids recompiling the multiline pattern per event.
var (
	mlRegexCache = map[string]*regexp.Regexp{}
)

func cachedMLRegex(pat string) (*regexp.Regexp, error) {
	if re, ok := mlRegexCache[pat]; ok {
		return re, nil
	}
	re, err := regexp.Compile(pat)
	if err != nil {
		return nil, err
	}
	mlRegexCache[pat] = re
	return re, nil
}

func sleep(ctx context.Context, d time.Duration) {
	select {
	case <-ctx.Done():
	case <-time.After(d):
	}
}

// hostInfo helpers for `agent status` rendering.
func hostUname() string { return runtime.GOOS + "/" + runtime.GOARCH + " go" + runtime.Version() }
func pid() string        { return strconv.Itoa(os.Getpid()) }
