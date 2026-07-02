package stagecaldav

import (
	"bytes"
	"crypto/subtle"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	defaultUsername     = "stage"
	defaultPassword     = "stage"
	defaultAdminToken   = "stage-admin"
	defaultCalendarHref = "/cal/work/"
	defaultCalendarName = "Work"

	maxRequestBodyBytes = 2 << 20
	adminTokenHeader    = "X-Stage-CalDAV-Admin-Token"
)

// Config contains runtime settings for the staging CalDAV server.
type Config struct {
	Username     string
	Password     string
	AdminToken   string
	CalendarHref string
	CalendarName string
}

// Calendar describes one in-memory CalDAV calendar.
type Calendar struct {
	Href        string `json:"href"`
	DisplayName string `json:"display_name"`
}

// Task describes safe task metadata returned by the admin API.
type Task struct {
	CalendarHref string `json:"calendar_href"`
	Href         string `json:"href"`
	UID          string `json:"uid"`
	ETag         string `json:"etag"`
}

type calendarObject struct {
	Task
	RawVTODO string
	Revision int64
}

// Server is an in-memory CalDAV server for local staging smoke tests.
type Server struct {
	mu         sync.Mutex
	username   string
	password   string
	adminToken string
	calendars  map[string]Calendar
	objects    map[string]calendarObject
	deleted    map[string]int64
	revision   int64
}

// DefaultConfig returns safe local defaults for a staging-only server.
func DefaultConfig() Config {
	return Config{
		Username:     defaultUsername,
		Password:     defaultPassword,
		AdminToken:   defaultAdminToken,
		CalendarHref: defaultCalendarHref,
		CalendarName: defaultCalendarName,
	}
}

// New creates an initialized in-memory staging CalDAV server.
func New(config Config) (*Server, error) {
	config = fillDefaults(config)
	calendarHref := normalizeCalendarHref(config.CalendarHref)
	if calendarHref == "" {
		return nil, fmt.Errorf("stage caldav: calendar href is required")
	}

	server := &Server{
		username:   config.Username,
		password:   config.Password,
		adminToken: config.AdminToken,
		calendars:  make(map[string]Calendar),
		objects:    make(map[string]calendarObject),
		deleted:    make(map[string]int64),
		revision:   100,
	}
	server.resetLocked(calendarHref, config.CalendarName)
	return server, nil
}

func fillDefaults(config Config) Config {
	if strings.TrimSpace(config.Username) == "" {
		config.Username = defaultUsername
	}
	if config.Password == "" {
		config.Password = defaultPassword
	}
	if strings.TrimSpace(config.AdminToken) == "" {
		config.AdminToken = defaultAdminToken
	}
	if strings.TrimSpace(config.CalendarHref) == "" {
		config.CalendarHref = defaultCalendarHref
	}
	if strings.TrimSpace(config.CalendarName) == "" {
		config.CalendarName = defaultCalendarName
	}
	return config
}

// ServeHTTP handles both CalDAV/WebDAV requests and token-protected admin requests.
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if strings.HasPrefix(r.URL.Path, "/stage/admin") {
		s.serveAdmin(w, r)
		return
	}

	if !s.authorizeCalDAV(w, r) {
		return
	}

	switch r.Method {
	case "PROPFIND":
		if strings.TrimSpace(r.Header.Get("Depth")) == "0" {
			s.writeCapabilityResponse(w, r.URL.Path)
			return
		}
		if s.isCalendarHref(r.URL.Path) {
			s.writeVTODOETagList(w, r.URL.Path)
			return
		}
		s.writeCalendarList(w)
	case "REPORT":
		s.handleReport(w, r)
	case "MKCALENDAR":
		s.handleMKCalendar(w, r)
	case http.MethodGet:
		s.handleGet(w, r)
	case http.MethodPut:
		s.handlePut(w, r)
	case http.MethodDelete:
		s.handleDelete(w, r)
	default:
		http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
	}
}

func (s *Server) authorizeCalDAV(w http.ResponseWriter, r *http.Request) bool {
	username, password, ok := r.BasicAuth()
	if !ok || !constantTimeEqual(username, s.username) || !constantTimeEqual(password, s.password) {
		w.Header().Set("WWW-Authenticate", `Basic realm="stage-caldav"`)
		http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
		return false
	}
	return true
}

