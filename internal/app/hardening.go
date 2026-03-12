package app

import (
	"crypto/tls"
	"fmt"
	"runtime"
	"runtime/debug"
	"time"

	"github.com/kingknull/oblivrashell/internal/logger"
)

type HardeningConfig struct {
	MaxMemoryMB        int           `json:"max_memory_mb"`
	GCPercent          int           `json:"gc_percent"`
	MaxSessionDuration time.Duration `json:"max_session_duration"`
	IdleTimeout        time.Duration `json:"idle_timeout"`
	RateLimitPerSec    int           `json:"rate_limit_per_sec"`
}

func DefaultHardeningConfig() HardeningConfig {
	return HardeningConfig{
		MaxMemoryMB:        512,
		GCPercent:          100,
		MaxSessionDuration: 24 * time.Hour,
		IdleTimeout:        30 * time.Minute,
		RateLimitPerSec:    100,
	}
}

func ApplyHardening(cfg HardeningConfig, log *logger.Logger) {
	log.Info("Applying security hardening...")

	// Memory limit
	if cfg.MaxMemoryMB > 0 {
		limit := int64(cfg.MaxMemoryMB) * 1024 * 1024
		debug.SetMemoryLimit(limit)
		log.Info("  Memory limit: %d MB", cfg.MaxMemoryMB)
	}

	// GC tuning
	if cfg.GCPercent > 0 {
		debug.SetGCPercent(cfg.GCPercent)
		log.Info("  GC percent: %d", cfg.GCPercent)
	}

	// Set GOMAXPROCS to available CPUs
	maxProcs := runtime.NumCPU()
	runtime.GOMAXPROCS(maxProcs)
	log.Info("  GOMAXPROCS: %d", maxProcs)

	log.Info("Security hardening applied")
}

// SecureTLSConfig returns a hardened TLS configuration
func SecureTLSConfig() *tls.Config {
	return &tls.Config{
		MinVersion: tls.VersionTLS13,
		CipherSuites: []uint16{
			tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305_SHA256,
			tls.TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305_SHA256,
			tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
			tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
		},
		PreferServerCipherSuites: true,
		CurvePreferences: []tls.CurveID{
			tls.X25519,
			tls.CurveP256,
		},
	}
}

// MemoryStats returns current memory statistics
func MemoryStats() map[string]interface{} {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	return map[string]interface{}{
		"alloc_mb":       float64(m.Alloc) / 1024 / 1024,
		"total_alloc_mb": float64(m.TotalAlloc) / 1024 / 1024,
		"sys_mb":         float64(m.Sys) / 1024 / 1024,
		"heap_objects":   m.HeapObjects,
		"goroutines":     runtime.NumGoroutine(),
		"gc_cycles":      m.NumGC,
		"gc_pause_ns":    m.PauseNs[(m.NumGC+255)%256],
	}
}

// PanicRecovery wraps a function with panic recovery
func PanicRecovery(log *logger.Logger, fn func()) {
	defer func() {
		if r := recover(); r != nil {
			buf := make([]byte, 4096)
			n := runtime.Stack(buf, false)
			log.Error("PANIC RECOVERED: %v\n%s", r, buf[:n])
		}
	}()
	fn()
}

// ValidateInput checks user input for safety
func ValidateInput(input string, maxLen int) error {
	if len(input) > maxLen {
		return fmt.Errorf("input exceeds max length %d", maxLen)
	}
	for _, c := range input {
		if c == 0 {
			return fmt.Errorf("null byte in input")
		}
	}
	return nil
}
