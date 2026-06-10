package handler

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/xml"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"caldo/internal/caldav"
	"caldo/internal/db"
	"caldo/internal/model"
	caldosync "caldo/internal/sync"
)

func TestRouterMVPFlowAgainstFakeCalDAVServer(t *testing.T) {
	const calendarHref = "/cal/work/"
	secret := []byte("12345678901234567890123456789012")
	ctx := context.Background()

	fake := newFakeRouterCalDAV(calendarHref, "Work")
	fake.putObject("/cal/work/imported.ics", "uid-import", e2eVTODO("uid-import", "Imported Task"), `"etag-import-1"`)
	fake.putObject("/cal/work/delete.ics", "uid-delete", e2eVTODO("uid-delete", "Remote Delete Candidate"), `"etag-delete-1"`)
	fake.putObject("/cal/work/conflict.ics", "uid-conflict", e2eVTODO("uid-conflict", "Conflict Base"), `"etag-conflict-1"`)

	server := httptest.NewServer(fake)
	t.Cleanup(server.Close)

	database, err := db.OpenSQLite(filepath.Join(t.TempDir(), "caldo.db"))
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	t.Cleanup(func() { _ = database.Close() })

	runner, err := caldosync.NewRunner(database, secret, caldav.NewTodoClient(nil))
	if err != nil {
		t.Fatalf("new sync runner: %v", err)
	}
	scheduler := &fakeSetupScheduler{}
	router := NewRouter(slog.New(slog.NewTextHandler(io.Discard, nil)), "X-Forwarded-User", testManifest(t), false, secret, database, ctx, scheduler, runner)

	rr := e2eRequest(t, router, secret, http.MethodGet, "/health", nil, "")
	if rr.Code != http.StatusOK {
		t.Fatalf("health status: got %d body=%q", rr.Code, rr.Body.String())
	}

	rr = e2eRequest(t, router, secret, http.MethodGet, "/", nil, "")
	if rr.Code != http.StatusFound || rr.Header().Get("Location") != "/setup" {
		t.Fatalf("setup gate: got status=%d location=%q body=%q", rr.Code, rr.Header().Get("Location"), rr.Body.String())
	}

	rr = e2eRequest(t, router, secret, http.MethodPost, "/setup/caldav", url.Values{
		"caldav_url":      {server.URL},
		"caldav_username": {"alice"},
		"caldav_password": {"secret"},
	}, "")
	if rr.Code != http.StatusFound || rr.Header().Get("Location") != "/setup/calendars" {
		t.Fatalf("setup caldav: got status=%d location=%q body=%q", rr.Code, rr.Header().Get("Location"), rr.Body.String())
	}

	rr = e2eRequest(t, router, secret, http.MethodPost, "/setup/calendars", url.Values{
		"calendar_href":            {calendarHref},
		"default_calendar_href":    {calendarHref},
		"new_default_project_name": {""},
	}, "")
	if rr.Code != http.StatusFound || rr.Header().Get("Location") != "/setup/import" {
		t.Fatalf("setup calendars: got status=%d location=%q body=%q", rr.Code, rr.Header().Get("Location"), rr.Body.String())
	}

	rr = e2eRequest(t, router, secret, http.MethodPost, "/setup/import", nil, "")
	if rr.Code != http.StatusAccepted {
		t.Fatalf("setup import: got status=%d body=%q", rr.Code, rr.Body.String())
	}
	e2eWait(t, func() bool {
		return e2eTaskCount(t, database) == 3
	})

	rr = e2eRequest(t, router, secret, http.MethodPost, "/setup/complete", nil, "")
	if rr.Code != http.StatusFound || rr.Header().Get("Location") != "/" {
		t.Fatalf("setup complete: got status=%d location=%q body=%q", rr.Code, rr.Header().Get("Location"), rr.Body.String())
	}
	if scheduler.startCalls != 1 {
		t.Fatalf("expected scheduler start once, got %d", scheduler.startCalls)
	}

	rr = e2eRequest(t, router, secret, http.MethodGet, "/", nil, "")
	if rr.Code != http.StatusOK {
		t.Fatalf("root after setup: got status=%d body=%q", rr.Code, rr.Body.String())
	}

	rr = e2eRequest(t, router, secret, http.MethodPost, "/tasks/", url.Values{"title": {"Local Created"}}, "tab-create")
	if rr.Code != http.StatusCreated {
		t.Fatalf("task create: got status=%d body=%q", rr.Code, rr.Body.String())
	}
	createdID, createdHref, createdVersion := e2eTaskByTitle(t, database, "Local Created")
	fake.assertRawContains(t, createdHref, "SUMMARY:Local Created")

	rr = e2eRequest(t, router, secret, http.MethodPost, "/tasks/", url.Values{"title": {"Local Subtask Parent"}}, "tab-subtask-parent")
	if rr.Code != http.StatusCreated {
		t.Fatalf("subtask parent create: got status=%d body=%q", rr.Code, rr.Body.String())
	}
	subtaskParentID, _, _ := e2eTaskByTitle(t, database, "Local Subtask Parent")
	subtaskParentUID := e2eTaskUID(t, database, subtaskParentID)
	rr = e2eRequest(t, router, secret, http.MethodPost, "/tasks/"+subtaskParentID+"/subtasks", url.Values{"title": {"Local Subtask Child"}}, "tab-subtask-create")
	if rr.Code != http.StatusCreated {
		t.Fatalf("subtask create: got status=%d body=%q", rr.Code, rr.Body.String())
	}
	subtaskID, subtaskHref, _ := e2eTaskByTitle(t, database, "Local Subtask Child")
	e2eAssertTaskParent(t, database, subtaskID, subtaskParentID)
	fake.assertRawContains(t, subtaskHref, "SUMMARY:Local Subtask Child")
	fake.assertRawContains(t, subtaskHref, "RELATED-TO;RELTYPE=PARENT:"+subtaskParentUID)

	rr = e2eRequest(t, router, secret, http.MethodPatch, "/tasks/"+createdID, url.Values{
		"expected_version": {fmt.Sprint(createdVersion)},
		"title":            {"Local Edited"},
		"description":      {"edited through router"},
		"status":           {"needs-action"},
	}, "tab-edit")
	if rr.Code != http.StatusOK {
		t.Fatalf("task edit: got status=%d body=%q", rr.Code, rr.Body.String())
	}
	createdVersion = e2eTaskVersion(t, database, createdID)
	fake.assertRawContains(t, createdHref, "SUMMARY:Local Edited")

	rr = e2eRequest(t, router, secret, http.MethodPost, "/tasks/"+createdID+"/complete", url.Values{
		"expected_version": {fmt.Sprint(createdVersion)},
	}, "tab-complete")
	if rr.Code != http.StatusOK {
		t.Fatalf("task complete: got status=%d body=%q", rr.Code, rr.Body.String())
	}
	createdVersion = e2eTaskVersion(t, database, createdID)
	e2eAssertTaskStatus(t, database, createdID, "completed")

	rr = e2eRequest(t, router, secret, http.MethodPost, "/tasks/"+createdID+"/reopen", url.Values{
		"expected_version": {fmt.Sprint(createdVersion)},
	}, "tab-reopen")
	if rr.Code != http.StatusOK {
		t.Fatalf("task reopen: got status=%d body=%q", rr.Code, rr.Body.String())
	}
	createdVersion = e2eTaskVersion(t, database, createdID)
	e2eAssertTaskStatus(t, database, createdID, "needs-action")

	rr = e2eRequest(t, router, secret, http.MethodDelete, "/tasks/"+createdID, url.Values{
		"expected_version": {fmt.Sprint(createdVersion)},
	}, "tab-delete")
	if rr.Code != http.StatusOK {
		t.Fatalf("task delete: got status=%d body=%q", rr.Code, rr.Body.String())
	}
	if fake.hasObject(createdHref) {
		t.Fatalf("expected created task to be deleted remotely")
	}

	updatedImportETag := fake.updateObjectTitle(t, "/cal/work/imported.ics", "uid-import", "Remote Updated")
	fake.deleteObject("/cal/work/delete.ics")
	fake.putObject("/cal/work/remote-new.ics", "uid-remote-new", e2eVTODO("uid-remote-new", "Remote New"), `"etag-new-1"`)

	rr = e2eRequest(t, router, secret, http.MethodPost, "/sync/manual", nil, "")
	if rr.Code != http.StatusOK {
		t.Fatalf("manual sync: got status=%d body=%q", rr.Code, rr.Body.String())
	}
	e2eWait(t, func() bool {
		return e2eTaskTitleByUID(t, database, "uid-import") == "Remote Updated" &&
			e2eTaskTitleByUID(t, database, "uid-remote-new") == "Remote New" &&
			!e2eTaskExistsByUID(t, database, "uid-delete")
	})
	if got := e2eTaskETagByUID(t, database, "uid-import"); got != updatedImportETag {
		t.Fatalf("updated clean task etag: got %q want %q", got, updatedImportETag)
	}

	baseConflictRaw := e2eTaskRawByUID(t, database, "uid-conflict")
	localConflictRaw := model.PatchVTODO(baseConflictRaw, model.VTODOPatch{Summary: stringPointer("Local Dirty")})
	if _, err := database.Conn.ExecContext(ctx, `
UPDATE tasks
SET raw_vtodo = ?,
    title = 'Local Dirty',
    sync_status = 'pending',
    updated_at = CURRENT_TIMESTAMP
WHERE uid = 'uid-conflict';
`, localConflictRaw); err != nil {
		t.Fatalf("seed local dirty conflict state: %v", err)
	}
	remoteConflictETag := fake.updateObjectTitle(t, "/cal/work/conflict.ics", "uid-conflict", "Remote Conflict")

	rr = e2eRequest(t, router, secret, http.MethodPost, "/sync/manual", nil, "")
	if rr.Code != http.StatusOK {
		t.Fatalf("manual conflict sync: got status=%d body=%q", rr.Code, rr.Body.String())
	}
	var conflictID string
	e2eWait(t, func() bool {
		conflictID = e2eUnresolvedConflictID(t, database)
		return conflictID != ""
	})
	e2eAssertTaskSyncStatusByUID(t, database, "uid-conflict", "conflict")
	if got := e2eTaskETagByUID(t, database, "uid-conflict"); got != remoteConflictETag {
		t.Fatalf("conflict task etag: got %q want %q", got, remoteConflictETag)
	}

	rr = e2eRequest(t, router, secret, http.MethodGet, "/conflicts", nil, "")
	if rr.Code != http.StatusOK || !strings.Contains(rr.Body.String(), conflictID) {
		t.Fatalf("conflicts list: got status=%d body=%q", rr.Code, rr.Body.String())
	}
	rr = e2eRequest(t, router, secret, http.MethodGet, "/conflicts/"+conflictID, nil, "")
	if rr.Code != http.StatusOK || !strings.Contains(rr.Body.String(), "Remote Conflict") {
		t.Fatalf("conflict detail: got status=%d body=%q", rr.Code, rr.Body.String())
	}

	rr = e2eRequest(t, router, secret, http.MethodPost, "/conflicts/"+conflictID+"/resolve", url.Values{
		"resolution": {"remote"},
	}, "tab-resolve")
	if rr.Code != http.StatusOK {
		t.Fatalf("conflict resolve: got status=%d body=%q", rr.Code, rr.Body.String())
	}
	if got := fake.lastIfMatch("/cal/work/conflict.ics"); got != remoteConflictETag {
		t.Fatalf("conflict resolution if-match: got %q want %q", got, remoteConflictETag)
	}
	e2eAssertConflictResolved(t, database, conflictID)
	e2eAssertTaskSyncStatusByUID(t, database, "uid-conflict", "synced")
	if got := e2eTaskTitleByUID(t, database, "uid-conflict"); got != "Remote Conflict" {
		t.Fatalf("resolved task title: got %q", got)
	}
}

