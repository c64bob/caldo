package handlers

import (
	"bytes"
	"errors"
	"log"
	"strings"
	"testing"

	"caldo/internal/service"
)

func TestLogTaskLoadError_UsesLoadContextValues(t *testing.T) {
	buf := &bytes.Buffer{}
	prev := log.Writer()
	log.SetOutput(buf)
	t.Cleanup(func() { log.SetOutput(prev) })

	err := &service.TaskLoadContextError{
		AuthPrincipal: "admin",
		DAVUsername:   "testuser",
		ListID:        "tasks",
		Err:           errors.New("backend down"),
	}

	logTaskLoadError("tasks.page", "admin", "", err)

	line := buf.String()
	if !strings.Contains(line, "principal=testuser") {
		t.Fatalf("expected dav principal in log, got %q", line)
	}
	if !strings.Contains(line, "list=tasks") {
		t.Fatalf("expected resolved list in log, got %q", line)
	}
}

func TestLogTaskLoadError_EmptyListUsesAuto(t *testing.T) {
	buf := &bytes.Buffer{}
	prev := log.Writer()
	log.SetOutput(buf)
	t.Cleanup(func() { log.SetOutput(prev) })

	logTaskLoadError("tasks.page", "admin", "", errors.New("boom"))
	line := buf.String()
	if !strings.Contains(line, "list=auto") {
		t.Fatalf("expected fallback list auto, got %q", line)
	}
}
