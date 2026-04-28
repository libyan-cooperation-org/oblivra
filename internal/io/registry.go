package io

// Plugin registry. Hand-maintained — every supported input/output
// type appears once in NewInput / NewOutput. Hand-maintained instead
// of reflection-based for two reasons:
//
//   1. Tree-shaking: every plugin is statically referenced, so
//      unused plugins don't pull deps into other binaries.
//   2. Zero runtime surprises: a typo in `type: filee` becomes a
//      compile-time error if we ever rename, not a silent "plugin not
//      found at runtime".
//
// All plugins live in the same `io` package (no subpackages) because
// plugins reference the package's `Event` type and a subpackage
// would create an import cycle.

import (
	"fmt"

	"github.com/kingknull/oblivrashell/internal/logger"
)

// NewInput resolves the YAML `type` field to a concrete Input plugin.
func NewInput(cfg PluginConfig, log *logger.Logger) (Input, error) {
	if cfg.ID == "" {
		return nil, fmt.Errorf("io: input is missing `id`")
	}
	switch cfg.Type {
	case "file":
		return NewFileInput(cfg.ID, cfg.Raw, log)
	case "syslog":
		return NewSyslogInput(cfg.ID, cfg.Raw, log)
	case "hec":
		return NewHECInput(cfg.ID, cfg.Raw, log)
	case "exec":
		return NewExecInput(cfg.ID, cfg.Raw, log)
	case "":
		return nil, fmt.Errorf("io: input %q is missing `type`", cfg.ID)
	default:
		return nil, fmt.Errorf("io: unknown input type %q for input %q", cfg.Type, cfg.ID)
	}
}

// NewOutput resolves the YAML `type` field to a concrete Output plugin.
func NewOutput(cfg PluginConfig, log *logger.Logger) (Output, error) {
	if cfg.ID == "" {
		return nil, fmt.Errorf("io: output is missing `id`")
	}
	switch cfg.Type {
	case "oblivra":
		return NewOblivraOutput(cfg.ID, cfg.Raw, log)
	case "syslog":
		return NewSyslogOutput(cfg.ID, cfg.Raw, log)
	case "s3":
		return NewS3Output(cfg.ID, cfg.Raw, log)
	case "http":
		return NewHTTPOutput(cfg.ID, cfg.Raw, log)
	case "file":
		return NewFileOutput(cfg.ID, cfg.Raw, log)
	case "":
		return nil, fmt.Errorf("io: output %q is missing `type`", cfg.ID)
	default:
		return nil, fmt.Errorf("io: unknown output type %q for output %q", cfg.Type, cfg.ID)
	}
}
