package handler

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"caldo/internal/db"
	"github.com/go-chi/chi/v5"
)

func TestTaskFavoriteSetAndUnsetPreservesCategories(t *testing.T) {
	t.Parallel()
	database := openSQLiteForTaskUpdateHandlerTest(t)
	seedTaskUpdateHandlerData(t, database)
	setTaskRawVTODO(t, database, `BEGIN:VCALENDAR
BEGIN:VTODO
UID:uid-1
SUMMARY:old
STATUS:NEEDS-ACTION
CATEGORIES:home,work
END:VTODO
END:VCALENDAR`)

	key := bytes.Repeat([]byte{0x61}, 32)
	if err := database.SaveCalDAVCredentials(context.Background(), key, db.CalDAVCredentials{URL: "https://dav.example", Username: "alice", Password: "secret"}); err != nil {
		t.Fatalf("save credentials: %v", err)
	}

	h := TaskFavorite(taskUpdateDependencies{database: database, encryptionKey: key, todos: &stubTaskUpdateTodoClient{updateETag: `"etag-2"`}})
	req := favoriteRequest(t, "2", "true")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("unexpected set status: %d body=%q", rr.Code, rr.Body.String())
	}

	var raw string
	if err := database.Conn.QueryRowContext(context.Background(), `SELECT raw_vtodo FROM tasks WHERE id='task-1';`).Scan(&raw); err != nil {
		t.Fatalf("query task: %v", err)
	}
	if !strings.Contains(raw, "CATEGORIES:home,work,STARRED") {
		t.Fatalf("expected STARRED category to be appended, got %q", raw)
	}

	req = favoriteRequest(t, "4", "false")
	rr = httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("unexpected unset status: %d body=%q", rr.Code, rr.Body.String())
	}
	if err := database.Conn.QueryRowContext(context.Background(), `SELECT raw_vtodo FROM tasks WHERE id='task-1';`).Scan(&raw); err != nil {
		t.Fatalf("query task: %v", err)
	}
	if strings.Contains(raw, "STARRED") || !strings.Contains(raw, "CATEGORIES:home,work") {
		t.Fatalf("expected STARRED removed but other categories preserved, got %q", raw)
	}
}

func TestTaskFavoriteRequiresExpectedVersion(t *testing.T) {
	t.Parallel()
	database := openSQLiteForTaskUpdateHandlerTest(t)
	seedTaskUpdateHandlerData(t, database)
	key := bytes.Repeat([]byte{0x62}, 32)
	_ = database.SaveCalDAVCredentials(context.Background(), key, db.CalDAVCredentials{URL: "https://dav.example", Username: "alice", Password: "secret"})

	h := TaskFavorite(taskUpdateDependencies{database: database, encryptionKey: key, todos: &stubTaskUpdateTodoClient{updateETag: `"etag-2"`}})
	req := favoriteRequest(t, "99", "true")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusConflict {
		t.Fatalf("expected conflict, got %d", rr.Code)
	}
}

func favoriteRequest(t *testing.T, expectedVersion string, favorite string) *http.Request {
	t.Helper()
	form := url.Values{"expected_version": {expectedVersion}, "favorite": {favorite}}
	req := httptest.NewRequest(http.MethodPost, "/tasks/task-1/favorite", bytes.NewBufferString(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("X-Tab-ID", "tab-1")
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("taskID", "task-1")
	return req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
}