func TestProjectCreateRouteAgainstFakeCalDAVServer(t *testing.T) {
	const calendarHref = "/cal/work/"
	secret := []byte("12345678901234567890123456789012")
	ctx := context.Background()

	fake := newFakeRouterCalDAV(calendarHref, "Work")
	server := httptest.NewServer(fake)
	t.Cleanup(server.Close)

	database, err := db.OpenSQLite(filepath.Join(t.TempDir(), "caldo.db"))
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	t.Cleanup(func() { _ = database.Close() })
	if err := database.SaveCalDAVCredentials(ctx, secret, db.CalDAVCredentials{URL: server.URL, Username: "alice", Password: "secret"}); err != nil {
		t.Fatalf("save credentials: %v", err)
	}
	if err := database.SaveCalDAVServerCapabilities(ctx, db.CalDAVServerCapabilities{CTag: true, FullScan: true}); err != nil {
		t.Fatalf("save capabilities: %v", err)
	}

	router := NewRouter(slog.New(slog.NewTextHandler(io.Discard, nil)), "X-Forwarded-User", testManifest(t), true, secret, database, ctx, nil)
	rr := e2eRequest(t, router, secret, http.MethodPost, "/projects", url.Values{"display_name": {"Local Empty Project"}}, "tab-project-create")
	if rr.Code != http.StatusCreated {
		t.Fatalf("project create: got status=%d body=%q", rr.Code, rr.Body.String())
	}
	e2eAssertProjectExists(t, database, "Local Empty Project", "/local-empty-project/")
	fake.assertCalendar(t, "/local-empty-project/", "Local Empty Project")
	if !strings.Contains(rr.Body.String(), "Local Empty Project") || !strings.Contains(rr.Body.String(), `data-project-create-form`) {
		t.Fatalf("project create response missing refreshed project page: %q", rr.Body.String())
	}
}

