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

// Only real secrets are masked: the values the caller passes explicitly plus the
// usual token fields. Item ids and user ids are 32-char hex as well, and blanking
// them makes a log line useless for diagnosis.
var secretFieldPattern = regexp.MustCompile(`(?i)(x-emby-token\s*[:=]\s*|api_key=|token=|accesstoken["']?\s*[:=]\s*["']?)([^\s&"',]+)`)

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
	return &Logger{w: file}, func() { _ = file.Close() }, nil
}

// Redact masks opaque secrets and the given exact secret values.
func Redact(message string, secrets ...string) string {
	for _, secret := range secrets {
		if strings.TrimSpace(secret) == "" {
			continue
		}
		message = strings.ReplaceAll(message, secret, "***")
	}
	return secretFieldPattern.ReplaceAllString(message, "${1}***")
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