package tamper

// Oplog shipper — tails the agent's own log file, batches lines, and
// POSTs to /api/v1/agent/oplog with HMAC fleet-secret auth.
//
// Why tail-from-disk instead of intercept-at-logger: the agent's
// existing logger.Logger writes to a file. Tailing that file means
// we ship EXACTLY what the operator sees in the local log, including
// lines from third-party libraries that don't go through our own
// logger. Reverse direction (intercept at logger) misses anything
// emitted via stdlib `log` or panic stacks.
//
// Behaviour matches the file input in internal/io/input_file.go but
// trimmed: single fixed path, no globbing, position survives in-place
// rotation via inode tracking.

import (
	"bufio"
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"crypto/tls"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

type oplogLine struct {
	TS       string `json:"ts"`
	Level    string `json:"level"`
	Source   string `json:"source"`
	Message  string `json:"message"`
	PrevHash string `json:"prev_hash"`
}

type oplogBatch struct {
	AgentID  string      `json:"agent_id"`
	BatchSeq int64       `json:"batch_seq"`
	Lines    []oplogLine `json:"lines"`
}

func (s *Subsystem) runOplogShipper(ctx context.Context) {
	defer s.wg.Done()

	// Per-shipper HTTP client. Long-lived; respects the verify_tls
	// config knob.
	client := &http.Client{
		Timeout: 30 * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: !s.cfg.VerifyTLS},
		},
	}

	// In-flight batch buffer + flush trigger.
	var (
		mu      sync.Mutex
		buffer  = make([]oplogLine, 0, s.cfg.OplogBatchSize)
		flushCh = make(chan struct{}, 4)
	)

	// Reader goroutine: tail the log file, push lines into buffer.
	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		s.tailLogFile(ctx, func(level, msg string) {
			head := s.chain.Commit([]byte(msg))
			mu.Lock()
			buffer = append(buffer, oplogLine{
				TS:       time.Now().UTC().Format(time.RFC3339Nano),
				Level:    level,
				Source:   "agent",
				Message:  msg,
				PrevHash: head,
			})
			full := len(buffer) >= s.cfg.OplogBatchSize
			mu.Unlock()
			if full {
				select {
				case flushCh <- struct{}{}:
				default:
				}
			}
		})
	}()

	flush := func() {
		mu.Lock()
		if len(buffer) == 0 {
			mu.Unlock()
			return
		}
		batch := buffer
		buffer = make([]oplogLine, 0, s.cfg.OplogBatchSize)
		mu.Unlock()

		if err := s.postOplog(ctx, client, batch); err != nil {
			// Restore the batch oldest-first and drop the failure
			// signal — next flush will retry. Bounded by the
			// reader's cap (it stops adding when buffer is full).
			s.log.Debug("oplog post failed (%d lines): %v", len(batch), err)
			mu.Lock()
			buffer = append(batch, buffer...)
			mu.Unlock()
		}
	}

	t := time.NewTicker(s.cfg.OplogBatchTimeout)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			// Final flush attempt before exit.
			flushCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			_ = s.postOplog(flushCtx, client, drainBuffer(&mu, &buffer))
			return
		case <-t.C:
			flush()
		case <-flushCh:
			flush()
		}
	}
}

// drainBuffer atomically empties the buffer and returns the contents.
// Used at shutdown for the final flush attempt.
func drainBuffer(mu *sync.Mutex, buffer *[]oplogLine) []oplogLine {
	mu.Lock()
	defer mu.Unlock()
	out := *buffer
	*buffer = nil
	return out
}

