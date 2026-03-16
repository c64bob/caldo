package service

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"caldo/internal/security"
	"caldo/internal/store/sqlite"
)

func newTaskServiceForTest(t *testing.T) (*TaskService, *sqlite.DAVAccountsRepo, []byte) {
	t.Helper()
	tmp := t.TempDir()
	db, err := sqlite.Open(filepath.Join(tmp, "caldo.db"))
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	key := []byte("0123456789abcdef0123456789abcdef")
	repo := sqlite.NewDAVAccountsRepo(db)
	return NewTaskService(repo, key, "Tasks"), repo, key
}

func TestLoadTaskPage_NoCredentials(t *testing.T) {
	svc, _, _ := newTaskServiceForTest(t)
	data, err := svc.LoadTaskPage(context.Background(), "alice@example.com", "")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if data.HasCredentials {
		t.Fatal("expected HasCredentials=false")
	}
}

func TestLoadTaskPage_WithCredentialsReturnsListsAndTasks(t *testing.T) {
	svc, repo, key := newTaskServiceForTest(t)
	caldavServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "REPORT" {
			t.Fatalf("expected REPORT request, got %s", r.Method)
		}
		if !strings.Contains(r.URL.Path, "/tasks/") {
			t.Fatalf("expected tasks collection request, got %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/xml")
		w.WriteHeader(http.StatusMultiStatus)
		_, _ = w.Write([]byte(`<?xml version="1.0" encoding="utf-8"?>
<multistatus xmlns="DAV:" xmlns:c="urn:ietf:params:xml:ns:caldav">
  <response>
    <href>/remote.php/dav/tasks/demo-1.ics</href>
    <propstat>
      <prop>
        <getetag>"demo-1"</getetag>
        <c:calendar-data>BEGIN:VCALENDAR
BEGIN:VTODO
UID:demo-1
SUMMARY:Remote task
STATUS:NEEDS-ACTION
END:VTODO
END:VCALENDAR</c:calendar-data>
      </prop>
      <status>HTTP/1.1 200 OK</status>
    </propstat>
  </response>
</multistatus>`))
	}))
	t.Cleanup(caldavServer.Close)

	encrypted, err := security.EncryptAESGCM(key, []byte("pw"))
	if err != nil {
		t.Fatalf("encrypt: %v", err)
	}
	err = repo.Upsert(context.Background(), sqlite.DAVAccount{
		PrincipalID:       "alice@example.com",
		ServerURL:         caldavServer.URL,
		Username:          "alice",
		PasswordEncrypted: encrypted,
	})
	if err != nil {
		t.Fatalf("upsert: %v", err)
	}

	data, err := svc.LoadTaskPage(context.Background(), "alice@example.com", "")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !data.HasCredentials {
		t.Fatal("expected HasCredentials=true")
	}
	if len(data.Lists) == 0 {
		t.Fatal("expected at least one list")
	}
	if len(data.Tasks) == 0 {
		t.Fatal("expected tasks from CalDAV server")
	}
	if data.Tasks[0].UID != "demo-1" {
		t.Fatalf("expected UID demo-1, got %s", data.Tasks[0].UID)
	}
}

func TestUpdateTask_PreservesExistingFieldsFromServer(t *testing.T) {
	svc, repo, key := newTaskServiceForTest(t)
	var putBody string
	caldavServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case "REPORT":
			w.Header().Set("Content-Type", "application/xml")
			w.WriteHeader(http.StatusMultiStatus)
			_, _ = w.Write([]byte(`<?xml version="1.0" encoding="utf-8"?>
<multistatus xmlns="DAV:" xmlns:c="urn:ietf:params:xml:ns:caldav">
  <response>
    <href>/tasks/demo-1.ics</href>
    <propstat>
      <prop>
        <getetag>"old"</getetag>
        <c:calendar-data>BEGIN:VCALENDAR
BEGIN:VTODO
UID:demo-1
SUMMARY:Old summary
STATUS:NEEDS-ACTION
DESCRIPTION:Keep me
CATEGORIES:work,focus
PERCENT-COMPLETE:40
END:VTODO
END:VCALENDAR</c:calendar-data>
      </prop>
      <status>HTTP/1.1 200 OK</status>
    </propstat>
  </response>
</multistatus>`))
		case "PUT":
			buf, _ := io.ReadAll(r.Body)
			putBody = string(buf)
			w.Header().Set("ETag", `"new"`)
			w.WriteHeader(http.StatusNoContent)
		default:
			t.Fatalf("unexpected method %s", r.Method)
		}
	}))
	t.Cleanup(caldavServer.Close)

	encrypted, err := security.EncryptAESGCM(key, []byte("pw"))
	if err != nil {
		t.Fatalf("encrypt: %v", err)
	}
	err = repo.Upsert(context.Background(), sqlite.DAVAccount{PrincipalID: "alice@example.com", ServerURL: caldavServer.URL, Username: "alice", PasswordEncrypted: encrypted})
	if err != nil {
		t.Fatalf("upsert: %v", err)
	}

	_, err = svc.UpdateTask(context.Background(), "alice@example.com", TaskMutationInput{ListID: "tasks", UID: "demo-1", Href: "/tasks/demo-1.ics", ETag: `"old"`, Summary: "New summary", Status: "COMPLETED", Priority: 5})
	if err != nil {
		t.Fatalf("update: %v", err)
	}

	if !strings.Contains(putBody, "DESCRIPTION:Keep me") {
		t.Fatalf("expected description preserved, body=%s", putBody)
	}
	if !strings.Contains(putBody, "CATEGORIES:work,focus") {
		t.Fatalf("expected categories preserved, body=%s", putBody)
	}
}

func TestUpdateTask_FailsWithoutETag(t *testing.T) {
	svc, repo, key := newTaskServiceForTest(t)
	caldavServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusMultiStatus)
		_, _ = w.Write([]byte(`<?xml version="1.0" encoding="utf-8"?>
<multistatus xmlns="DAV:" xmlns:c="urn:ietf:params:xml:ns:caldav"></multistatus>`))
	}))
	t.Cleanup(caldavServer.Close)
	encrypted, _ := security.EncryptAESGCM(key, []byte("pw"))
	_ = repo.Upsert(context.Background(), sqlite.DAVAccount{PrincipalID: "alice@example.com", ServerURL: caldavServer.URL, Username: "alice", PasswordEncrypted: encrypted})

	_, err := svc.UpdateTask(context.Background(), "alice@example.com", TaskMutationInput{ListID: "tasks", UID: "demo-1", Href: "/tasks/demo-1.ics", Summary: "New"})
	if err == nil || !strings.Contains(err.Error(), "etag") {
		t.Fatalf("expected missing etag error, got %v", err)
	}
}
