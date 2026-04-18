package logger

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/rs/zerolog"
)

type Level int

const (
	DebugLevel Level = iota
	InfoLevel
	WarnLevel
	ErrorLevel
	FatalLevel
)

func (l Level) String() string {
	switch l {
	case DebugLevel:
		return "DEBUG"
	case InfoLevel:
		return "INFO"
	case WarnLevel:
		return "WARN"
	case ErrorLevel:
		return "ERROR"
	case FatalLevel:
		return "FATAL"
	default:
		return "UNKNOWN"
	}
}

// toZerologLevel converts custom Level to zerolog.Level
func (l Level) toZerologLevel() zerolog.Level {
	switch l {
	case DebugLevel:
		return zerolog.DebugLevel
	case InfoLevel:
		return zerolog.InfoLevel
	case WarnLevel:
		return zerolog.WarnLevel
	case ErrorLevel:
		return zerolog.ErrorLevel
	case FatalLevel:
		return zerolog.FatalLevel
	default:
		return zerolog.InfoLevel
	}
}

type Config struct {
	Level      Level
	OutputPath string
	MaxSize    int
	MaxBackups int
	Sanitize   bool
	JSON       bool
}

type Logger struct {
	level    Level
	file     *os.File
	logger   zerolog.Logger
	sanitize bool
	patterns []*regexp.Regexp
	json     bool
	prefix   string
}

// sanitizeWriter wraps an io.Writer and sanitizes any writes
type sanitizeWriter struct {
	target   io.Writer
	patterns []*regexp.Regexp
}

func (s *sanitizeWriter) Write(p []byte) (n int, err error) {
	msg := string(p)
	for _, pattern := range s.patterns {
		msg = pattern.ReplaceAllString(msg, "[REDACTED]")
	}
	// Write the sanitized string to the underlying writer
	_, err = s.target.Write([]byte(msg))
	// Always return the original length to satisfy io.Writer interface, even if we reduced its length
	return len(p), err
}

func New(cfg Config) (*Logger, error) {
	if err := os.MkdirAll(filepath.Dir(cfg.OutputPath), 0700); err != nil {
		return nil, fmt.Errorf("create log dir: %w", err)
	}

	file, err := os.OpenFile(cfg.OutputPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0600)
	if err != nil {
		return nil, fmt.Errorf("open log file: %w", err)
	}

	zerolog.TimeFieldFormat = zerolog.TimeFormatUnixMs

	var writers []io.Writer

	if !cfg.JSON {
		// Console writer for human readability formatting
		writers = append(writers, zerolog.ConsoleWriter{
			Out:        os.Stdout,
			TimeFormat: "2006-01-02 15:04:05.000",
		})
	} else {
		// JSON directly to stdout
		writers = append(writers, os.Stdout)
	}

	// Always write JSON to file
	writers = append(writers, file)
	
	multi := io.MultiWriter(writers...)
	
	l := &Logger{
		level:    cfg.Level,
		file:     file,
		sanitize: cfg.Sanitize,
		json:     cfg.JSON,
	}

	if cfg.Sanitize {
		l.patterns = []*regexp.Regexp{
			regexp.MustCompile(`(?i)(password|passwd|pwd)\s*[=:]\s*\S+`),
			regexp.MustCompile(`(?i)(secret|token|key|api_key)\s*[=:]\s*\S+`),
			regexp.MustCompile(`-----BEGIN[\s\S]*?-----END[^\n]*`),
			regexp.MustCompile(`(?i)AKIA[0-9A-Z]{16}`),
		}
		
		// Wrap the multiwriter with the sanitizer
		multi = &sanitizeWriter{
			target:   multi,
			patterns: l.patterns,
		}
	}

	l.logger = zerolog.New(multi).Level(cfg.Level.toZerologLevel()).With().Timestamp().Logger()

	return l, nil
}

// NewStdoutLogger creates a logger that only writes to stdout.
func NewStdoutLogger() *Logger {
	writers := []io.Writer{
		zerolog.ConsoleWriter{
			Out:        os.Stdout,
			TimeFormat: "2006-01-02 15:04:05.000",
		},
	}
	multi := io.MultiWriter(writers...)

	patterns := []*regexp.Regexp{
		regexp.MustCompile(`(?i)(password|passwd|pwd)\s*[=:]\s*\S+`),
		regexp.MustCompile(`(?i)(secret|token|key|api_key)\s*[=:]\s*\S+`),
		regexp.MustCompile(`-----BEGIN[\s\S]*?-----END[^\n]*`),
		regexp.MustCompile(`(?i)AKIA[0-9A-Z]{16}`),
	}
	
	sanitized := &sanitizeWriter{
		target:   multi,
		patterns: patterns,
	}

	return &Logger{
		level:    InfoLevel,
		logger:   zerolog.New(sanitized).Level(zerolog.InfoLevel).With().Timestamp().Logger(),
		sanitize: true,
		patterns: patterns,
	}
}

func (l *Logger) log(level zerolog.Level, format string, args ...interface{}) {
	if len(args) == 0 {
		l.logger.WithLevel(level).Msg(format)
	} else {
		l.logger.WithLevel(level).Msgf(format, args...)
	}
}

func (l *Logger) Debug(format string, args ...interface{}) { l.log(zerolog.DebugLevel, format, args...) }
func (l *Logger) Info(format string, args ...interface{})  { l.log(zerolog.InfoLevel, format, args...) }
func (l *Logger) Warn(format string, args ...interface{})  { l.log(zerolog.WarnLevel, format, args...) }
func (l *Logger) Error(format string, args ...interface{}) { l.log(zerolog.ErrorLevel, format, args...) }
func (l *Logger) Fatal(format string, args ...interface{}) { l.log(zerolog.FatalLevel, format, args...) }

func (l *Logger) Close() error {
	if l.file != nil {
		return l.file.Close()
	}
	return nil
}

func (l *Logger) WithPrefix(prefix string) *Logger {
	prefix = strings.ToUpper(prefix)
	newLog := &Logger{
		level:    l.level,
		file:     l.file,
		sanitize: l.sanitize,
		patterns: l.patterns,
		json:     l.json,
		prefix:   prefix,
	}
	
	newLog.logger = l.logger.With().Str("prefix", prefix).Logger()
	return newLog
}
