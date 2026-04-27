package config

import (
	"encoding/base64"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"strings"
)

const (
	defaultLogLevel = "info"
	defaultPort     = "8080"
	defaultDBPath   = "/data/caldo.db"
)

// Config contains runtime configuration loaded from environment variables.
type Config struct {
	BaseURL         string
	EncryptionKey   []byte
	ProxyUserHeader string
	LogLevel        string
	Port            string
	DBPath          string
}

// ValidationError captures a startup configuration validation failure.
type ValidationError struct {
	Field string
	Code  string
	Err   error
}

// Error returns a safe, user-facing error message.
func (e *ValidationError) Error() string {
	if e.Err == nil {
		return fmt.Sprintf("invalid %s", e.Field)
	}

	return fmt.Sprintf("invalid %s: %v", e.Field, e.Err)
}

// Unwrap returns the wrapped error.
func (e *ValidationError) Unwrap() error {
	return e.Err
}

// LogValue returns structured log fields for safe startup error logs.
func (e *ValidationError) LogValue() slog.Value {
	attrs := []slog.Attr{
		slog.String("field", e.Field),
		slog.String("code", e.Code),
	}

	if e.Err != nil {
		attrs = append(attrs, slog.String("cause", e.Err.Error()))
	}

	return slog.GroupValue(attrs...)
}

// Load reads and validates configuration from process environment variables.
func Load() (Config, error) {
	return LoadFromLookup(func(key string) (string, bool) {
		return os.LookupEnv(key)
	})
}

// LoadFromLookup reads and validates configuration using a custom environment lookup.
func LoadFromLookup(lookup func(key string) (string, bool)) (Config, error) {
	cfg := Config{
		LogLevel: defaultLogLevel,
		Port:     defaultPort,
		DBPath:   defaultDBPath,
	}

	baseURL := strings.TrimSpace(getenv(lookup, "BASE_URL"))
	if baseURL == "" {
		return Config{}, &ValidationError{Field: "BASE_URL", Code: "missing"}
	}

	if !strings.HasPrefix(baseURL, "https://") {
		return Config{}, &ValidationError{Field: "BASE_URL", Code: "must_use_https"}
	}
	cfg.BaseURL = baseURL

	proxyUserHeader := strings.TrimSpace(getenv(lookup, "PROXY_USER_HEADER"))
	if proxyUserHeader == "" {
		return Config{}, &ValidationError{Field: "PROXY_USER_HEADER", Code: "missing"}
	}
	cfg.ProxyUserHeader = proxyUserHeader

	encryptionKeyRaw := strings.TrimSpace(getenv(lookup, "ENCRYPTION_KEY"))
	if encryptionKeyRaw == "" {
		return Config{}, &ValidationError{Field: "ENCRYPTION_KEY", Code: "missing"}
	}

	encryptionKey, err := base64.StdEncoding.DecodeString(encryptionKeyRaw)
	if err != nil {
		return Config{}, &ValidationError{Field: "ENCRYPTION_KEY", Code: "invalid_base64", Err: errors.New("must be valid base64")}
	}

	if len(encryptionKey) != 32 {
		return Config{}, &ValidationError{Field: "ENCRYPTION_KEY", Code: "invalid_length", Err: fmt.Errorf("must decode to exactly 32 bytes")}
	}
	cfg.EncryptionKey = encryptionKey

	if logLevel := strings.TrimSpace(getenv(lookup, "LOG_LEVEL")); logLevel != "" {
		cfg.LogLevel = logLevel
	}
	if port := strings.TrimSpace(getenv(lookup, "PORT")); port != "" {
		cfg.Port = port
	}
	if dbPath := strings.TrimSpace(getenv(lookup, "DB_PATH")); dbPath != "" {
		cfg.DBPath = dbPath
	}

	return cfg, nil
}

func getenv(lookup func(key string) (string, bool), key string) string {
	value, ok := lookup(key)
	if !ok {
		return ""
	}

	return value
}
