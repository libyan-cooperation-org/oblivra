package listeners

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/segmentio/kafka-go"
	"github.com/segmentio/kafka-go/sasl"
	"github.com/segmentio/kafka-go/sasl/plain"
	"github.com/segmentio/kafka-go/sasl/scram"

	"github.com/kingknull/oblivra/internal/events"
	"github.com/kingknull/oblivra/internal/ingest"
	"github.com/kingknull/oblivra/internal/parsers"
)

// Kafka — pre-SIEM streaming detection upstream (DetectFlow / Kafka Streams /
// Flink / etc). One goroutine per topic; each record becomes an Event with
// Provenance.IngestPath="kafka". Records are auto-detected as JSON / RFC5424
// / RFC3164 / CEF / auditd via the existing parsers.FormatAuto path, so the
// listener doesn't care what shape the upstream pushed.
//
// Designed for the DetectFlow integration pattern: DetectFlow tags events
// in-flight with Sigma rule matches and writes back to a Kafka topic;
// OBLIVRA reads that topic and gets the enriched stream as if it had run
// the rules itself, with rule-match metadata preserved as event fields.
//
// Auth modes (decreasing convenience):
//   - PLAINTEXT   — no auth (lab / on-VPN with no Kafka ACLs)
//   - SASL/PLAIN  — username + password
//   - SASL/SCRAM  — SCRAM-SHA-256 or SCRAM-SHA-512
//   - mTLS        — client cert + CA
//
// Lossless guarantees: kafka-go reader auto-commits at the configured
// interval. We commit after Submit() returns so a crash mid-batch
// re-reads the in-flight records on restart, never loses them.

type Kafka struct {
	log      *slog.Logger
	pipeline *ingest.Pipeline
	cfg      KafkaConfig
	tenant   string

	mu      sync.Mutex
	readers []*kafka.Reader
	count   atomic.Int64
}

// KafkaConfig is the operator-facing config slice. Populate from env
// vars (see KafkaConfigFromEnv) or hand-construct in tests.
type KafkaConfig struct {
	Brokers []string // e.g. ["kafka-0:9092", "kafka-1:9092"]
	Topics  []string // one consumer goroutine per topic
	GroupID string   // consumer group; defaults to "oblivra"

	// SecurityProtocol: "PLAINTEXT" | "SSL" | "SASL_PLAINTEXT" | "SASL_SSL"
	SecurityProtocol string

	// SASLMechanism: "" | "PLAIN" | "SCRAM-SHA-256" | "SCRAM-SHA-512"
	SASLMechanism string
	SASLUsername  string
	SASLPassword  string

	// TLS — set when SecurityProtocol contains SSL.
	TLSCAFile     string // PEM CA bundle
	TLSCertFile   string // mTLS client cert (optional)
	TLSKeyFile    string // mTLS client key  (optional)
	TLSSkipVerify bool   // dev only

	// StartFromOldest — first run with an empty consumer group.
	// Default false (start from latest), so a fresh OBLIVRA doesn't
	// blow up reading days of backlog.
	StartFromOldest bool

	// CommitInterval — how often to checkpoint offset progress. Lower
	// = less duplicate replay on crash; higher = lighter load on broker.
	// Default 1s.
	CommitInterval time.Duration

	// MaxBytesPerRecord — drop records larger than this so a misbehaving
	// upstream can't run us out of memory. Default 1 MiB.
	MaxBytesPerRecord int
}

// KafkaConfigFromEnv pulls config from OBLIVRA_KAFKA_* env vars.
// Returns (nil, nil) if OBLIVRA_KAFKA_BROKERS is unset — the listener
// is opt-in.
func KafkaConfigFromEnv() (*KafkaConfig, error) {
	brokers := os.Getenv("OBLIVRA_KAFKA_BROKERS")
	if brokers == "" {
		return nil, nil
	}
	topics := os.Getenv("OBLIVRA_KAFKA_TOPICS")
	if topics == "" {
		return nil, errors.New("OBLIVRA_KAFKA_BROKERS set but OBLIVRA_KAFKA_TOPICS empty")
	}
	c := &KafkaConfig{
		Brokers:           splitCSV(brokers),
		Topics:            splitCSV(topics),
		GroupID:           getenvOr("OBLIVRA_KAFKA_GROUP", "oblivra"),
		SecurityProtocol:  strings.ToUpper(getenvOr("OBLIVRA_KAFKA_SECURITY_PROTOCOL", "PLAINTEXT")),
		SASLMechanism:     strings.ToUpper(os.Getenv("OBLIVRA_KAFKA_SASL_MECHANISM")),
		SASLUsername:      os.Getenv("OBLIVRA_KAFKA_SASL_USERNAME"),
		SASLPassword:      os.Getenv("OBLIVRA_KAFKA_SASL_PASSWORD"),
		TLSCAFile:         os.Getenv("OBLIVRA_KAFKA_TLS_CA_FILE"),
		TLSCertFile:       os.Getenv("OBLIVRA_KAFKA_TLS_CERT_FILE"),
		TLSKeyFile:        os.Getenv("OBLIVRA_KAFKA_TLS_KEY_FILE"),
		TLSSkipVerify:     os.Getenv("OBLIVRA_KAFKA_TLS_SKIP_VERIFY") == "1",
		StartFromOldest:   os.Getenv("OBLIVRA_KAFKA_START_FROM_OLDEST") == "1",
		CommitInterval:    1 * time.Second,
		MaxBytesPerRecord: 1 << 20,
	}
	return c, nil
}