type fakeRouterCalDAV struct {
	mu                sync.Mutex
	calendars         map[string]string
	objects           map[string]fakeRouterCalDAVObject
	revision          int
	lastIfMatchByHref map[string]string
}

type fakeRouterCalDAVObject struct {
	Href     string
	UID      string
	RawVTODO string
	ETag     string
}

func newFakeRouterCalDAV(calendarHref string, displayName string) *fakeRouterCalDAV {
	return &fakeRouterCalDAV{
		calendars:         map[string]string{calendarHref: displayName},
		objects:           make(map[string]fakeRouterCalDAVObject),
		revision:          100,
		lastIfMatchByHref: make(map[string]string),
	}
}

func (f *fakeRouterCalDAV) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "PROPFIND":
		if r.Header.Get("Depth") == "0" {
			f.writeCapabilityResponse(w, r.URL.Path)
			return
		}
		f.writeCalendarList(w)
	case "REPORT":
		f.writeVTODOReport(w, r.URL.Path)
	case "MKCALENDAR":
		f.handleMKCalendar(w, r)
	case http.MethodPut:
		f.handlePut(w, r)
	case http.MethodDelete:
		f.handleDelete(w, r)
	case http.MethodGet:
		f.handleGet(w, r)
	default:
		http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
	}
}

