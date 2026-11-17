// Package logger writes one line per run and never records credentials.
package logger

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

// A Jellyfin token, API key or cookie value is a long opaque string; redact
// anything that looks like one so a log line can be shared safely.
var opaquePattern = regexp.MustCompile(`[A-Za-z0-9_\-\.]{24,}`)

type Logger struct {
	w io.Writer
}

func New(dir string) (*Logger, func(), error) {
	if dir == "" {
		return &Logger{w: os.Stderr}, func() {}, nil
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, nil, err
	}
	file, err := os.OpenFile(filepath.Join(dir, "bridge.log"), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		return nil, nil, err
	}
	return &Logger{w: io.MultiWriter(file, os.Stderr)}, func() { _ = file.Close() }, nil
}

// Redact masks opaque secrets and the given exact secret values.
func Redact(message string, secrets ...string) string {
	for _, secret := range secrets {
		if strings.TrimSpace(secret) == "" {
			continue
		}
		message = strings.ReplaceAll(message, secret, "***")
	}
	return opaquePattern.ReplaceAllString(message, "***")
}

func (l *Logger) Info(step, message string, secrets ...string) {
	l.write("info", step, message, secrets...)
}

func (l *Logger) Fail(step, message string, secrets ...string) {
	l.write("error", step, message, secrets...)
}

func (l *Logger) write(level, step, message string, secrets ...string) {
	fmt.Fprintf(l.w, "%s\t%s\t%s\t%s\n", time.Now().Format(time.RFC3339), level, step, Redact(message, secrets...))
}