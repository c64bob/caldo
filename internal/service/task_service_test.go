package service

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

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

func TestDeleteTask_UsesServerTaskHrefInsteadOfRawInput(t *testing.T) {
	svc, repo, key := newTaskServiceForTest(t)
	var deletePath string
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
SUMMARY:Task
STATUS:NEEDS-ACTION
END:VTODO
END:VCALENDAR</c:calendar-data>
      </prop>
      <status>HTTP/1.1 200 OK</status>
    </propstat>
  </response>
</multistatus>`))
		case "DELETE":
			deletePath = r.URL.Path
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

	err = svc.DeleteTask(context.Background(), "alice@example.com", TaskMutationInput{ListID: "tasks", UID: "demo-1", Href: "/evil.ics", ETag: `"old"`})
	if err != nil {
		t.Fatalf("delete: %v", err)
	}
	if deletePath != "/tasks/demo-1.ics" {
		t.Fatalf("expected delete on server task href, got %s", deletePath)
	}
}

func TestParseCategories_DeduplicatesAndTrims(t *testing.T) {
	got := ParseCategories(" work, focus,Work, , home ")
	want := []string{"work", "focus", "home"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("expected %v, got %v", want, got)
	}
}

func TestParseDue_AcceptsDateAndDatetime(t *testing.T) {
	due, kind := ParseDue("2026-03-16")
	if due == nil || kind != "date" {
		t.Fatalf("expected date parse, got due=%v kind=%q", due, kind)
	}
	due, kind = ParseDue("2026-03-16T09:45")
	if due == nil || kind != "datetime" {
		t.Fatalf("expected datetime parse, got due=%v kind=%q", due, kind)
	}
}

func TestParseDue_UsesLocalTimezoneForDatetime(t *testing.T) {
	originalLocal := time.Local
	loc := time.FixedZone("UTC+02", 2*60*60)
	time.Local = loc
	t.Cleanup(func() { time.Local = originalLocal })

	due, kind := ParseDue("2026-03-16T09:45")
	if due == nil {
		t.Fatal("expected due to be parsed")
	}
	if kind != "datetime" {
		t.Fatalf("expected kind datetime, got %q", kind)
	}
	if gotOffset := due.Format("-0700"); gotOffset != "+0200" {
		t.Fatalf("expected local timezone offset +0200, got %s", gotOffset)
	}
	if due.Hour() != 9 || due.Minute() != 45 {
		t.Fatalf("expected local wall time 09:45, got %02d:%02d", due.Hour(), due.Minute())
	}
}

func TestParseDue_AcceptsNaturalLanguage(t *testing.T) {
	due, kind := ParseDue("tomorrow")
	if due == nil || kind != "date" {
		t.Fatalf("expected natural date parse, got due=%v kind=%q", due, kind)
	}
}

func TestParseSmartAdd_ParsesTokens(t *testing.T) {
	in, err := ParseSmartAdd(`"Arzt" /folder:Privat /context:@Telefon /due:friday !high #Gesundheit`)
	if err != nil {
		t.Fatalf("parse smart add: %v", err)
	}
	if in.Summary != "Arzt" {
		t.Fatalf("expected summary Arzt, got %q", in.Summary)
	}
	if in.ListID != "Privat" {
		t.Fatalf("expected folder Privat, got %q", in.ListID)
	}
	if in.Priority != 7 {
		t.Fatalf("expected priority 7, got %d", in.Priority)
	}
	if in.Due == nil || in.DueKind != "date" {
		t.Fatalf("expected due date from natural language, got due=%v kind=%q", in.Due, in.DueKind)
	}
	if !reflect.DeepEqual(in.Categories, []string{"@Telefon", "Gesundheit"}) {
		t.Fatalf("unexpected categories: %v", in.Categories)
	}
}
