package logger

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"
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

type Config struct {
	Level      Level
	OutputPath string
	MaxSize    int
	MaxBackups int
	Sanitize   bool
	JSON       bool
}

type Logger struct {
	mu       sync.Mutex
	level    Level
	file     *os.File
	logger   *log.Logger
	sanitize bool
	patterns []*regexp.Regexp
	json     bool
	prefix   string
}

func New(cfg Config) (*Logger, error) {
	if err := os.MkdirAll(filepath.Dir(cfg.OutputPath), 0700); err != nil {
		return nil, fmt.Errorf("create log dir: %w", err)
	}

	file, err := os.OpenFile(cfg.OutputPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0600)
	if err != nil {
		return nil, fmt.Errorf("open log file: %w", err)
	}

	multi := io.MultiWriter(os.Stdout, file)
	l := &Logger{
		level:    cfg.Level,
		file:     file,
		logger:   log.New(multi, "", 0),
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
	}

	return l, nil
}

func (l *Logger) log(level Level, format string, args ...interface{}) {
	if level < l.level {
		return
	}
	l.mu.Lock()
	defer l.mu.Unlock()

	msg := fmt.Sprintf(format, args...)
	if l.sanitize {
		msg = l.sanitizeMessage(msg)
	}

	now := time.Now()
	if l.json {
		entry := map[string]interface{}{
			"timestamp": now.Format(time.RFC3339Nano),
			"level":     level.String(),
			"message":   msg,
		}
		if l.prefix != "" {
			entry["prefix"] = l.prefix
		}
		data, _ := json.Marshal(entry)
		l.logger.Printf("%s", string(data))
	} else {
		timestamp := now.Format("2006-01-02 15:04:05.000")
		l.logger.Printf("[%s] [%s] %s", timestamp, level.String(), msg)
	}
}

func (l *Logger) sanitizeMessage(msg string) string {
	for _, pattern := range l.patterns {
		msg = pattern.ReplaceAllString(msg, "[REDACTED]")
	}
	return msg
}

func (l *Logger) Debug(format string, args ...interface{}) { l.log(DebugLevel, format, args...) }
func (l *Logger) Info(format string, args ...interface{})  { l.log(InfoLevel, format, args...) }
func (l *Logger) Warn(format string, args ...interface{})  { l.log(WarnLevel, format, args...) }
func (l *Logger) Error(format string, args ...interface{}) { l.log(ErrorLevel, format, args...) }
func (l *Logger) Fatal(format string, args ...interface{}) { l.log(FatalLevel, format, args...) }

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
	if l.json {
		newLog.logger = l.logger
	} else {
		newLog.logger = log.New(l.logger.Writer(), fmt.Sprintf("[%s] ", prefix), 0)
	}
	return newLog
}
