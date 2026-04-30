// oblivra-agent — log-tailing agent that watches files (or stdin) and forwards
// events to an OBLIVRA server in batches.
//
// Usage:
//
//	oblivra-agent --server http://oblivra:8080 --token agent-key \
//	              --hostname web-01 \
//	              --tail /var/log/auth.log --tail /var/log/syslog
//
// If no --tail is given, the agent reads from stdin (one line per event).
// On network errors, lines are kept in a local on-disk buffer and replayed
// at the next successful round-trip.
package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"
)

type stringSlice []string

func (s *stringSlice) String() string     { return strings.Join(*s, ",") }
func (s *stringSlice) Set(v string) error { *s = append(*s, v); return nil }

func main() {
	var (
		server   = flag.String("server", "http://localhost:8080", "server URL")
		token    = flag.String("token", os.Getenv("OBLIVRA_TOKEN"), "API key")
		hostname = flag.String("hostname", hostnameDefault(), "host id to tag every event with")
		bufDir   = flag.String("buffer", filepath.Join(os.TempDir(), "oblivra-agent"), "on-disk retry buffer dir")
		batch    = flag.Int("batch", 100, "max events per batch")
		flush    = flag.Duration("flush", 2*time.Second, "max time before flushing a partial batch")
		tags     stringSlice
		paths    stringSlice
	)
	flag.Var(&tags, "tag", "additional tag (repeatable)")
	flag.Var(&paths, "tail", "file to tail (repeatable). If empty, reads stdin.")
	flag.Parse()

	if err := os.MkdirAll(*bufDir, 0o755); err != nil {
		log.Fatalf("buffer dir: %v", err)
	}

	a := &agent{
		server:   strings.TrimRight(*server, "/"),
		token:    *token,
		hostname: *hostname,
		tags:     tags,
		batch:    *batch,
		flush:    *flush,
		bufDir:   *bufDir,
		queue:    make(chan string, 8192),
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	go a.run(ctx)

	if len(paths) == 0 {
		log.Printf("oblivra-agent: tailing stdin → %s", a.server)
		a.tailReader(ctx, os.Stdin, "stdin")
	} else {
		for _, p := range paths {
			go a.tailFile(ctx, p)
		}
		log.Printf("oblivra-agent: tailing %d files → %s", len(paths), a.server)
		<-ctx.Done()
	}
	log.Printf("oblivra-agent: shutdown")
}

type agent struct {
	server   string
	token    string
	hostname string
	tags     []string
	batch    int
	flush    time.Duration
	bufDir   string

	queue chan string
}

func (a *agent) tailFile(ctx context.Context, path string) {
	for ctx.Err() == nil {
		f, err := os.Open(path)
		if err != nil {
			log.Printf("tail %s: %v", path, err)
			select {
			case <-ctx.Done():
				return
			case <-time.After(2 * time.Second):
			}
			continue
		}
		_, _ = f.Seek(0, io.SeekEnd) // start at tail like `tail -F`
		a.tailReader(ctx, f, path)
		_ = f.Close()
	}
}

func (a *agent) tailReader(ctx context.Context, r io.Reader, source string) {
	br := bufio.NewReader(r)
	for {
		if ctx.Err() != nil {
			return
		}
		line, err := br.ReadString('\n')
		line = strings.TrimRight(line, "\r\n")
		if line != "" {
			a.enqueue(packLine(a.hostname, source, a.tags, line))
		}
		if err != nil {
			if errors.Is(err, io.EOF) {
				select {
				case <-ctx.Done():
					return
				case <-time.After(200 * time.Millisecond):
				}
				continue
			}
			log.Printf("tail %s: %v", source, err)
			return
		}
	}
}

func (a *agent) enqueue(ev string) {
	select {
	case a.queue <- ev:
	default:
		// queue full — drop oldest
		select {
		case <-a.queue:
		default:
		}
		a.queue <- ev
	}
}

func packLine(host, source string, tags []string, raw string) string {
	fields := map[string]string{"agentSource": source}
	if len(tags) > 0 {
		fields["tags"] = strings.Join(tags, ",")
	}
	doc := map[string]any{
		"source":    "agent",
		"hostId":    host,
		"message":   raw,
		"raw":       raw,
		"eventType": "tail",
		"fields":    fields,
	}
	b, _ := json.Marshal(doc)
	return string(b)
}

func (a *agent) run(ctx context.Context) {
	timer := time.NewTimer(a.flush)
	defer timer.Stop()
	var pending []string

	send := func() {
		if len(pending) == 0 {
			return
		}
		if err := a.flushBatch(ctx, pending); err != nil {
			a.spillToDisk(pending)
			log.Printf("send failed (%d events buffered to disk): %v", len(pending), err)
		} else {
			a.replaySpilled(ctx)
		}
		pending = pending[:0]
	}

	for {
		select {
		case <-ctx.Done():
			send()
			return
		case ev := <-a.queue:
			pending = append(pending, ev)
			if len(pending) >= a.batch {
				send()
				if !timer.Stop() {
					<-timer.C
				}
				timer.Reset(a.flush)
			}
		case <-timer.C:
			send()
			timer.Reset(a.flush)
		}
	}
}

func (a *agent) flushBatch(ctx context.Context, items []string) error {
	if len(items) == 0 {
		return nil
	}
	body := "[" + strings.Join(items, ",") + "]"
	req, err := http.NewRequestWithContext(ctx, "POST", a.server+"/api/v1/siem/ingest/batch", bytes.NewReader([]byte(body)))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	if a.token != "" {
		req.Header.Set("Authorization", "Bearer "+a.token)
	}
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		b, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("server %s: %s", resp.Status, string(b))
	}
	return nil
}

func (a *agent) spillToDisk(items []string) {
	name := filepath.Join(a.bufDir, fmt.Sprintf("spill-%d.jsonl", time.Now().UnixNano()))
	f, err := os.Create(name)
	if err != nil {
		log.Printf("spill: %v", err)
		return
	}
	for _, it := range items {
		fmt.Fprintln(f, it)
	}
	_ = f.Close()
}

func (a *agent) replaySpilled(ctx context.Context) {
	entries, err := os.ReadDir(a.bufDir)
	if err != nil {
		return
	}
	for _, e := range entries {
		if e.IsDir() || !strings.HasPrefix(e.Name(), "spill-") {
			continue
		}
		path := filepath.Join(a.bufDir, e.Name())
		b, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		lines := strings.Split(strings.TrimSpace(string(b)), "\n")
		if err := a.flushBatch(ctx, lines); err != nil {
			return
		}
		_ = os.Remove(path)
		log.Printf("replayed spill %s (%d events)", e.Name(), len(lines))
	}
}

func hostnameDefault() string {
	h, err := os.Hostname()
	if err != nil {
		return "unknown"
	}
	return h
}
