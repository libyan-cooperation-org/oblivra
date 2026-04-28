package io

// Stubs for not-yet-implemented input/output plugins. They satisfy
// the registry's switch arms so the package compiles incrementally;
// each will be replaced by a real implementation in its own file as
// Slice 2 / Slice 4 lands.
//
// Each constructor returns a "noop" plugin that:
//   • for Inputs: starts cleanly but produces no events
//   • for Outputs: accepts Write() but never delivers
// Plus a one-time INFO log so operators see "X plugin not yet built"
// rather than a silent no-op.

import (
	"context"
	"fmt"

	"github.com/kingknull/oblivrashell/internal/logger"
)

// ── Input stubs ──────────────────────────────────────────────────

type stubInput struct {
	id, kind string
	log      *logger.Logger
}

func (s *stubInput) Name() string { return s.id }
func (s *stubInput) Type() string { return s.kind }
func (s *stubInput) Start(ctx context.Context, out chan<- Event) error {
	s.log.Warn("input %s/%q is not yet implemented — no events will be produced", s.kind, s.id)
	return nil
}
func (s *stubInput) Stop() error { return nil }

// SyslogInput / HECInput have real implementations in input_syslog.go
// + input_hec.go. These wrappers keep the registry signatures uniform.
func NewSyslogInput(id string, raw map[string]interface{}, log *logger.Logger) (Input, error) {
	return NewSyslogInputReal(id, raw, log)
}
func NewHECInput(id string, raw map[string]interface{}, log *logger.Logger) (Input, error) {
	return NewHECInputReal(id, raw, log)
}
func NewExecInput(id string, raw map[string]interface{}, log *logger.Logger) (Input, error) {
	return &stubInput{id: id, kind: "exec", log: log.WithPrefix("input.exec")}, nil
}

// ── Output stubs ─────────────────────────────────────────────────

type stubOutput struct {
	id, kind string
	log      *logger.Logger
}

func (s *stubOutput) Name() string                       { return s.id }
func (s *stubOutput) Type() string                       { return s.kind }
func (s *stubOutput) Write(_ context.Context, _ Event) error { return nil }
func (s *stubOutput) Flush(_ context.Context) error      { return nil }
func (s *stubOutput) Close() error                       { return nil }

// All output plugins have real implementations. These wrappers keep
// the registry's switch arms calling a uniform `NewXxxOutput(...)`
// signature.
func NewSyslogOutput(id string, raw map[string]interface{}, log *logger.Logger) (Output, error) {
	return NewSyslogOutputReal(id, raw, log)
}
func NewS3Output(id string, raw map[string]interface{}, log *logger.Logger) (Output, error) {
	return NewS3OutputReal(id, raw, log)
}
func NewHTTPOutput(id string, raw map[string]interface{}, log *logger.Logger) (Output, error) {
	return NewHTTPOutputReal(id, raw, log)
}
func NewFileOutput(id string, raw map[string]interface{}, log *logger.Logger) (Output, error) {
	return NewFileOutputReal(id, raw, log)
}

// ─────────────────────────────────────────────────────────────────

// Compile-time check: every stub satisfies the interface.
var (
	_ Input  = (*stubInput)(nil)
	_ Output = (*stubOutput)(nil)
)

// Package-level compile guard for the documented behaviour above.
func init() {
	_ = fmt.Sprintf // keep fmt as a useful import for plugin extensions
}
