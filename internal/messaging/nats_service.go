package messaging

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/kingknull/oblivrashell/internal/logger"
	"github.com/nats-io/nats-server/v2/server"
	"github.com/nats-io/nats.go"
)

// NATSService manages an embedded NATS server and JetStream connection.
type NATSService struct {
	server *server.Server
	conn   *nats.Conn
	js     nats.JetStreamContext
	log    *logger.Logger
	mu     sync.Mutex
	config *NATSConfig
}

type NATSConfig struct {
	Port       int
	DataDir    string
	StreamName string
	Subjects   []string
}

// NewNATSService creates a new NATS-backed messaging service.
func NewNATSService(config *NATSConfig, log *logger.Logger) *NATSService {
	return &NATSService{
		config: config,
		log:    log.WithPrefix("messaging:nats"),
	}
}

// Name returns the service identifier.
func (s *NATSService) Name() string {
	return "NATSService"
}

// Start launches the embedded NATS server and initializes JetStream.
func (s *NATSService) Start(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	opts := &server.Options{
		Port:    s.config.Port,
		StoreDir: s.config.DataDir,
		JetStream: true,
	}

	ns, err := server.NewServer(opts)
	if err != nil {
		return fmt.Errorf("failed to create NATS server: %w", err)
	}

	go ns.Start()

	if !ns.ReadyForConnections(10 * time.Second) {
		return fmt.Errorf("NATS server failed to become ready")
	}

	s.server = ns
	s.log.Info("Embedded NATS server started on port %d", s.config.Port)

	// Connect to local server
	nc, err := nats.Connect(ns.ClientURL())
	if err != nil {
		return fmt.Errorf("failed to connect to NATS: %w", err)
	}
	s.conn = nc

	// Initialize JetStream
	js, err := nc.JetStream()
	if err != nil {
		return fmt.Errorf("failed to initialize JetStream: %w", err)
	}
	s.js = js

	// Create or Update Stream
	_, err = js.AddStream(&nats.StreamConfig{
		Name:     s.config.StreamName,
		Subjects: []string{"oblivra.ingest.logs.*"},
		Storage:  nats.FileStorage,
		Retention: nats.LimitsPolicy,
		MaxMsgs:   1000000, // 1M messages in-flight
		Discard:   nats.DiscardOld,
	})
	if err != nil {
		return fmt.Errorf("failed to create JetStream stream: %w", err)
	}

	s.log.Info("JetStream initialized: %s (Subjects: %v)", s.config.StreamName, s.config.Subjects)
	return nil
}

// Stop shuts down the NATS server and connection.
func (s *NATSService) Stop(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.conn != nil {
		s.conn.Close()
	}
	if s.server != nil {
		s.server.Shutdown()
		s.server.WaitForShutdown()
	}
	return nil
}

type Priority string

const (
	PriorityCritical Priority = "critical" // Alerts, emergency signals
	PriorityHigh     Priority = "high"     // Security events, process logs
	PriorityDefault  Priority = "default"  // Standard telemetry, routine logs
)

func GetSubject(base string, p Priority) string {
	return fmt.Sprintf("%s.%s", base, p)
}

// Publish sends a message to the specified subject with a given priority.
func (s *NATSService) Publish(subject string, p Priority, data []byte) error {
	fullSubject := GetSubject(subject, p)
	_, err := s.js.Publish(fullSubject, data)
	return err
}

// Subscribe async registers a handler for a subject and priority.
func (s *NATSService) Subscribe(subject string, p Priority, handler func([]byte)) (*nats.Subscription, error) {
	fullSubject := GetSubject(subject, p)
	return s.js.Subscribe(fullSubject, func(m *nats.Msg) {
		handler(m.Data)
		m.Ack()
	}, nats.ManualAck())
}

// Conn returns the underlying NATS connection.
func (s *NATSService) Conn() *nats.Conn {
	return s.conn
}

// JS returns the JetStream context.
func (s *NATSService) JS() nats.JetStreamContext {
	return s.js
}
