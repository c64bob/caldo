package config

import (
	"bytes"
	"encoding/base64"
	"errors"
	"log/slog"
	"strings"
	"testing"
)

func TestLoadFromLookup(t *testing.T) {
	t.Parallel()

	validKey := make([]byte, 32)
	encodedKey := base64.StdEncoding.EncodeToString(validKey)

	tests := []struct {
		name      string
		env       map[string]string
		assertErr func(t *testing.T, err error)
		assertCfg func(t *testing.T, cfg Config)
	}{
		{
			name: "loads required config and defaults",
			env: map[string]string{
				"BASE_URL":          "https://todos.example.com",
				"PROXY_USER_HEADER": "X-Forwarded-User",
				"ENCRYPTION_KEY":    encodedKey,
			},
			assertCfg: func(t *testing.T, cfg Config) {
				t.Helper()
				if cfg.LogLevel != defaultLogLevel {
					t.Fatalf("unexpected log level: %s", cfg.LogLevel)
				}
				if cfg.Port != defaultPort {
					t.Fatalf("unexpected port: %s", cfg.Port)
				}
				if cfg.DBPath != defaultDBPath {
					t.Fatalf("unexpected db path: %s", cfg.DBPath)
				}
			},
		},
		{
			name: "uses optional overrides",
			env: map[string]string{
				"BASE_URL":          "https://todos.example.com",
				"PROXY_USER_HEADER": "X-Forwarded-User",
				"ENCRYPTION_KEY":    encodedKey,
				"LOG_LEVEL":         "debug",
				"PORT":              "8181",
				"DB_PATH":           "/tmp/caldo.db",
			},
			assertCfg: func(t *testing.T, cfg Config) {
				t.Helper()
				if cfg.LogLevel != "debug" || cfg.Port != "8181" || cfg.DBPath != "/tmp/caldo.db" {
					t.Fatalf("unexpected overrides: %+v", cfg)
				}
			},
		},
		{
			name: "fails when base url missing",
			env: map[string]string{
				"PROXY_USER_HEADER": "X-Forwarded-User",
				"ENCRYPTION_KEY":    encodedKey,
			},
			assertErr: validationError("BASE_URL", "missing"),
		},
		{
			name: "fails when base url not https",
			env: map[string]string{
				"BASE_URL":          "http://todos.example.com",
				"PROXY_USER_HEADER": "X-Forwarded-User",
				"ENCRYPTION_KEY":    encodedKey,
			},
			assertErr: validationError("BASE_URL", "must_use_https"),
		},
		{
			name: "fails when proxy user header missing",
			env: map[string]string{
				"BASE_URL":       "https://todos.example.com",
				"ENCRYPTION_KEY": encodedKey,
			},
			assertErr: validationError("PROXY_USER_HEADER", "missing"),
		},
		{
			name: "fails when encryption key missing",
			env: map[string]string{
				"BASE_URL":          "https://todos.example.com",
				"PROXY_USER_HEADER": "X-Forwarded-User",
			},
			assertErr: validationError("ENCRYPTION_KEY", "missing"),
		},
		{
			name: "fails when encryption key is not base64",
			env: map[string]string{
				"BASE_URL":          "https://todos.example.com",
				"PROXY_USER_HEADER": "X-Forwarded-User",
				"ENCRYPTION_KEY":    "not base64",
			},
			assertErr: validationError("ENCRYPTION_KEY", "invalid_base64"),
		},
		{
			name: "fails when encryption key decodes to wrong length",
			env: map[string]string{
				"BASE_URL":          "https://todos.example.com",
				"PROXY_USER_HEADER": "X-Forwarded-User",
				"ENCRYPTION_KEY":    base64.StdEncoding.EncodeToString([]byte("too short")),
			},
			assertErr: validationError("ENCRYPTION_KEY", "invalid_length"),
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			cfg, err := LoadFromLookup(mapLookup(tc.env))
			if tc.assertErr != nil {
				tc.assertErr(t, err)
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if cfg.BaseURL != "https://todos.example.com" {
				t.Fatalf("unexpected base url: %s", cfg.BaseURL)
			}
			if cfg.ProxyUserHeader != "X-Forwarded-User" {
				t.Fatalf("unexpected proxy user header: %s", cfg.ProxyUserHeader)
			}
			if len(cfg.EncryptionKey) != 32 {
				t.Fatalf("unexpected encryption key length: %d", len(cfg.EncryptionKey))
			}
			if tc.assertCfg != nil {
				tc.assertCfg(t, cfg)
			}
		})
	}
}

func TestValidationErrorLogValueDoesNotExposeSecrets(t *testing.T) {
	t.Parallel()

	buf := bytes.NewBuffer(nil)
	logger := slog.New(slog.NewJSONHandler(buf, nil))

	err := &ValidationError{Field: "ENCRYPTION_KEY", Code: "invalid_base64", Err: errors.New("must be valid base64")}
	logger.Error("startup configuration invalid", "validation", err)

	output := buf.String()
	forbidden := []string{"super-secret", "ENCRYPTION_KEY=", "cGFzc3dvcmQ="}
	for _, needle := range forbidden {
		if strings.Contains(output, needle) {
			t.Fatalf("log unexpectedly includes secret-like value %q: %s", needle, output)
		}
	}

	if !strings.Contains(output, `"field":"ENCRYPTION_KEY"`) {
		t.Fatalf("expected field in log output: %s", output)
	}
	if !strings.Contains(output, `"code":"invalid_base64"`) {
		t.Fatalf("expected code in log output: %s", output)
	}
}

func mapLookup(values map[string]string) func(string) (string, bool) {
	return func(key string) (string, bool) {
		value, ok := values[key]
		return value, ok
	}
}

func validationError(expectedField, expectedCode string) func(t *testing.T, err error) {
	return func(t *testing.T, err error) {
		t.Helper()

		if err == nil {
			t.Fatal("expected error")
		}

		var validationErr *ValidationError
		if !errors.As(err, &validationErr) {
			t.Fatalf("expected validation error, got %T", err)
		}

		if validationErr.Field != expectedField {
			t.Fatalf("unexpected field: %s", validationErr.Field)
		}
		if validationErr.Code != expectedCode {
			t.Fatalf("unexpected code: %s", validationErr.Code)
		}
	}
}
