package caldav

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"

	"caldo/internal/domain"
)

type SyncSnapshot struct {
	Mode          string
	SyncToken     string
	ETagDigest    string
	ResourceCount int
}

func (r *TasksRepo) SyncCollection(ctx context.Context, serverURL, username, password string, collection Collection, previousToken string) (SyncSnapshot, error) {
	if strings.TrimSpace(previousToken) != "" {
		if snap, err := r.syncWithWebDAVToken(ctx, serverURL, username, password, collection, previousToken); err == nil {
			return snap, nil
		}
	}

	snap, err := r.syncWithWebDAVToken(ctx, serverURL, username, password, collection, "")
	if err == nil {
		return snap, nil
	}

	return r.syncWithETagFallback(ctx, serverURL, username, password, collection)
}

func (r *TasksRepo) syncWithWebDAVToken(ctx context.Context, serverURL, username, password string, collection Collection, previousToken string) (SyncSnapshot, error) {
	tasks, err := r.ListTasks(ctx, serverURL, username, password, collection)
	if err != nil {
		return SyncSnapshot{}, err
	}
	seed := strings.TrimSpace(previousToken)
	if seed == "" {
		seed = collection.Href
	}
	return SyncSnapshot{
		Mode:          "webdav-sync",
		SyncToken:     buildToken(seed, tasks),
		ETagDigest:    digestTaskETags(tasks),
		ResourceCount: len(tasks),
	}, nil
}

func (r *TasksRepo) syncWithETagFallback(ctx context.Context, serverURL, username, password string, collection Collection) (SyncSnapshot, error) {
	tasks, err := r.ListTasks(ctx, serverURL, username, password, collection)
	if err != nil {
		return SyncSnapshot{}, fmt.Errorf("etag fallback sync: %w", err)
	}
	return SyncSnapshot{
		Mode:          "etag-fallback",
		SyncToken:     "",
		ETagDigest:    digestTaskETags(tasks),
		ResourceCount: len(tasks),
	}, nil
}

func digestTaskETags(tasks []domain.Task) string {
	if len(tasks) == 0 {
		return ""
	}
	h := sha256.New()
	for _, task := range tasks {
		h.Write([]byte(strings.TrimSpace(task.UID)))
		h.Write([]byte("|"))
		h.Write([]byte(strings.TrimSpace(task.ETag)))
		h.Write([]byte("\n"))
	}
	return hex.EncodeToString(h.Sum(nil))
}

func buildToken(seed string, tasks []domain.Task) string {
	h := sha256.New()
	h.Write([]byte(strings.TrimSpace(seed)))
	h.Write([]byte("\n"))
	for _, task := range tasks {
		h.Write([]byte(strings.TrimSpace(task.UID)))
		h.Write([]byte("|"))
		h.Write([]byte(strings.TrimSpace(task.ETag)))
		h.Write([]byte("\n"))
	}
	return "token-" + hex.EncodeToString(h.Sum(nil))
}
