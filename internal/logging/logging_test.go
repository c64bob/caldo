package logging

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"github.com/google/uuid"
)

func TestNewUsesJSONInProduction(t *testing.T) {
	t.Parallel()

	buf := bytes.NewBuffer(nil)
	logger := New(buf, "production", "info")
	logger.Info("test", "safe", "value")

	if !strings.Contains(buf.String(), `{"time":`) {
		t.Fatalf("expected json log output, got %s", buf.String())
	}
}

func TestNewUsesTextInDevelopment(t *testing.T) {
	t.Parallel()

	buf := bytes.NewBuffer(nil)
	logger := New(buf, "development", "info")
	logger.Info("test", "safe", "value")

	if !strings.Contains(buf.String(), "level=INFO") {
		t.Fatalf("expected text log output, got %s", buf.String())
	}
}

func TestMaskingHandlerMasksSensitiveAndErrorMessage(t *testing.T) {
	t.Parallel()

	buf := bytes.NewBuffer(nil)
	logger := New(buf, "production", "info")
	logger.Error("test", "session_id", "abc", "error", errors.New("contains user input"))

	output := buf.String()
	if strings.Contains(output, "abc") {
		t.Fatalf("unexpected sensitive value in logs: %s", output)
	}
	if strings.Contains(output, "contains user input") {
		t.Fatalf("unexpected raw error message in logs: %s", output)
	}
	if !strings.Contains(output, `"session_id":"[REDACTED]"`) {
		t.Fatalf("expected redacted sensitive value in logs: %s", output)
	}
	if !strings.Contains(output, `"type":"*errors.errorString"`) {
		t.Fatalf("expected error type in logs: %s", output)
	}
}

func TestNewCorrelationIDReturnsUUID(t *testing.T) {
	t.Parallel()

	id, err := NewCorrelationID()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if _, err := uuid.Parse(id); err != nil {
		t.Fatalf("expected UUID, got %q: %v", id, err)
	}
}

func TestNewSyncRunLoggerAddsSyncRunID(t *testing.T) {
	t.Parallel()

	buf := bytes.NewBuffer(nil)
	logger := New(buf, "production", "info")
	syncLogger, syncRunID, err := NewSyncRunLogger(logger)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	syncLogger.Info("sync started")
	output := buf.String()
	if _, err := uuid.Parse(syncRunID); err != nil {
		t.Fatalf("expected UUID sync_run_id, got %q: %v", syncRunID, err)
	}

	if !strings.Contains(output, `"sync_run_id":"`+syncRunID+`"`) {
		t.Fatalf("expected sync_run_id in logs: %s", output)
	}
}