func splitCSV(s string) []string {
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}

func getenvOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

// NewKafka constructs the listener. Validation happens here so a
// misconfigured cluster fails fast at boot, not on the first record.
func NewKafka(log *slog.Logger, p *ingest.Pipeline, cfg KafkaConfig, tenant string) (*Kafka, error) {
	if len(cfg.Brokers) == 0 {
		return nil, errors.New("kafka: at least one broker required")
	}
	if len(cfg.Topics) == 0 {
		return nil, errors.New("kafka: at least one topic required")
	}
	if cfg.GroupID == "" {
		cfg.GroupID = "oblivra"
	}
	if cfg.CommitInterval == 0 {
		cfg.CommitInterval = 1 * time.Second
	}
	if cfg.MaxBytesPerRecord == 0 {
		cfg.MaxBytesPerRecord = 1 << 20
	}
	if tenant == "" {
		tenant = "default"
	}
	return &Kafka{log: log, pipeline: p, cfg: cfg, tenant: tenant}, nil
}

// Start spins up one reader per topic. Returns when ctx is cancelled.
func (k *Kafka) Start(ctx context.Context) error {
	dialer, err := k.buildDialer()
	if err != nil {
		return err
	}

	startOffset := kafka.LastOffset
	if k.cfg.StartFromOldest {
		startOffset = kafka.FirstOffset
	}

	var wg sync.WaitGroup
	for _, topic := range k.cfg.Topics {
		r := kafka.NewReader(kafka.ReaderConfig{
			Brokers:        k.cfg.Brokers,
			Topic:          topic,
			GroupID:        k.cfg.GroupID,
			Dialer:         dialer,
			StartOffset:    startOffset,
			CommitInterval: k.cfg.CommitInterval,
			MinBytes:       1,
			MaxBytes:       10 << 20, // 10 MiB fetch — plenty for a SIEM stream
			MaxWait:        500 * time.Millisecond,
		})
		k.mu.Lock()
		k.readers = append(k.readers, r)
		k.mu.Unlock()

		wg.Add(1)
		go func(topic string, r *kafka.Reader) {
			defer wg.Done()
			k.consume(ctx, topic, r)
		}(topic, r)
	}

	k.log.Info("kafka listener started",
		"brokers", k.cfg.Brokers, "topics", k.cfg.Topics,
		"group", k.cfg.GroupID, "security", k.cfg.SecurityProtocol)

	wg.Wait()
	return nil
}

func (k *Kafka) consume(ctx context.Context, topic string, r *kafka.Reader) {
	defer func() {
		if err := r.Close(); err != nil {
			k.log.Warn("kafka reader close", "topic", topic, "err", err)
		}
	}()

	for {
		m, err := r.FetchMessage(ctx)
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			k.log.Warn("kafka fetch", "topic", topic, "err", err)
			// Brief backoff so a wedged broker doesn't busy-loop us.
			select {
			case <-ctx.Done():
				return
			case <-time.After(2 * time.Second):
			}
			continue
		}
		if len(m.Value) > k.cfg.MaxBytesPerRecord {
			k.log.Warn("kafka record exceeds max-bytes; dropping",
				"topic", topic, "offset", m.Offset, "size", len(m.Value))
			_ = r.CommitMessages(ctx, m)
			continue
		}
		ev := k.recordToEvent(m, topic)
		if ev == nil {
			_ = r.CommitMessages(ctx, m)
			continue
		}
		if err := k.pipeline.Submit(ctx, ev); err != nil {
			k.log.Error("kafka submit", "topic", topic, "offset", m.Offset, "err", err)
			// Don't commit on submit failure — let the reader pick it
			// back up after the broker session reset. This is the
			// at-least-once side of the trade-off.
			continue
		}
		if err := r.CommitMessages(ctx, m); err != nil {
			k.log.Warn("kafka commit", "topic", topic, "err", err)
		}
		k.count.Add(1)
	}
}

