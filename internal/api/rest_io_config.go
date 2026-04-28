package api

// I/O config REST endpoints (Slice 5).
//
//   GET  /api/v1/io/config       — return the current YAML as JSON
//   PUT  /api/v1/io/config       — replace the entire config (validates
//                                  before writing; triggers hot-reload
//                                  via the watcher)
//   POST /api/v1/io/test         — body: { "type", "config" } → returns
//                                  { ok, error } describing whether the
//                                  config validates without committing
//   GET  /api/v1/io/stats        — pipeline counters (events_in, _out,
//                                  _drop)
//
// The /connectors UI page reads/writes via these endpoints. The
// concrete pipeline lives in main.go alongside the rest of the
// services; we accept an interface here to dodge the api ↔ services
// import cycle.

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"

	"github.com/kingknull/oblivrashell/internal/auth"
	"gopkg.in/yaml.v3"
)

// IOConfigProvider is the surface the api package needs. The concrete
// impl lives next to main.go where the io.Pipeline is constructed.
type IOConfigProvider interface {
	// ReadFile returns the current on-disk YAML content. Operators may
	// have hand-edited the file outside the UI — we always present the
	// current truth, not a cached version.
	ReadFile() ([]byte, error)
	// WriteFile replaces the on-disk YAML. Validates by re-parsing
	// before persisting; returns an error on syntax / semantic
	// failures so the UI can show them to the operator without
	// committing a broken config.
	WriteFile(yaml []byte) error
	// PipelineStats returns the (in, out, drop) counters from the
	// running pipeline for the /api/v1/io/stats endpoint.
	PipelineStats() (uint64, uint64, uint64)
}

// SetIOConfig wires the provider. Same setter pattern as
// SetSuppression / SetSettings to dodge import cycles.
func (s *RESTServer) SetIOConfig(p IOConfigProvider) {
	s.ioConfig = p
}

func (s *RESTServer) handleIOConfig(w http.ResponseWriter, r *http.Request) {
	if s.ioConfig == nil {
		http.Error(w, "I/O config provider not available", http.StatusServiceUnavailable)
		return
	}
	role := auth.GetRole(r.Context())
	switch r.Method {
	case http.MethodGet:
		// Reads are admin-only — the YAML can hold tokens / fleet
		// secrets / TLS material, even though we env-substitute.
		if role != auth.RoleAdmin {
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		}
		body, err := s.ioConfig.ReadFile()
		if err != nil {
			if os.IsNotExist(err) {
				// First-run — return an empty skeleton so the UI can
				// render the "add input" form.
				s.jsonResponse(w, http.StatusOK, map[string]any{
					"yaml": "tls:\n  mode: on\ninputs: []\noutputs: []\n",
					"new":  true,
				})
				return
			}
			http.Error(w, "read: "+err.Error(), http.StatusInternalServerError)
			return
		}
		s.jsonResponse(w, http.StatusOK, map[string]any{
			"yaml": string(body),
			"new":  false,
		})

	case http.MethodPut:
		if role != auth.RoleAdmin {
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		}
		r.Body = http.MaxBytesReader(w, r.Body, 1<<20) // 1 MiB
		var body struct {
			YAML string `json:"yaml"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			http.Error(w, "invalid JSON: "+err.Error(), http.StatusBadRequest)
			return
		}
		if body.YAML == "" {
			http.Error(w, "yaml is required", http.StatusBadRequest)
			return
		}
		// Validate before writing — catch typos at the UI surface,
		// not at runtime when the watcher tries to apply.
		var probe struct {
			TLS     map[string]any   `yaml:"tls"`
			Inputs  []map[string]any `yaml:"inputs"`
			Outputs []map[string]any `yaml:"outputs"`
		}
		if err := yaml.Unmarshal([]byte(body.YAML), &probe); err != nil {
			http.Error(w, "yaml parse: "+err.Error(), http.StatusBadRequest)
			return
		}
		if err := s.ioConfig.WriteFile([]byte(body.YAML)); err != nil {
			http.Error(w, "write: "+err.Error(), http.StatusInternalServerError)
			return
		}
		s.appendAuditEntry(connectorActor(r), "io.config.update",
			"config",
			fmt.Sprintf("inputs=%d outputs=%d", len(probe.Inputs), len(probe.Outputs)),
			r,
		)
		s.jsonResponse(w, http.StatusOK, map[string]any{
			"ok":      true,
			"inputs":  len(probe.Inputs),
			"outputs": len(probe.Outputs),
		})

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *RESTServer) handleIOStats(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	role := auth.GetRole(r.Context())
	if role != auth.RoleAnalyst && role != auth.RoleAdmin && role != auth.RoleReadOnly {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}
	in, out, drop := uint64(0), uint64(0), uint64(0)
	if s.ioConfig != nil {
		in, out, drop = s.ioConfig.PipelineStats()
	}
	s.jsonResponse(w, http.StatusOK, map[string]any{
		"events_in":   in,
		"events_out":  out,
		"events_drop": drop,
	})
}

func (s *RESTServer) handleIOTest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	role := auth.GetRole(r.Context())
	if role != auth.RoleAdmin {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 64*1024)
	var body struct {
		YAML string `json:"yaml"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "invalid JSON: "+err.Error(), http.StatusBadRequest)
		return
	}
	var probe struct {
		Inputs  []map[string]any `yaml:"inputs"`
		Outputs []map[string]any `yaml:"outputs"`
	}
	if err := yaml.Unmarshal([]byte(body.YAML), &probe); err != nil {
		s.jsonResponse(w, http.StatusOK, map[string]any{
			"ok":    false,
			"error": "parse: " + err.Error(),
		})
		return
	}
	s.jsonResponse(w, http.StatusOK, map[string]any{
		"ok":      true,
		"inputs":  len(probe.Inputs),
		"outputs": len(probe.Outputs),
	})
}
