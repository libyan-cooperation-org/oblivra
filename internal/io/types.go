// Package io is OBLIVRA's pluggable input/output framework.
//
// The package gives us a Splunk-style "inputs flow into a pipeline,
// pipeline produces events into outputs" architecture, but with a
// strictly bounded scope: 5 input plugins, 5 output plugins, one
// pipeline. The same engine runs in both the agent and the server
// binaries — the agent biases toward "ship to oblivra server" output
// while the server biases toward "ingest from many syslog feeds"
// inputs, but the framework is identical.
//
// Key design decisions:
//
//  • One Event shape, no inheritance, no generics. Inputs produce
//    Events; pipeline filters/redacts; outputs consume. Every plugin
//    handles the same struct.
//
//  • Channel-based, not callback-based. Each Input writes to a chan
//    Event; the pipeline fans out to Outputs over their own channels.
//    Back-pressure handled by bounded channels — slow outputs don't
//    block fast inputs (events drop with a metric).
//
//  • Plugins implement small interfaces (~3 methods). New input/output
//    type = ~150 lines of code in inputs/<type>.go.
//
//  • Config is YAML, hot-reloaded via fsnotify. Plugin instances are
//    keyed by `id`; reload diffs and only restarts plugins whose config
//    changed.
//
//  • No reflect-based plugin discovery. The registry is hand-maintained
//    in registry.go — every supported plugin type appears exactly once
//    in two switches (one for inputs, one for outputs). Trade: zero
//    runtime surprises, every plugin is statically tree-shakeable in
//    the build.
package io

import (
	"context"
	"time"
)

// Event is the canonical shape every input produces and every output
// consumes. Keep this struct small; richer fields belong in `Fields`.
type Event struct {
	// Timestamp the source observed the event. Inputs SHOULD set this
	// from the source data; the pipeline back-fills with `time.Now()`
	// if zero.
	Timestamp time.Time `json:"timestamp"`

	// Source identifies WHERE the event came from. Format is
	// "<input-type>:<input-specifics>", e.g. "file:/var/log/auth.log"
	// or "syslog:udp/514". Operators search by this prefix.
	Source string `json:"source"`

	// Sourcetype is a stable taxonomy label (Splunk-style). Detection
	// rules and dashboards filter by this. Examples: "linux:auth",
	// "windows:eventlog:security", "cisco:asa".
	Sourcetype string `json:"sourcetype"`

	// Host is the originating host's identifier. Inputs default to the
	// local hostname; syslog/HEC inputs override from the message.
	Host string `json:"host"`

	// Raw is the unparsed message. Always populated. Outputs that
	// can pass-through (syslog forward, file sink) write Raw verbatim.
	Raw string `json:"raw"`

	// Fields holds parsed/enriched key/values. The pipeline's parse
	// step populates this; redact filters scrub it. Outputs decide
	// whether to send Fields, Raw, or both.
	Fields map[string]any `json:"fields,omitempty"`

	// InputID identifies the input plugin instance that produced the
	// event. Used for routing rules ("send only events from input X
	// to output Y") and metrics.
	InputID string `json:"input_id,omitempty"`
}

// Input is the contract every input plugin satisfies.
//
// Lifecycle:
//   1. registry.New("file", config) returns Input
//   2. pipeline calls Start(ctx, out)
//   3. plugin spawns its own goroutines, writes Events to `out`
//   4. on shutdown, ctx is cancelled and Stop() is called
type Input interface {
	// Name is the operator-supplied id (`inputs[].id` in YAML). Unique
	// per pipeline instance. Used for metrics, routing, hot-reload diff.
	Name() string

	// Type is the plugin family ("file", "syslog", "hec", ...).
	// Matches the `type` field in YAML.
	Type() string

	// Start launches the input. Should return quickly; the actual
	// work runs in goroutines spawned by Start. Events go to `out`.
	// Honour ctx for cancellation.
	Start(ctx context.Context, out chan<- Event) error

	// Stop is called after ctx is cancelled. Block until all
	// internal goroutines have drained. Bounded by a 5s timeout in
	// the pipeline — exceed that and you'll be force-killed.
	Stop() error
}

// Output is the contract every output plugin satisfies.
//
// Lifecycle:
//   1. registry.NewOutput("oblivra", config) returns Output
//   2. pipeline calls Write(ctx, event) per event
//   3. periodically pipeline calls Flush(ctx) (default 5s)
//   4. on shutdown, ctx is cancelled, Flush(ctx) once more, then Close()
type Output interface {
	Name() string
	Type() string

	// Write enqueues an event for delivery. SHOULD NOT block on I/O —
	// outputs that batch keep an internal buffer and flush in their
	// own goroutine. Returns an error only when the buffer is so full
	// the event is dropped (the pipeline records the drop in metrics).
	Write(ctx context.Context, ev Event) error

	// Flush blocks until every event currently buffered has been
	// delivered (or has hit terminal failure). Called periodically by
	// the pipeline; outputs that don't batch can no-op.
	Flush(ctx context.Context) error

	// Close drains and shuts down the output. Called once at pipeline
	// teardown. After Close, Write/Flush MUST return errors.
	Close() error
}

// PluginConfig is the per-plugin chunk of YAML — `id` and `type` plus
// type-specific fields. We unmarshal twice: once to read id+type
// generically (this struct), then again into the concrete plugin's
// own config struct.
type PluginConfig struct {
	ID   string                 `yaml:"id"`
	Type string                 `yaml:"type"`
	Raw  map[string]interface{} `yaml:",inline"`
}

// Filter is a pipeline-stage transform. Returns the (possibly
// modified) event, or false to drop. Cheap, synchronous, runs once
// per event between input and output.
type Filter func(Event) (Event, bool)