// recordToEvent turns a Kafka record into an OBLIVRA Event. We try
// parsers.FormatAuto first (covers JSON / syslog / CEF / auditd); if
// that fails we fall back to a plain-message wrapper so the operator
// still sees the bytes.
func (k *Kafka) recordToEvent(m kafka.Message, topic string) *events.Event {
	raw := string(m.Value)
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	ev, perr := parsers.Parse(raw, parsers.FormatAuto)
	if perr != nil || ev == nil {
		ev = &events.Event{
			Source:    events.SourceREST,
			Message:   raw,
			Raw:       raw,
			EventType: "kafka:" + topic,
			Severity:  events.SeverityInfo,
			Fields:    map[string]string{},
		}
	}
	ev.TenantID = k.tenant
	if ev.Fields == nil {
		ev.Fields = map[string]string{}
	}
	ev.Fields["kafkaTopic"] = topic
	ev.Fields["kafkaPartition"] = fmt.Sprintf("%d", m.Partition)
	ev.Fields["kafkaOffset"] = fmt.Sprintf("%d", m.Offset)
	if len(m.Key) > 0 {
		ev.Fields["kafkaKey"] = string(m.Key)
	}
	// Kafka headers are the natural carrier for DetectFlow's rule-match
	// tags — surface every header as a field so OQL can filter on them.
	for _, h := range m.Headers {
		ev.Fields["kafka:"+h.Key] = string(h.Value)
	}
	if !m.Time.IsZero() {
		ev.Timestamp = m.Time
	}
	ev.Provenance.IngestPath = "kafka"
	ev.Provenance.Peer = strings.Join(k.cfg.Brokers, ",")
	ev.Provenance.Format = topic
	ev.Provenance.Parser = string(parsers.FormatAuto)
	return ev
}

func (k *Kafka) buildDialer() (*kafka.Dialer, error) {
	d := &kafka.Dialer{
		Timeout:   10 * time.Second,
		DualStack: true,
	}

	if strings.Contains(k.cfg.SecurityProtocol, "SSL") {
		tlsCfg, err := k.buildTLS()
		if err != nil {
			return nil, err
		}
		d.TLS = tlsCfg
	}

	if strings.HasPrefix(k.cfg.SecurityProtocol, "SASL") {
		mech, err := k.buildSASL()
		if err != nil {
			return nil, err
		}
		d.SASLMechanism = mech
	}

	return d, nil
}

func (k *Kafka) buildTLS() (*tls.Config, error) {
	cfg := &tls.Config{InsecureSkipVerify: k.cfg.TLSSkipVerify} //nolint:gosec
	if k.cfg.TLSCAFile != "" {
		caPEM, err := os.ReadFile(k.cfg.TLSCAFile)
		if err != nil {
			return nil, fmt.Errorf("kafka tls ca: %w", err)
		}
		pool := x509.NewCertPool()
		if !pool.AppendCertsFromPEM(caPEM) {
			return nil, errors.New("kafka tls ca: no certs parsed")
		}
		cfg.RootCAs = pool
	}
	if k.cfg.TLSCertFile != "" || k.cfg.TLSKeyFile != "" {
		if k.cfg.TLSCertFile == "" || k.cfg.TLSKeyFile == "" {
			return nil, errors.New("kafka tls: cert and key must both be set for mTLS")
		}
		cert, err := tls.LoadX509KeyPair(k.cfg.TLSCertFile, k.cfg.TLSKeyFile)
		if err != nil {
			return nil, fmt.Errorf("kafka tls keypair: %w", err)
		}
		cfg.Certificates = []tls.Certificate{cert}
	}
	return cfg, nil
}

func (k *Kafka) buildSASL() (sasl.Mechanism, error) {
	if k.cfg.SASLUsername == "" || k.cfg.SASLPassword == "" {
		return nil, errors.New("kafka sasl: username and password required")
	}
	switch k.cfg.SASLMechanism {
	case "", "PLAIN":
		return plain.Mechanism{Username: k.cfg.SASLUsername, Password: k.cfg.SASLPassword}, nil
	case "SCRAM-SHA-256":
		m, err := scram.Mechanism(scram.SHA256, k.cfg.SASLUsername, k.cfg.SASLPassword)
		if err != nil {
			return nil, fmt.Errorf("kafka sasl scram-256: %w", err)
		}
		return m, nil
	case "SCRAM-SHA-512":
		m, err := scram.Mechanism(scram.SHA512, k.cfg.SASLUsername, k.cfg.SASLPassword)
		if err != nil {
			return nil, fmt.Errorf("kafka sasl scram-512: %w", err)
		}
		return m, nil
	default:
		return nil, fmt.Errorf("kafka sasl: unsupported mechanism %q (use PLAIN, SCRAM-SHA-256, SCRAM-SHA-512)", k.cfg.SASLMechanism)
	}
}

// Count returns total records ingested across all topics.
func (k *Kafka) Count() int64 { return k.count.Load() }

// Stats returns a small operator snapshot — used by /metrics.
type KafkaStats struct {
	Brokers []string `json:"brokers"`
	Topics  []string `json:"topics"`
	GroupID string   `json:"groupId"`
	Records int64    `json:"records"`
}

func (k *Kafka) Stats() KafkaStats {
	return KafkaStats{
		Brokers: k.cfg.Brokers,
		Topics:  k.cfg.Topics,
		GroupID: k.cfg.GroupID,
		Records: k.count.Load(),
	}
}

var _ = json.Marshal // reserved for future broker-side JSON probe