func (f *fakeRouterCalDAV) putObject(href string, uid string, rawVTODO string, etag string) {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.objects[href] = fakeRouterCalDAVObject{
		Href:     href,
		UID:      uid,
		RawVTODO: rawVTODO,
		ETag:     etag,
	}
}

func (f *fakeRouterCalDAV) updateObjectTitle(t *testing.T, href string, uid string, title string) string {
	t.Helper()

	f.mu.Lock()
	defer f.mu.Unlock()

	object, ok := f.objects[href]
	if !ok {
		t.Fatalf("fake caldav object %q not found", href)
	}
	object.UID = uid
	object.RawVTODO = e2eVTODO(uid, title)
	object.ETag = f.nextETagLocked()
	f.objects[href] = object
	return object.ETag
}

func (f *fakeRouterCalDAV) deleteObject(href string) {
	f.mu.Lock()
	defer f.mu.Unlock()

	delete(f.objects, href)
}

func (f *fakeRouterCalDAV) hasObject(href string) bool {
	f.mu.Lock()
	defer f.mu.Unlock()

	_, ok := f.objects[href]
	return ok
}

func (f *fakeRouterCalDAV) assertCalendar(t *testing.T, href string, displayName string) {
	t.Helper()

	f.mu.Lock()
	defer f.mu.Unlock()

	got, ok := f.calendars[href]
	if !ok {
		t.Fatalf("fake caldav calendar %q not found", href)
	}
	if got != displayName {
		t.Fatalf("fake caldav calendar %q display name: got %q want %q", href, got, displayName)
	}
}

