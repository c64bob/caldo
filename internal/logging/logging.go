package logging

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"log/slog"
	"reflect"
	"strings"
)

const (
	// RedactedValue is used when masking sensitive logging attributes.
	RedactedValue = "[REDACTED]"
)

var sensitiveKeys = map[string]struct{}{
	"title":                   {},
	"description":             {},
	"raw_vtodo":               {},
	"password":                {},
	"token":                   {},
	"encryption_key":          {},
	"session_id":              {},
	"csrf_token":              {},
	"authorization":           {},
	"proxy_auth_header_value": {},
	"proxy_user_header_value": {},
	"caldav_credentials":      {},
	"caldav_password":         {},
	"caldav_url":              {},
	"caldav_username":         {},
}

// New creates the default logger with safe central masking.
func New(w io.Writer, appEnv, logLevel string) *slog.Logger {
	level := parseLevel(logLevel)
	opts := &slog.HandlerOptions{Level: level}

	var base slog.Handler
	if strings.EqualFold(strings.TrimSpace(appEnv), "development") {
		base = slog.NewTextHandler(w, opts)
	} else {
		base = slog.NewJSONHandler(w, opts)
	}

	return slog.New(&maskingHandler{next: base})
}

// NewCorrelationID returns a random ID for request_id and sync_run_id fields.
func NewCorrelationID() (string, error) {
	return newID()
}

// NewSyncRunLogger returns a logger enriched with sync_run_id and the generated ID.
func NewSyncRunLogger(logger *slog.Logger) (*slog.Logger, string, error) {
	syncRunID, err := NewCorrelationID()
	if err != nil {
		return nil, "", fmt.Errorf("generate sync run id: %w", err)
	}

	return logger.With("sync_run_id", syncRunID), syncRunID, nil
}

func parseLevel(raw string) slog.Level {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "debug":
		return slog.LevelDebug
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

func newID() (string, error) {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("read random bytes: %w", err)
	}

	return hex.EncodeToString(buf), nil
}

type maskingHandler struct{ next slog.Handler }

func (h *maskingHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.next.Enabled(ctx, level)
}

func (h *maskingHandler) Handle(ctx context.Context, record slog.Record) error {
	cloned := slog.NewRecord(record.Time, record.Level, record.Message, record.PC)
	record.Attrs(func(attr slog.Attr) bool {
		cloned.AddAttrs(maskAttr(attr))
		return true
	})
	return h.next.Handle(ctx, cloned)
}

func (h *maskingHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	masked := make([]slog.Attr, 0, len(attrs))
	for _, attr := range attrs {
		masked = append(masked, maskAttr(attr))
	}
	return &maskingHandler{next: h.next.WithAttrs(masked)}
}

func (h *maskingHandler) WithGroup(name string) slog.Handler {
	return &maskingHandler{next: h.next.WithGroup(name)}
}

func maskAttr(attr slog.Attr) slog.Attr {
	if _, ok := sensitiveKeys[strings.ToLower(attr.Key)]; ok {
		return slog.String(attr.Key, RedactedValue)
	}
	if attr.Value.Kind() == slog.KindGroup {
		group := attr.Value.Group()
		masked := make([]slog.Attr, 0, len(group))
		for _, nested := range group {
			masked = append(masked, maskAttr(nested))
		}
		return slog.Group(attr.Key, attrsToAny(masked)...)
	}
	if err, ok := attr.Value.Any().(error); ok {
		return slog.Group(attr.Key, slog.String("type", reflect.TypeOf(err).String()))
	}
	return attr
}

func attrsToAny(attrs []slog.Attr) []any {
	values := make([]any, 0, len(attrs))
	for _, attr := range attrs {
		values = append(values, attr)
	}
	return values
}
