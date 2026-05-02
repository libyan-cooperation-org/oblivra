package listeners

import (
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/segmentio/kafka-go"
)

func TestKafkaConfigFromEnv_DisabledByDefault(t *testing.T) {
	clearKafkaEnv()
	cfg, err := KafkaConfigFromEnv()
	if err != nil {
		t.Fatal(err)
	}
	if cfg != nil {
		t.Errorf("expected nil config when OBLIVRA_KAFKA_BROKERS unset, got %+v", cfg)
	}
}

func TestKafkaConfigFromEnv_TopicsRequired(t *testing.T) {
	clearKafkaEnv()
	t.Setenv("OBLIVRA_KAFKA_BROKERS", "kafka-0:9092")
	if _, err := KafkaConfigFromEnv(); err == nil {
		t.Error("expected error when OBLIVRA_KAFKA_TOPICS missing")
	}
}

func TestKafkaConfigFromEnv_Defaults(t *testing.T) {
	clearKafkaEnv()
	t.Setenv("OBLIVRA_KAFKA_BROKERS", "kafka-0:9092 ,  kafka-1:9092")
	t.Setenv("OBLIVRA_KAFKA_TOPICS", "detectflow.matched, detectflow.raw")
	cfg, err := KafkaConfigFromEnv()
	if err != nil {
		t.Fatal(err)
	}
	if cfg == nil {
		t.Fatal("nil config")
	}
	if got, want := len(cfg.Brokers), 2; got != want {
		t.Errorf("brokers = %d, want %d", got, want)
	}
	if cfg.Brokers[0] != "kafka-0:9092" || cfg.Brokers[1] != "kafka-1:9092" {
		t.Errorf("broker trim wrong: %+v", cfg.Brokers)
	}
	if got, want := len(cfg.Topics), 2; got != want {
		t.Errorf("topics = %d, want %d", got, want)
	}
	if cfg.GroupID != "oblivra" {
		t.Errorf("default group = %q", cfg.GroupID)
	}
	if cfg.SecurityProtocol != "PLAINTEXT" {
		t.Errorf("default security = %q", cfg.SecurityProtocol)
	}
}

func TestKafkaConfigFromEnv_SASLSCRAM(t *testing.T) {
	clearKafkaEnv()
	t.Setenv("OBLIVRA_KAFKA_BROKERS", "kafka:9094")
	t.Setenv("OBLIVRA_KAFKA_TOPICS", "logs")
	t.Setenv("OBLIVRA_KAFKA_SECURITY_PROTOCOL", "sasl_ssl")
	t.Setenv("OBLIVRA_KAFKA_SASL_MECHANISM", "scram-sha-512")
	t.Setenv("OBLIVRA_KAFKA_SASL_USERNAME", "alice")
	t.Setenv("OBLIVRA_KAFKA_SASL_PASSWORD", "secret")
	cfg, err := KafkaConfigFromEnv()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.SecurityProtocol != "SASL_SSL" {
		t.Errorf("security = %q", cfg.SecurityProtocol)
	}
	if cfg.SASLMechanism != "SCRAM-SHA-512" {
		t.Errorf("mechanism = %q", cfg.SASLMechanism)
	}
}

// TestRecordToEvent_JSON: a JSON-shaped Kafka payload is auto-parsed
// and Kafka metadata lands in Fields.
func TestRecordToEvent_JSON(t *testing.T) {
	k := newTestKafka(t)
	msg := kafka.Message{
		Topic:     "detectflow.matched",
		Partition: 3,
		Offset:    42,
		Key:       []byte("web-01"),
		Time:      time.Date(2026, 5, 2, 12, 0, 0, 0, time.UTC),
		Value:     []byte(`{"hostId":"web-01","message":"sshd brute force","severity":"warning"}`),
		Headers: []kafka.Header{
			{Key: "rule.id", Value: []byte("sigma-ssh-bruteforce")},
			{Key: "rule.severity", Value: []byte("high")},
		},
	}
	ev := k.recordToEvent(msg, "detectflow.matched")
	if ev == nil {
		t.Fatal("recordToEvent returned nil for valid JSON")
	}
	if ev.HostID != "web-01" {
		t.Errorf("hostId = %q", ev.HostID)
	}
	if ev.Provenance.IngestPath != "kafka" {
		t.Errorf("ingestPath = %q", ev.Provenance.IngestPath)
	}
	if ev.Fields["kafkaTopic"] != "detectflow.matched" {
		t.Errorf("kafkaTopic = %q", ev.Fields["kafkaTopic"])
	}
	if ev.Fields["kafkaOffset"] != "42" {
		t.Errorf("kafkaOffset = %q", ev.Fields["kafkaOffset"])
	}
	if ev.Fields["kafka:rule.id"] != "sigma-ssh-bruteforce" {
		t.Errorf("rule.id header = %q", ev.Fields["kafka:rule.id"])
	}
	if !ev.Timestamp.Equal(msg.Time) {
		t.Errorf("timestamp = %v, want %v", ev.Timestamp, msg.Time)
	}
}

// TestRecordToEvent_PlainFallback: a non-parseable payload still produces
// an event with the raw bytes so we never silently drop data.
func TestRecordToEvent_PlainFallback(t *testing.T) {
	k := newTestKafka(t)
	msg := kafka.Message{
		Topic: "raw-stream",
		Value: []byte("totally-not-a-known-format payload"),
	}
	ev := k.recordToEvent(msg, "raw-stream")
	if ev == nil {
		t.Fatal("plain payload should still produce an event")
	}
	if ev.Raw != "totally-not-a-known-format payload" {
		t.Errorf("raw not preserved: %q", ev.Raw)
	}
	if ev.EventType != "kafka:raw-stream" && ev.EventType != "plain" {
		t.Errorf("eventType = %q (expected kafka:raw-stream or plain)", ev.EventType)
	}
}

// TestRecordToEvent_EmptySkipped: empty payloads are dropped (nothing to ingest).
func TestRecordToEvent_EmptySkipped(t *testing.T) {
	k := newTestKafka(t)
	msg := kafka.Message{Topic: "x", Value: []byte("   \n  ")}
	if ev := k.recordToEvent(msg, "x"); ev != nil {
		t.Errorf("expected nil for empty/whitespace payload, got %+v", ev)
	}
}

func newTestKafka(t *testing.T) *Kafka {
	t.Helper()
	return &Kafka{
		log:    slog.New(slog.NewTextHandler(os.Stderr, nil)),
		tenant: "default",
		cfg: KafkaConfig{
			Brokers: []string{"kafka:9092"},
			Topics:  []string{"detectflow.matched"},
		},
	}
}

func clearKafkaEnv() {
	for _, k := range []string{
		"OBLIVRA_KAFKA_BROKERS", "OBLIVRA_KAFKA_TOPICS", "OBLIVRA_KAFKA_GROUP",
		"OBLIVRA_KAFKA_SECURITY_PROTOCOL", "OBLIVRA_KAFKA_SASL_MECHANISM",
		"OBLIVRA_KAFKA_SASL_USERNAME", "OBLIVRA_KAFKA_SASL_PASSWORD",
		"OBLIVRA_KAFKA_TLS_CA_FILE", "OBLIVRA_KAFKA_TLS_CERT_FILE",
		"OBLIVRA_KAFKA_TLS_KEY_FILE", "OBLIVRA_KAFKA_TLS_SKIP_VERIFY",
		"OBLIVRA_KAFKA_START_FROM_OLDEST",
	} {
		_ = os.Unsetenv(k)
	}
}