func (f *fakeRouterCalDAV) assertRawContains(t *testing.T, href string, want string) {
	t.Helper()

	f.mu.Lock()
	defer f.mu.Unlock()

	object, ok := f.objects[href]
	if !ok {
		t.Fatalf("fake caldav object %q not found", href)
	}
	if !strings.Contains(object.RawVTODO, want) {
		t.Fatalf("fake caldav object %q missing %q", href, want)
	}
}

func (f *fakeRouterCalDAV) lastIfMatch(href string) string {
	f.mu.Lock()
	defer f.mu.Unlock()

	return f.lastIfMatchByHref[href]
}

func (f *fakeRouterCalDAV) writeCapabilityResponse(w http.ResponseWriter, href string) {
	w.Header().Set("DAV", "1, calendar-access, sync-collection")
	w.Header().Set("Content-Type", "application/xml; charset=utf-8")
	w.WriteHeader(http.StatusMultiStatus)
	_, _ = w.Write([]byte(`<?xml version="1.0" encoding="utf-8"?>
<d:multistatus xmlns:d="DAV:" xmlns:cs="http://calendarserver.org/ns/">
  <d:response>
    <d:href>` + xmlEscapeText(href) + `</d:href>
    <d:propstat>
      <d:prop>
        <d:getetag>"root-etag"</d:getetag>
        <cs:getctag>"ctag-1"</cs:getctag>
      </d:prop>
      <d:status>HTTP/1.1 200 OK</d:status>
    </d:propstat>
  </d:response>
</d:multistatus>`))
}

func (f *fakeRouterCalDAV) writeCalendarList(w http.ResponseWriter) {
	f.mu.Lock()
	defer f.mu.Unlock()

	var body strings.Builder
	body.WriteString(`<?xml version="1.0" encoding="utf-8"?>` + "\n")
	body.WriteString(`<d:multistatus xmlns:d="DAV:" xmlns:c="urn:ietf:params:xml:ns:caldav">`)
	for href, displayName := range f.calendars {
		body.WriteString(`<d:response><d:href>`)
		body.WriteString(xmlEscapeText(href))
		body.WriteString(`</d:href><d:propstat><d:prop><d:displayname>`)
		body.WriteString(xmlEscapeText(displayName))
		body.WriteString(`</d:displayname><d:resourcetype><d:collection/><c:calendar/></d:resourcetype></d:prop><d:status>HTTP/1.1 200 OK</d:status></d:propstat></d:response>`)
	}
	body.WriteString(`</d:multistatus>`)

	w.Header().Set("Content-Type", "application/xml; charset=utf-8")
	w.WriteHeader(http.StatusMultiStatus)
	_, _ = w.Write([]byte(body.String()))
}

func (f *fakeRouterCalDAV) writeVTODOReport(w http.ResponseWriter, calendarHref string) {
	f.mu.Lock()
	defer f.mu.Unlock()

	var body strings.Builder
	body.WriteString(`<?xml version="1.0" encoding="utf-8"?>` + "\n")
	body.WriteString(`<d:multistatus xmlns:d="DAV:" xmlns:c="urn:ietf:params:xml:ns:caldav">`)
	for _, object := range f.objects {
		if !strings.HasPrefix(object.Href, calendarHref) {
			continue
		}
		body.WriteString(`<d:response><d:href>`)
		body.WriteString(xmlEscapeText(object.Href))
		body.WriteString(`</d:href><d:propstat><d:prop><d:getetag>`)
		body.WriteString(xmlEscapeText(object.ETag))
		body.WriteString(`</d:getetag><c:calendar-data>`)
		body.WriteString(xmlEscapeText(object.RawVTODO))
		body.WriteString(`</c:calendar-data></d:prop><d:status>HTTP/1.1 200 OK</d:status></d:propstat></d:response>`)
	}
	body.WriteString(`</d:multistatus>`)

	w.Header().Set("Content-Type", "application/xml; charset=utf-8")
	w.WriteHeader(http.StatusMultiStatus)
	_, _ = w.Write([]byte(body.String()))
}