func (s *Server) authorizeAdmin(w http.ResponseWriter, r *http.Request) bool {
	token := strings.TrimSpace(r.Header.Get(adminTokenHeader))
	if token == "" {
		authHeader := strings.TrimSpace(r.Header.Get("Authorization"))
		token = strings.TrimSpace(strings.TrimPrefix(authHeader, "Bearer "))
	}
	if !constantTimeEqual(token, s.adminToken) {
		http.Error(w, http.StatusText(http.StatusForbidden), http.StatusForbidden)
		return false
	}
	return true
}

func constantTimeEqual(left string, right string) bool {
	leftBytes := []byte(left)
	rightBytes := []byte(right)
	if len(leftBytes) != len(rightBytes) {
		return false
	}
	return subtle.ConstantTimeCompare(leftBytes, rightBytes) == 1
}

func (s *Server) serveAdmin(w http.ResponseWriter, r *http.Request) {
	if !s.authorizeAdmin(w, r) {
		return
	}

	switch {
	case r.Method == http.MethodGet && r.URL.Path == "/stage/admin/state":
		s.writeAdminState(w)
	case r.Method == http.MethodPost && r.URL.Path == "/stage/admin/reset":
		s.resetAdminState(w)
	case r.Method == http.MethodPost && r.URL.Path == "/stage/admin/tasks":
		s.createAdminTask(w, r)
	case r.Method == http.MethodPatch && r.URL.Path == "/stage/admin/tasks":
		s.updateAdminTask(w, r)
	case r.Method == http.MethodDelete && r.URL.Path == "/stage/admin/tasks":
		s.deleteAdminTask(w, r)
	default:
		http.NotFound(w, r)
	}
}

type adminStateResponse struct {
	Calendars []Calendar `json:"calendars"`
	Tasks     []Task     `json:"tasks"`
}

func (s *Server) writeAdminState(w http.ResponseWriter) {
	s.mu.Lock()
	defer s.mu.Unlock()

	response := adminStateResponse{
		Calendars: make([]Calendar, 0, len(s.calendars)),
		Tasks:     make([]Task, 0, len(s.objects)),
	}
	for _, calendar := range s.calendars {
		response.Calendars = append(response.Calendars, calendar)
	}
	for _, object := range s.objects {
		response.Tasks = append(response.Tasks, object.Task)
	}
	sort.Slice(response.Calendars, func(i, j int) bool {
		return response.Calendars[i].Href < response.Calendars[j].Href
	})
	sort.Slice(response.Tasks, func(i, j int) bool {
		return response.Tasks[i].Href < response.Tasks[j].Href
	})

	writeJSON(w, http.StatusOK, response)
}

func (s *Server) resetAdminState(w http.ResponseWriter) {
	s.mu.Lock()
	defer s.mu.Unlock()

	var href, displayName string
	for _, calendar := range s.calendars {
		href = calendar.Href
		displayName = calendar.DisplayName
		break
	}
	if href == "" {
		href = defaultCalendarHref
		displayName = defaultCalendarName
	}
	s.resetLocked(href, displayName)
	writeJSON(w, http.StatusOK, map[string]string{"status": "reset"})
}

type adminTaskRequest struct {
	CalendarHref string `json:"calendar_href"`
	Href         string `json:"href"`
	UID          string `json:"uid"`
	Title        string `json:"title"`
	RawVTODO     string `json:"raw_vtodo"`
}

