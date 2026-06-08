package stagecaldav

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"caldo/internal/caldav"
)

func TestServerSupportsCaldoClients(t *testing.T) {
	t.Parallel()

	server := newTestServer(t)
	credentials := caldav.Credentials{URL: server.URL, Username: "stage", Password: "stage"}

	capabilities, err := caldav.NewConnectionTester(server.Client()).TestConnection(context.Background(), credentials)
	if err != nil {
		t.Fatalf("test connection: %v", err)
	}
	if !capabilities.FullScan || !capabilities.ETag || !capabilities.CTag || !capabilities.WebDAVSync {
		t.Fatalf("unexpected capabilities: %#v", capabilities)
	}

	calendarClient := caldav.NewCalendarClient(server.Client())
	calendars, err := calendarClient.ListCalendars(context.Background(), credentials)
	if err != nil {
		t.Fatalf("list calendars: %v", err)
	}
	if len(calendars) != 1 || calendars[0].Href != defaultCalendarHref {
		t.Fatalf("unexpected calendars: %#v", calendars)
	}

	createdCalendar, err := calendarClient.CreateCalendar(context.Background(), credentials, "Inbox")
	if err != nil {
		t.Fatalf("create calendar: %v", err)
	}
	if createdCalendar.Href != "/inbox/" {
		t.Fatalf("unexpected created calendar href: %q", createdCalendar.Href)
	}

	todos := caldav.NewTodoClient(server.Client())
	createdETag, err := todos.PutVTODOCreate(context.Background(), credentials, "/cal/work/client-created.ics", testRawVTODO("client-created"))
	if err != nil {
		t.Fatalf("put create: %v", err)
	}
	if createdETag == "" {
		t.Fatal("expected etag from create")
	}

	objects, err := todos.ListVTODOs(context.Background(), credentials, defaultCalendarHref)
	if err != nil {
		t.Fatalf("list vtodos: %v", err)
	}
	if len(objects) != 2 {
		t.Fatalf("unexpected object count: %d", len(objects))
	}

	if _, err := todos.PutVTODOUpdate(context.Background(), credentials, "/cal/work/client-created.ics", testRawVTODO("client-created"), `"stale"`); !errors.Is(err, caldav.ErrPreconditionFailed) {
		t.Fatalf("expected precondition failure, got %v", err)
	}

	updatedETag, err := todos.PutVTODOUpdate(context.Background(), credentials, "/cal/work/client-created.ics", testRawVTODO("client-created"), createdETag)
	if err != nil {
		t.Fatalf("put update: %v", err)
	}
	if updatedETag == "" || updatedETag == createdETag {
		t.Fatalf("expected new etag, got %q after %q", updatedETag, createdETag)
	}

	if err := todos.DeleteVTODO(context.Background(), credentials, "/cal/work/client-created.ics", updatedETag); err != nil {
		t.Fatalf("delete vtodo: %v", err)
	}
	if err := todos.DeleteVTODO(context.Background(), credentials, "/cal/work/missing.ics", `"missing"`); err != nil {
		t.Fatalf("delete missing vtodo should be successful for client: %v", err)
	}
}

func TestServerRejectsInvalidBasicAuth(t *testing.T) {
	t.Parallel()

	server := newTestServer(t)
	req, err := http.NewRequest("PROPFIND", server.URL, strings.NewReader(""))
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	req.Header.Set("Depth", "0")
	req.SetBasicAuth("stage", "wrong")

	resp, err := server.Client().Do(req)
	if err != nil {
		t.Fatalf("do request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("unexpected status: %d", resp.StatusCode)
	}
}

func TestAdminEndpointsMutateRemoteState(t *testing.T) {
	t.Parallel()

	server := newTestServer(t)

	var created Task
	adminRequest(t, server, http.MethodPost, "/stage/admin/tasks", adminTaskRequest{
		CalendarHref: defaultCalendarHref,
		UID:          "admin-created",
		Title:        "Admin Created",
	}, &created)
	if created.Href != "/cal/work/admin-created.ics" || created.ETag == "" {
		t.Fatalf("unexpected created task: %#v", created)
	}

	var state adminStateResponse
	body := adminRequest(t, server, http.MethodGet, "/stage/admin/state", nil, &state)
	if len(state.Tasks) != 2 {
		t.Fatalf("unexpected task count: %d", len(state.Tasks))
	}
	if bytes.Contains(body, []byte("SUMMARY")) || bytes.Contains(body, []byte("Admin Created")) {
		t.Fatal("admin state must not expose raw task content")
	}

	var updated Task
	adminRequest(t, server, http.MethodPatch, "/stage/admin/tasks", adminTaskRequest{
		Href:  created.Href,
		Title: "Admin Updated",
	}, &updated)
	if updated.ETag == created.ETag {
		t.Fatalf("expected updated etag, got %q", updated.ETag)
	}

	adminRequest(t, server, http.MethodDelete, "/stage/admin/tasks?href="+created.Href, nil, nil)
	adminRequest(t, server, http.MethodPost, "/stage/admin/reset", nil, nil)

	body = adminRequest(t, server, http.MethodGet, "/stage/admin/state", nil, &state)
	if len(state.Tasks) != 1 || !bytes.Contains(body, []byte("stage-seed")) {
		t.Fatalf("unexpected reset state: %s", body)
	}
}

func TestAdminEndpointsRequireToken(t *testing.T) {
	t.Parallel()

	server := newTestServer(t)
	req, err := http.NewRequest(http.MethodGet, server.URL+"/stage/admin/state", nil)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	resp, err := server.Client().Do(req)
	if err != nil {
		t.Fatalf("do request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("unexpected status: %d", resp.StatusCode)
	}
}

func TestDeleteMissingVTODOReturnsNotFound(t *testing.T) {
	t.Parallel()

	server := newTestServer(t)
	req, err := http.NewRequest(http.MethodDelete, server.URL+"/cal/work/missing.ics", nil)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	req.SetBasicAuth("stage", "stage")
	resp, err := server.Client().Do(req)
	if err != nil {
		t.Fatalf("do request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("unexpected status: %d", resp.StatusCode)
	}
}

func newTestServer(t *testing.T) *httptest.Server {
	t.Helper()

	handler, err := New(DefaultConfig())
	if err != nil {
		t.Fatalf("new server: %v", err)
	}
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	return server
}

func adminRequest(t *testing.T, server *httptest.Server, method string, path string, input any, output any) []byte {
	t.Helper()

	var body io.Reader
	if input != nil {
		payload, err := json.Marshal(input)
		if err != nil {
			t.Fatalf("marshal request: %v", err)
		}
		body = bytes.NewReader(payload)
	}
	req, err := http.NewRequest(method, server.URL+path, body)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	req.Header.Set(adminTokenHeader, defaultAdminToken)
	if input != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := server.Client().Do(req)
	if err != nil {
		t.Fatalf("do request: %v", err)
	}
	defer resp.Body.Close()

	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read response: %v", err)
	}
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		t.Fatalf("unexpected status %d body=%q", resp.StatusCode, responseBody)
	}
	if output != nil && len(responseBody) > 0 {
		if err := json.Unmarshal(responseBody, output); err != nil {
			t.Fatalf("unmarshal response: %v", err)
		}
	}
	return responseBody
}

func testRawVTODO(uid string) string {
	return "BEGIN:VCALENDAR\r\nBEGIN:VTODO\r\nUID:" + uid + "\r\nSUMMARY:client task\r\nSTATUS:NEEDS-ACTION\r\nEND:VTODO\r\nEND:VCALENDAR\r\n"
}