func (f *fakeRouterCalDAV) handleMKCalendar(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(io.LimitReader(r.Body, 2<<20))
	if err != nil {
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}
	displayName := e2eDisplayNameFromMKCalendar(body)
	if displayName == "" {
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}
	href := r.URL.Path
	if !strings.HasSuffix(href, "/") {
		href += "/"
	}

	f.mu.Lock()
	defer f.mu.Unlock()

	if _, exists := f.calendars[href]; exists {
		http.Error(w, http.StatusText(http.StatusConflict), http.StatusConflict)
		return
	}
	f.calendars[href] = displayName
	w.WriteHeader(http.StatusCreated)
}

func (f *fakeRouterCalDAV) handlePut(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(io.LimitReader(r.Body, 2<<20))
	if err != nil {
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}

	href := r.URL.Path
	ifMatch := strings.TrimSpace(r.Header.Get("If-Match"))

	f.mu.Lock()
	defer f.mu.Unlock()

	if ifMatch != "" {
		f.lastIfMatchByHref[href] = ifMatch
		existing, ok := f.objects[href]
		if !ok {
			http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
			return
		}
		if strings.TrimSpace(existing.ETag) != ifMatch {
			http.Error(w, http.StatusText(http.StatusPreconditionFailed), http.StatusPreconditionFailed)
			return
		}
	}

	etag := f.nextETagLocked()
	f.objects[href] = fakeRouterCalDAVObject{
		Href:     href,
		UID:      e2eUIDFromVTODO(string(body)),
		RawVTODO: string(body),
		ETag:     etag,
	}

	w.Header().Set("ETag", etag)
	w.WriteHeader(http.StatusCreated)
}

func (f *fakeRouterCalDAV) handleDelete(w http.ResponseWriter, r *http.Request) {
	href := r.URL.Path
	ifMatch := strings.TrimSpace(r.Header.Get("If-Match"))

	f.mu.Lock()
	defer f.mu.Unlock()

	existing, ok := f.objects[href]
	if !ok {
		http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
		return
	}
	if ifMatch != "" && strings.TrimSpace(existing.ETag) != ifMatch {
		http.Error(w, http.StatusText(http.StatusPreconditionFailed), http.StatusPreconditionFailed)
		return
	}
	delete(f.objects, href)
	w.WriteHeader(http.StatusNoContent)
}