func (s *Server) createAdminTask(w http.ResponseWriter, r *http.Request) {
	var input adminTaskRequest
	if !decodeJSONRequest(w, r, &input) {
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	calendarHref := normalizeCalendarHref(firstNonEmpty(input.CalendarHref, s.firstCalendarHrefLocked()))
	if _, ok := s.calendars[calendarHref]; !ok {
		http.Error(w, "calendar not found", http.StatusNotFound)
		return
	}

	rawVTODO, uid, err := adminRawVTODO(input)
	if err != nil {
		http.Error(w, "invalid task payload", http.StatusBadRequest)
		return
	}
	href := strings.TrimSpace(input.Href)
	if href == "" {
		href = joinCalendarHref(calendarHref, uid)
	}
	if _, exists := s.objects[href]; exists {
		http.Error(w, "task already exists", http.StatusConflict)
		return
	}

	etag := s.nextETagLocked()
	object := calendarObject{
		Task: Task{
			CalendarHref: calendarHref,
			Href:         href,
			UID:          uid,
			ETag:         etag,
		},
		RawVTODO: rawVTODO,
		Revision: s.revision,
	}
	s.objects[object.Href] = object
	delete(s.deleted, object.Href)
	writeJSON(w, http.StatusCreated, object.Task)
}

func (s *Server) updateAdminTask(w http.ResponseWriter, r *http.Request) {
	var input adminTaskRequest
	if !decodeJSONRequest(w, r, &input) {
		return
	}

	href := strings.TrimSpace(input.Href)
	if href == "" {
		http.Error(w, "href is required", http.StatusBadRequest)
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	object, ok := s.objects[href]
	if !ok {
		http.Error(w, "task not found", http.StatusNotFound)
		return
	}

	rawVTODO := strings.TrimSpace(input.RawVTODO)
	uid := strings.TrimSpace(input.UID)
	if rawVTODO == "" {
		if uid == "" {
			uid = object.UID
		}
		rawVTODO = buildVTODO(uid, firstNonEmpty(input.Title, "Updated Remote Task"))
	} else {
		parsedUID := uidFromVTODO(rawVTODO)
		if parsedUID == "" {
			http.Error(w, "invalid task payload", http.StatusBadRequest)
			return
		}
		uid = parsedUID
	}

	object.UID = uid
	object.RawVTODO = rawVTODO
	object.ETag = s.nextETagLocked()
	object.Revision = s.revision
	s.objects[href] = object
	delete(s.deleted, href)
	writeJSON(w, http.StatusOK, object.Task)
}

func (s *Server) deleteAdminTask(w http.ResponseWriter, r *http.Request) {
	href := strings.TrimSpace(r.URL.Query().Get("href"))
	if href == "" {
		http.Error(w, "href is required", http.StatusBadRequest)
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.objects[href]; !ok {
		http.Error(w, "task not found", http.StatusNotFound)
		return
	}
	delete(s.objects, href)
	s.revision++
	s.deleted[href] = s.revision
	w.WriteHeader(http.StatusNoContent)
}

func decodeJSONRequest(w http.ResponseWriter, r *http.Request, output any) bool {
	defer func() { _ = r.Body.Close() }()
	decoder := json.NewDecoder(io.LimitReader(r.Body, maxRequestBodyBytes))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(output); err != nil {
		http.Error(w, "invalid json payload", http.StatusBadRequest)
		return false
	}
	return true
}

func writeJSON(w http.ResponseWriter, statusCode int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	_ = json.NewEncoder(w).Encode(value)
}

func adminRawVTODO(input adminTaskRequest) (string, string, error) {
	rawVTODO := strings.TrimSpace(input.RawVTODO)
	if rawVTODO != "" {
		uid := uidFromVTODO(rawVTODO)
		if uid == "" {
			return "", "", fmt.Errorf("uid missing")
		}
		return rawVTODO, uid, nil
	}

	uid := strings.TrimSpace(input.UID)
	if uid == "" {
		uid = fmt.Sprintf("stage-%d", time.Now().UTC().UnixNano())
	}
	title := firstNonEmpty(input.Title, "Remote Stage Task")
	return buildVTODO(uid, title), uid, nil
}

func (s *Server) resetLocked(calendarHref string, displayName string) {
	calendarHref = normalizeCalendarHref(calendarHref)
	if displayName == "" {
		displayName = defaultCalendarName
	}
	s.calendars = map[string]Calendar{
		calendarHref: {Href: calendarHref, DisplayName: displayName},
	}
	s.objects = make(map[string]calendarObject)
	s.deleted = make(map[string]int64)

	uid := "stage-seed"
	href := joinCalendarHref(calendarHref, uid)
	etag := s.nextETagLocked()
	s.objects[href] = calendarObject{
		Task: Task{
			CalendarHref: calendarHref,
			Href:         href,
			UID:          uid,
			ETag:         etag,
		},
		RawVTODO: buildVTODO(uid, "Stage Seed Task"),
		Revision: s.revision,
	}
}

func (s *Server) firstCalendarHrefLocked() string {
	hrefs := make([]string, 0, len(s.calendars))
	for href := range s.calendars {
		hrefs = append(hrefs, href)
	}
	sort.Strings(hrefs)
	if len(hrefs) == 0 {
		return ""
	}
	return hrefs[0]
}

func (s *Server) currentCTagLocked(calendarHref string) string {
	if _, ok := s.calendars[calendarHref]; !ok {
		return fmt.Sprintf(`"stage-ctag-%d"`, s.revision)
	}
	return fmt.Sprintf(`"stage-ctag-%s-%d"`, strings.Trim(strings.ReplaceAll(calendarHref, "/", "-"), "-"), s.revision)
}

func (s *Server) writeCapabilityResponse(w http.ResponseWriter, href string) {
	s.mu.Lock()
	ctag := s.currentCTagLocked(normalizeCalendarHref(href))
	s.mu.Unlock()

	w.Header().Set("DAV", "1, calendar-access, sync-collection")
	w.Header().Set("Content-Type", "application/xml; charset=utf-8")
	w.WriteHeader(http.StatusMultiStatus)
	_, _ = w.Write([]byte(`<?xml version="1.0" encoding="utf-8"?>
<d:multistatus xmlns:d="DAV:" xmlns:cs="http://calendarserver.org/ns/">
  <d:response>
    <d:href>` + xmlEscapeText(href) + `</d:href>
    <d:propstat>
      <d:prop>
        <d:getetag>"stage-root"</d:getetag>
        <cs:getctag>` + xmlEscapeText(ctag) + `</cs:getctag>
      </d:prop>
      <d:status>HTTP/1.1 200 OK</d:status>
    </d:propstat>
  </d:response>
</d:multistatus>`))
}

func (s *Server) isCalendarHref(href string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	_, ok := s.calendars[normalizeCalendarHref(href)]
	return ok
}

func (s *Server) writeCalendarList(w http.ResponseWriter) {
	s.mu.Lock()
	defer s.mu.Unlock()

	hrefs := make([]string, 0, len(s.calendars))
	for href := range s.calendars {
		hrefs = append(hrefs, href)
	}
	sort.Strings(hrefs)

	var body strings.Builder
	body.WriteString(`<?xml version="1.0" encoding="utf-8"?>` + "\n")
	body.WriteString(`<d:multistatus xmlns:d="DAV:" xmlns:c="urn:ietf:params:xml:ns:caldav">`)
	for _, href := range hrefs {
		calendar := s.calendars[href]
		body.WriteString(`<d:response><d:href>`)
		body.WriteString(xmlEscapeText(calendar.Href))
		body.WriteString(`</d:href><d:propstat><d:prop><d:displayname>`)
		body.WriteString(xmlEscapeText(calendar.DisplayName))
		body.WriteString(`</d:displayname><d:resourcetype><d:collection/><c:calendar/></d:resourcetype></d:prop><d:status>HTTP/1.1 200 OK</d:status></d:propstat></d:response>`)
	}
	body.WriteString(`</d:multistatus>`)

	w.Header().Set("Content-Type", "application/xml; charset=utf-8")
	w.WriteHeader(http.StatusMultiStatus)
	_, _ = w.Write([]byte(body.String()))
}

func (s *Server) handleReport(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(io.LimitReader(r.Body, maxRequestBodyBytes))
	if err != nil {
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}
	_ = r.Body.Close()
	if strings.Contains(strings.ToLower(string(body)), "sync-collection") {
		s.writeSyncCollectionReport(w, r.URL.Path, body)
		return
	}
	s.writeVTODOReport(w, r.URL.Path)
}

func (s *Server) writeVTODOReport(w http.ResponseWriter, calendarHref string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	calendarHref = normalizeCalendarHref(calendarHref)
	if _, ok := s.calendars[calendarHref]; !ok {
		http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
		return
	}

	hrefs := make([]string, 0, len(s.objects))
	for href, object := range s.objects {
		if object.CalendarHref == calendarHref {
			hrefs = append(hrefs, href)
		}
	}
	sort.Strings(hrefs)

	var body strings.Builder
	body.WriteString(`<?xml version="1.0" encoding="utf-8"?>` + "\n")
	body.WriteString(`<d:multistatus xmlns:d="DAV:" xmlns:c="urn:ietf:params:xml:ns:caldav">`)
	for _, href := range hrefs {
		object := s.objects[href]
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

func (s *Server) writeVTODOETagList(w http.ResponseWriter, calendarHref string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	calendarHref = normalizeCalendarHref(calendarHref)
	if _, ok := s.calendars[calendarHref]; !ok {
		http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
		return
	}

	hrefs := make([]string, 0, len(s.objects))
	for href, object := range s.objects {
		if object.CalendarHref == calendarHref {
			hrefs = append(hrefs, href)
		}
	}
	sort.Strings(hrefs)

	var body strings.Builder
	body.WriteString(`<?xml version="1.0" encoding="utf-8"?>` + "\n")
	body.WriteString(`<d:multistatus xmlns:d="DAV:">`)
	body.WriteString(`<d:response><d:href>`)
	body.WriteString(xmlEscapeText(calendarHref))
	body.WriteString(`</d:href><d:propstat><d:prop><d:getetag>`)
	body.WriteString(xmlEscapeText(s.currentCTagLocked(calendarHref)))
	body.WriteString(`</d:getetag></d:prop><d:status>HTTP/1.1 200 OK</d:status></d:propstat></d:response>`)
	for _, href := range hrefs {
		object := s.objects[href]
		body.WriteString(`<d:response><d:href>`)
		body.WriteString(xmlEscapeText(object.Href))
		body.WriteString(`</d:href><d:propstat><d:prop><d:getetag>`)
		body.WriteString(xmlEscapeText(object.ETag))
		body.WriteString(`</d:getetag></d:prop><d:status>HTTP/1.1 200 OK</d:status></d:propstat></d:response>`)
	}
	body.WriteString(`</d:multistatus>`)

	w.Header().Set("Content-Type", "application/xml; charset=utf-8")
	w.WriteHeader(http.StatusMultiStatus)
	_, _ = w.Write([]byte(body.String()))
}

func (s *Server) writeSyncCollectionReport(w http.ResponseWriter, calendarHref string, requestBody []byte) {
	s.mu.Lock()
	defer s.mu.Unlock()

	calendarHref = normalizeCalendarHref(calendarHref)
	if _, ok := s.calendars[calendarHref]; !ok {
		http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
		return
	}

	tokenRevision, ok := syncTokenRevision(syncTokenFromReport(requestBody))
	if !ok {
		http.Error(w, http.StatusText(http.StatusConflict), http.StatusConflict)
		return
	}
	if tokenRevision > s.revision {
		http.Error(w, http.StatusText(http.StatusConflict), http.StatusConflict)
		return
	}

	hrefs := make([]string, 0, len(s.objects)+len(s.deleted))
	for href, object := range s.objects {
		if object.CalendarHref == calendarHref && object.Revision > tokenRevision {
			hrefs = append(hrefs, href)
		}
	}
	for href, revision := range s.deleted {
		if strings.HasPrefix(href, calendarHref) && revision > tokenRevision {
			hrefs = append(hrefs, href)
		}
	}
	sort.Strings(hrefs)

	var body strings.Builder
	body.WriteString(`<?xml version="1.0" encoding="utf-8"?>` + "\n")
	body.WriteString(`<d:multistatus xmlns:d="DAV:" xmlns:c="urn:ietf:params:xml:ns:caldav">`)
	body.WriteString(`<d:sync-token>`)
	body.WriteString(xmlEscapeText(syncTokenForRevision(s.revision)))
	body.WriteString(`</d:sync-token>`)
	for _, href := range hrefs {
		if object, ok := s.objects[href]; ok {
			body.WriteString(`<d:response><d:href>`)
			body.WriteString(xmlEscapeText(object.Href))
			body.WriteString(`</d:href><d:propstat><d:prop><d:getetag>`)
			body.WriteString(xmlEscapeText(object.ETag))
			body.WriteString(`</d:getetag><c:calendar-data>`)
			body.WriteString(xmlEscapeText(object.RawVTODO))
			body.WriteString(`</c:calendar-data></d:prop><d:status>HTTP/1.1 200 OK</d:status></d:propstat></d:response>`)
			continue
		}
		body.WriteString(`<d:response><d:href>`)
		body.WriteString(xmlEscapeText(href))
		body.WriteString(`</d:href><d:status>HTTP/1.1 404 Not Found</d:status></d:response>`)
	}
	body.WriteString(`</d:multistatus>`)

	w.Header().Set("Content-Type", "application/xml; charset=utf-8")
	w.WriteHeader(http.StatusMultiStatus)
	_, _ = w.Write([]byte(body.String()))
}

func (s *Server) handleMKCalendar(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(io.LimitReader(r.Body, maxRequestBodyBytes))
	if err != nil {
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}
	_ = r.Body.Close()

	displayName := displayNameFromMKCalendar(body)
	if displayName == "" {
		displayName = strings.Trim(strings.TrimSpace(r.URL.Path), "/")
	}
	if displayName == "" {
		displayName = "Calendar"
	}
	calendarHref := normalizeCalendarHref(r.URL.Path)

	s.mu.Lock()
	defer s.mu.Unlock()

	s.revision++
	statusCode := http.StatusCreated
	if _, exists := s.calendars[calendarHref]; exists {
		statusCode = http.StatusOK
	}
	s.calendars[calendarHref] = Calendar{Href: calendarHref, DisplayName: displayName}
	w.WriteHeader(statusCode)
}

func displayNameFromMKCalendar(body []byte) string {
	var payload struct {
		DisplayName string `xml:"set>prop>displayname"`
	}
	if err := xml.Unmarshal(body, &payload); err != nil {
		return ""
	}
	return strings.TrimSpace(payload.DisplayName)
}

func (s *Server) handleGet(w http.ResponseWriter, r *http.Request) {
	href := strings.TrimSpace(r.URL.Path)

	s.mu.Lock()
	defer s.mu.Unlock()

	object, ok := s.objects[href]
	if !ok {
		http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
		return
	}
	w.Header().Set("ETag", object.ETag)
	w.Header().Set("Content-Type", "text/calendar; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(object.RawVTODO))
}

func (s *Server) handlePut(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(io.LimitReader(r.Body, maxRequestBodyBytes))
	if err != nil {
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}
	_ = r.Body.Close()

	rawVTODO := strings.TrimSpace(string(body))
	uid := uidFromVTODO(rawVTODO)
	if uid == "" {
		http.Error(w, "invalid vtodo payload", http.StatusBadRequest)
		return
	}
	href := strings.TrimSpace(r.URL.Path)
	ifMatch := strings.TrimSpace(r.Header.Get("If-Match"))

	s.mu.Lock()
	defer s.mu.Unlock()

	existing, exists := s.objects[href]
	if ifMatch != "" {
		if !exists {
			http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
			return
		}
		if strings.TrimSpace(existing.ETag) != ifMatch {
			http.Error(w, http.StatusText(http.StatusPreconditionFailed), http.StatusPreconditionFailed)
			return
		}
	}

	calendarHref := ownerCalendarHrefLocked(s.calendars, href)
	if calendarHref == "" {
		http.Error(w, "calendar not found", http.StatusConflict)
		return
	}

	etag := s.nextETagLocked()
	s.objects[href] = calendarObject{
		Task: Task{
			CalendarHref: calendarHref,
			Href:         href,
			UID:          uid,
			ETag:         etag,
		},
		RawVTODO: rawVTODO,
		Revision: s.revision,
	}
	delete(s.deleted, href)

	w.Header().Set("ETag", etag)
	if exists {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	w.WriteHeader(http.StatusCreated)
}

func (s *Server) handleDelete(w http.ResponseWriter, r *http.Request) {
	href := strings.TrimSpace(r.URL.Path)
	ifMatch := strings.TrimSpace(r.Header.Get("If-Match"))

	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.calendars[normalizeCalendarHref(href)]; ok {
		delete(s.calendars, normalizeCalendarHref(href))
		for objectHref, object := range s.objects {
			if object.CalendarHref == normalizeCalendarHref(href) {
				delete(s.objects, objectHref)
				s.revision++
				s.deleted[objectHref] = s.revision
			}
		}
		w.WriteHeader(http.StatusNoContent)
		return
	}

	existing, ok := s.objects[href]
	if !ok {
		http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
		return
	}
	if ifMatch != "" && strings.TrimSpace(existing.ETag) != ifMatch {
		http.Error(w, http.StatusText(http.StatusPreconditionFailed), http.StatusPreconditionFailed)
		return
	}
	delete(s.objects, href)
	s.revision++
	s.deleted[href] = s.revision
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) nextETagLocked() string {
	s.revision++
	return fmt.Sprintf(`"stage-etag-%d"`, s.revision)
}

func ownerCalendarHrefLocked(calendars map[string]Calendar, objectHref string) string {
	hrefs := make([]string, 0, len(calendars))
	for href := range calendars {
		hrefs = append(hrefs, href)
	}
	sort.Slice(hrefs, func(i, j int) bool {
		return len(hrefs[i]) > len(hrefs[j])
	})
	for _, href := range hrefs {
		if strings.HasPrefix(objectHref, href) {
			return href
		}
	}
	return ""
}

func normalizeCalendarHref(href string) string {
	trimmed := strings.TrimSpace(href)
	if trimmed == "" {
		return ""
	}
	parsed, err := url.Parse(trimmed)
	if err == nil && parsed.Path != "" {
		trimmed = parsed.Path
	}
	if !strings.HasPrefix(trimmed, "/") {
		trimmed = "/" + trimmed
	}
	if !strings.HasSuffix(trimmed, "/") {
		trimmed += "/"
	}
	return path.Clean(trimmed) + "/"
}

func joinCalendarHref(calendarHref string, uid string) string {
	return normalizeCalendarHref(calendarHref) + strings.TrimSpace(uid) + ".ics"
}

func buildVTODO(uid string, title string) string {
	return "BEGIN:VCALENDAR\r\nVERSION:2.0\r\nBEGIN:VTODO\r\nUID:" + sanitizeICalText(uid) + "\r\nSUMMARY:" + sanitizeICalText(title) + "\r\nSTATUS:NEEDS-ACTION\r\nEND:VTODO\r\nEND:VCALENDAR\r\n"
}

func uidFromVTODO(raw string) string {
	for _, line := range strings.Split(strings.ReplaceAll(raw, "\r\n", "\n"), "\n") {
		if strings.HasPrefix(strings.ToUpper(line), "UID:") {
			return strings.TrimSpace(line[len("UID:"):])
		}
	}
	return ""
}

func sanitizeICalText(value string) string {
	replacer := strings.NewReplacer(`\`, `\\`, "\r", " ", "\n", " ", ";", `\;`, ",", `\,`)
	return replacer.Replace(strings.TrimSpace(value))
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func syncTokenFromReport(body []byte) string {
	var payload struct {
		SyncToken string `xml:"sync-token"`
	}
	if err := xml.Unmarshal(body, &payload); err != nil {
		return ""
	}
	return strings.TrimSpace(payload.SyncToken)
}

func syncTokenForRevision(revision int64) string {
	return fmt.Sprintf("stage-sync-%d", revision)
}

func syncTokenRevision(token string) (int64, bool) {
	trimmed := strings.TrimSpace(token)
	if trimmed == "" {
		return 0, true
	}
	if !strings.HasPrefix(trimmed, "stage-sync-") {
		return 0, false
	}
	revision, err := strconv.ParseInt(strings.TrimPrefix(trimmed, "stage-sync-"), 10, 64)
	if err != nil || revision < 0 {
		return 0, false
	}
	return revision, true
}

func xmlEscapeText(value string) string {
	var b bytes.Buffer
	_ = xml.EscapeText(&b, []byte(value))
	return b.String()
}