// postOplog signs the batch with HMAC fleet-secret + timestamp and
// POSTs it to /api/v1/agent/oplog. Same signature shape as the
// existing agent → server ingest path so server-side verification is
// uniform.
func (s *Subsystem) postOplog(ctx context.Context, client *http.Client, lines []oplogLine) error {
	if len(lines) == 0 {
		return nil
	}
	body, err := json.Marshal(oplogBatch{
		AgentID:  s.cfg.AgentID,
		BatchSeq: s.nextBatchSeq(),
		Lines:    lines,
	})
	if err != nil {
		return err
	}
	// Match the canonical agent → server HMAC format used by
	// /api/v1/agent/ingest: X-Timestamp (Unix epoch seconds) +
	// X-Signature, MAC over body || timestamp. See
	// internal/api/middleware.go::VerifyHMAC for the verifier.
	ts := strconv.FormatInt(time.Now().Unix(), 10)
	mac := hmac.New(sha256.New, s.cfg.FleetSecret)
	mac.Write(body)
	mac.Write([]byte(ts))
	sig := hex.EncodeToString(mac.Sum(nil))

	url := strings.TrimRight(s.cfg.ServerURL, "/") + "/api/v1/agent/oplog"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Timestamp", ts)
	req.Header.Set("X-Signature", sig)
	req.Header.Set("X-Agent-ID", s.cfg.AgentID)

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode/100 != 2 {
		return &httpError{Code: resp.StatusCode}
	}
	return nil
}

// tailLogFile is a stripped-down version of the file input — single
// path, line-by-line, calls `emit(level, message)` for each new line.
// Honours rotation via inode change. Sleeps briefly when the file is
// unavailable so a missing log doesn't hot-loop.
//
// Levels are extracted by scanning for the standard logger prefix
// emitted by internal/logger (e.g. "INF", "WRN", "ERR"). On no match
// we default to "INF".
func (s *Subsystem) tailLogFile(ctx context.Context, emit func(level, message string)) {
	var (
		curInode uint64
		curPos   int64
	)

	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		fh, err := os.Open(s.cfg.LogPath)
		if err != nil {
			select {
			case <-ctx.Done():
				return
			case <-time.After(5 * time.Second):
			}
			continue
		}

		st, err := fh.Stat()
		if err != nil {
			fh.Close()
			return
		}
		newInode := statInode(st)
		if curInode == 0 {
			curInode = newInode
			// Start at end on first attach — operators don't want a
			// flood of historical lines on agent restart.
			curPos, _ = fh.Seek(0, 2)
		} else if newInode != curInode {
			// Rotated.
			curInode = newInode
			curPos = 0
		}
		_, _ = fh.Seek(curPos, 0)
		reader := bufio.NewReader(fh)

		for {
			select {
			case <-ctx.Done():
				fh.Close()
				return
			default:
			}
			line, err := reader.ReadString('\n')
			if err != nil {
				curPos, _ = fh.Seek(0, 1)
				break
			}
			line = strings.TrimRight(line, "\r\n")
			if line == "" {
				continue
			}
			level := extractLevel(line)
			emit(level, line)
		}
		fh.Close()

		select {
		case <-ctx.Done():
			return
		case <-time.After(500 * time.Millisecond):
		}
	}
}

// extractLevel scans a line for the canonical logger level marker and
// returns one of INF / WRN / ERR / DBG. The logger writes lines like
// "2026-04-28 17:32:01.234 INF [PREFIX] message" — we look for the
// 3-letter token in the first 32 bytes.
func extractLevel(line string) string {
	if len(line) > 32 {
		line = line[:32]
	}
	for _, lvl := range []string{"ERR", "WRN", "WAR", "INF", "DBG", "TRC"} {
		if strings.Contains(line, " "+lvl+" ") {
			if lvl == "WAR" {
				return "WRN"
			}
			return lvl
		}
	}
	return "INF"
}

// httpError lets us distinguish HTTP-status failures from network
// failures in retry logic.
type httpError struct{ Code int }

func (e *httpError) Error() string { return "http " + httpStatus(e.Code) }
func httpStatus(c int) string {
	return statusCodes[c]
}

var statusCodes = map[int]string{
	400: "400 bad request",
	401: "401 unauthorized",
	403: "403 forbidden",
	404: "404 not found",
	500: "500 internal server error",
	502: "502 bad gateway",
	503: "503 service unavailable",
}