func (f *fakeRouterCalDAV) handleGet(w http.ResponseWriter, r *http.Request) {
	href := r.URL.Path

	f.mu.Lock()
	defer f.mu.Unlock()

	object, ok := f.objects[href]
	if !ok {
		http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
		return
	}
	w.Header().Set("ETag", object.ETag)
	w.Header().Set("Content-Type", "text/calendar; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(object.RawVTODO))
}

func (f *fakeRouterCalDAV) nextETagLocked() string {
	f.revision++
	return fmt.Sprintf(`"etag-%d"`, f.revision)
}

func e2eRequest(t *testing.T, router http.Handler, secret []byte, method string, target string, form url.Values, tabID string) *httptest.ResponseRecorder {
	t.Helper()

	var body io.Reader
	if form != nil {
		body = strings.NewReader(form.Encode())
	}
	req := httptest.NewRequest(method, target, body)
	req.Header.Set("X-Forwarded-User", "alice")
	if form != nil {
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	}
	if tabID != "" {
		req.Header.Set("X-Tab-ID", tabID)
	}
	if method == http.MethodPost || method == http.MethodPatch || method == http.MethodDelete || method == http.MethodPut {
		token, err := generateSignedCSRF(secret)
		if err != nil {
			t.Fatalf("generate csrf token: %v", err)
		}
		req.Header.Set(csrfHeaderName, token)
		req.AddCookie(&http.Cookie{Name: csrfCookieName, Value: token})
	}

	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	return rr
}

func e2eVTODO(uid string, title string) string {
	return "BEGIN:VCALENDAR\r\nBEGIN:VTODO\r\nUID:" + uid + "\r\nSUMMARY:" + title + "\r\nSTATUS:NEEDS-ACTION\r\nEND:VTODO\r\nEND:VCALENDAR\r\n"
}

func e2eUIDFromVTODO(raw string) string {
	for _, line := range strings.Split(strings.ReplaceAll(raw, "\r\n", "\n"), "\n") {
		if strings.HasPrefix(strings.ToUpper(line), "UID:") {
			return strings.TrimSpace(line[len("UID:"):])
		}
	}
	return ""
}

func e2eDisplayNameFromMKCalendar(raw []byte) string {
	var request struct {
		Set struct {
			Prop struct {
				DisplayName string `xml:"DAV: displayname"`
			} `xml:"DAV: prop"`
		} `xml:"DAV: set"`
	}
	if err := xml.Unmarshal(raw, &request); err != nil {
		return ""
	}
	return strings.TrimSpace(request.Set.Prop.DisplayName)
}

func xmlEscapeText(value string) string {
	var b bytes.Buffer
	_ = xml.EscapeText(&b, []byte(value))
	return b.String()
}

func e2eWait(t *testing.T, ok func() bool) {
	t.Helper()

	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if ok() {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("condition did not match before deadline")
}

func e2eTaskCount(t *testing.T, database *db.Database) int {
	t.Helper()

	var count int
	if err := database.Conn.QueryRowContext(context.Background(), `SELECT COUNT(*) FROM tasks;`).Scan(&count); err != nil {
		t.Fatalf("count tasks: %v", err)
	}
	return count
}

func e2eTaskByTitle(t *testing.T, database *db.Database, title string) (string, string, int) {
	t.Helper()

	var id, href string
	var version int
	if err := database.Conn.QueryRowContext(context.Background(), `SELECT id, href, server_version FROM tasks WHERE title = ?;`, title).Scan(&id, &href, &version); err != nil {
		t.Fatalf("query task by title %q: %v", title, err)
	}
	return id, href, version
}

func e2eTaskVersion(t *testing.T, database *db.Database, taskID string) int {
	t.Helper()

	var version int
	if err := database.Conn.QueryRowContext(context.Background(), `SELECT server_version FROM tasks WHERE id = ?;`, taskID).Scan(&version); err != nil {
		t.Fatalf("query task version: %v", err)
	}
	return version
}

func e2eTaskUID(t *testing.T, database *db.Database, taskID string) string {
	t.Helper()

	var uid string
	if err := database.Conn.QueryRowContext(context.Background(), `SELECT uid FROM tasks WHERE id = ?;`, taskID).Scan(&uid); err != nil {
		t.Fatalf("query task uid: %v", err)
	}
	return uid
}

func e2eAssertTaskParent(t *testing.T, database *db.Database, taskID string, wantParentID string) {
	t.Helper()

	var parentID sql.NullString
	if err := database.Conn.QueryRowContext(context.Background(), `SELECT parent_id FROM tasks WHERE id = ?;`, taskID).Scan(&parentID); err != nil {
		t.Fatalf("query task parent: %v", err)
	}
	if !parentID.Valid || parentID.String != wantParentID {
		t.Fatalf("task parent: got %#v want %q", parentID, wantParentID)
	}
}

func e2eAssertProjectExists(t *testing.T, database *db.Database, displayName string, calendarHref string) {
	t.Helper()

	var count int
	if err := database.Conn.QueryRowContext(context.Background(), `SELECT COUNT(*) FROM projects WHERE display_name = ? AND calendar_href = ?;`, displayName, calendarHref).Scan(&count); err != nil {
		t.Fatalf("query project %q: %v", displayName, err)
	}
	if count != 1 {
		t.Fatalf("project %q with href %q: got %d rows want 1", displayName, calendarHref, count)
	}
}

func e2eAssertTaskStatus(t *testing.T, database *db.Database, taskID string, want string) {
	t.Helper()

	var status string
	if err := database.Conn.QueryRowContext(context.Background(), `SELECT status FROM tasks WHERE id = ?;`, taskID).Scan(&status); err != nil {
		t.Fatalf("query task status: %v", err)
	}
	if status != want {
		t.Fatalf("task status: got %q want %q", status, want)
	}
}

func e2eTaskTitleByUID(t *testing.T, database *db.Database, uid string) string {
	t.Helper()

	var title string
	err := database.Conn.QueryRowContext(context.Background(), `SELECT title FROM tasks WHERE uid = ?;`, uid).Scan(&title)
	if errorsIsNoRows(err) {
		return ""
	}
	if err != nil {
		t.Fatalf("query task title by uid %q: %v", uid, err)
	}
	return title
}

func e2eTaskExistsByUID(t *testing.T, database *db.Database, uid string) bool {
	t.Helper()

	var count int
	if err := database.Conn.QueryRowContext(context.Background(), `SELECT COUNT(*) FROM tasks WHERE uid = ?;`, uid).Scan(&count); err != nil {
		t.Fatalf("query task exists by uid %q: %v", uid, err)
	}
	return count > 0
}

func e2eTaskETagByUID(t *testing.T, database *db.Database, uid string) string {
	t.Helper()

	var etag sql.NullString
	if err := database.Conn.QueryRowContext(context.Background(), `SELECT etag FROM tasks WHERE uid = ?;`, uid).Scan(&etag); err != nil {
		t.Fatalf("query task etag by uid %q: %v", uid, err)
	}
	return strings.TrimSpace(etag.String)
}

func e2eTaskRawByUID(t *testing.T, database *db.Database, uid string) string {
	t.Helper()

	var raw string
	if err := database.Conn.QueryRowContext(context.Background(), `SELECT raw_vtodo FROM tasks WHERE uid = ?;`, uid).Scan(&raw); err != nil {
		t.Fatalf("query task raw by uid %q: %v", uid, err)
	}
	return raw
}

func e2eAssertTaskSyncStatusByUID(t *testing.T, database *db.Database, uid string, want string) {
	t.Helper()

	var status string
	if err := database.Conn.QueryRowContext(context.Background(), `SELECT sync_status FROM tasks WHERE uid = ?;`, uid).Scan(&status); err != nil {
		t.Fatalf("query task sync status by uid %q: %v", uid, err)
	}
	if status != want {
		t.Fatalf("task sync status: got %q want %q", status, want)
	}
}

func e2eUnresolvedConflictID(t *testing.T, database *db.Database) string {
	t.Helper()

	var id string
	err := database.Conn.QueryRowContext(context.Background(), `
SELECT c.id
FROM conflicts c
JOIN tasks t ON t.id = c.task_id
WHERE c.resolved_at IS NULL AND t.uid = 'uid-conflict'
ORDER BY c.created_at DESC
LIMIT 1;
`).Scan(&id)
	if errorsIsNoRows(err) {
		return ""
	}
	if err != nil {
		t.Fatalf("query unresolved conflict: %v", err)
	}
	return id
}

func e2eAssertConflictResolved(t *testing.T, database *db.Database, conflictID string) {
	t.Helper()

	var resolvedAt sql.NullTime
	if err := database.Conn.QueryRowContext(context.Background(), `SELECT resolved_at FROM conflicts WHERE id = ?;`, conflictID).Scan(&resolvedAt); err != nil {
		t.Fatalf("query conflict resolved_at: %v", err)
	}
	if !resolvedAt.Valid {
		t.Fatalf("expected conflict %q to be resolved", conflictID)
	}
}

func errorsIsNoRows(err error) bool {
	return err == sql.ErrNoRows
}
