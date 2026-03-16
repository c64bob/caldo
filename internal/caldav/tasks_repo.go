package caldav

import (
	"context"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	"caldo/internal/domain"
)

type TasksRepo struct {
	client *Client
}

func NewTasksRepo(client *Client) *TasksRepo {
	return &TasksRepo{client: client}
}

func (r *TasksRepo) ListTasks(ctx context.Context, serverURL, username, password string, collection Collection) ([]domain.Task, error) {
	if strings.TrimSpace(serverURL) == "" {
		return nil, fmt.Errorf("CalDAV server URL fehlt")
	}
	if strings.TrimSpace(username) == "" {
		return nil, fmt.Errorf("CalDAV Benutzername fehlt")
	}
	if strings.TrimSpace(password) == "" {
		return nil, fmt.Errorf("CalDAV Passwort fehlt")
	}
	if !collection.SupportsVTODO {
		return nil, nil
	}

	collectionURL, err := resolveCollectionURL(serverURL, collection.Href)
	if err != nil {
		return nil, err
	}

	tasks, err := r.fetchTasks(ctx, collectionURL, username, password, collection)
	if err != nil {
		return nil, err
	}

	sort.Slice(tasks, func(i, j int) bool { return tasks[i].Summary < tasks[j].Summary })
	return tasks, nil
}

func (r *TasksRepo) CreateTask(ctx context.Context, serverURL, username, password string, collection Collection, task domain.Task) (domain.Task, error) {
	if strings.TrimSpace(task.UID) == "" {
		task.UID = fmt.Sprintf("caldo-%d", time.Now().UnixNano())
	}
	collectionURL, err := resolveCollectionURL(serverURL, collection.Href)
	if err != nil {
		return domain.Task{}, err
	}
	resourceURL := strings.TrimRight(collectionURL, "/") + "/" + task.UID + ".ics"
	task.CollectionID = collection.ID
	task.CollectionHref = collection.Href
	task.Href = resourceURL

	req, err := http.NewRequestWithContext(ctx, http.MethodPut, resourceURL, strings.NewReader(buildVTODOCalendar(task)))
	if err != nil {
		return domain.Task{}, fmt.Errorf("CalDAV PUT request erstellen: %w", err)
	}
	req.SetBasicAuth(username, password)
	req.Header.Set("Content-Type", "text/calendar; charset=utf-8")
	req.Header.Set("If-None-Match", "*")

	resp, err := r.client.httpClient.Do(req)
	if err != nil {
		return domain.Task{}, fmt.Errorf("CalDAV PUT ausführen: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusNoContent {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 2<<10))
		return domain.Task{}, fmt.Errorf("CalDAV PUT fehlgeschlagen (%d): %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	task.ETag = strings.TrimSpace(resp.Header.Get("ETag"))
	if task.Status == "" {
		task.Status = "NEEDS-ACTION"
	}
	return task, nil
}

func (r *TasksRepo) UpdateTask(ctx context.Context, serverURL, username, password string, task domain.Task) (domain.Task, error) {
	taskURL, err := resolveTaskURL(serverURL, task.Href)
	if err != nil {
		return domain.Task{}, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, taskURL, strings.NewReader(buildVTODOCalendar(task)))
	if err != nil {
		return domain.Task{}, fmt.Errorf("CalDAV PUT request erstellen: %w", err)
	}
	req.SetBasicAuth(username, password)
	req.Header.Set("Content-Type", "text/calendar; charset=utf-8")
	if strings.TrimSpace(task.ETag) == "" {
		return domain.Task{}, ErrMissingETag
	}
	req.Header.Set("If-Match", task.ETag)

	resp, err := r.client.httpClient.Do(req)
	if err != nil {
		return domain.Task{}, fmt.Errorf("CalDAV PUT ausführen: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusPreconditionFailed {
		return domain.Task{}, ErrPreconditionFailed
	}
	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 2<<10))
		return domain.Task{}, fmt.Errorf("CalDAV PUT fehlgeschlagen (%d): %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	task.ETag = strings.TrimSpace(resp.Header.Get("ETag"))
	return task, nil
}

func (r *TasksRepo) DeleteTask(ctx context.Context, serverURL, username, password, href, etag string) error {
	taskURL, err := resolveTaskURL(serverURL, href)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, taskURL, nil)
	if err != nil {
		return fmt.Errorf("CalDAV DELETE request erstellen: %w", err)
	}
	req.SetBasicAuth(username, password)
	if strings.TrimSpace(etag) == "" {
		return ErrMissingETag
	}
	req.Header.Set("If-Match", etag)
	resp, err := r.client.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("CalDAV DELETE ausführen: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusPreconditionFailed {
		return ErrPreconditionFailed
	}
	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 2<<10))
		return fmt.Errorf("CalDAV DELETE fehlgeschlagen (%d): %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	return nil
}

func resolveTaskURL(serverURL, href string) (string, error) {
	base, err := url.Parse(strings.TrimSpace(serverURL))
	if err != nil {
		return "", fmt.Errorf("CalDAV server URL ungültig: %w", err)
	}
	rel, err := url.Parse(strings.TrimSpace(href))
	if err != nil {
		return "", ErrInvalidTaskHref
	}
	if rel.Scheme != "" || rel.Host != "" || rel.User != nil {
		return "", ErrInvalidTaskHref
	}
	if !strings.HasPrefix(rel.Path, "/") {
		return "", ErrInvalidTaskHref
	}
	if rel.RawQuery != "" || rel.Fragment != "" {
		return "", ErrInvalidTaskHref
	}
	if !strings.HasSuffix(strings.ToLower(rel.Path), ".ics") {
		return "", ErrInvalidTaskHref
	}
	return base.ResolveReference(rel).String(), nil
}

func buildVTODOCalendar(task domain.Task) string {
	status := strings.TrimSpace(task.Status)
	if status == "" {
		status = "NEEDS-ACTION"
	}
	summary := escapeICalText(task.Summary)
	if summary == "" {
		summary = task.UID
	}
	lines := []string{
		"BEGIN:VCALENDAR",
		"VERSION:2.0",
		"PRODID:-//caldo//task//EN",
		"BEGIN:VTODO",
		"UID:" + escapeICalText(task.UID),
		"SUMMARY:" + summary,
		"STATUS:" + status,
		"DTSTAMP:" + time.Now().UTC().Format("20060102T150405Z"),
	}
	if strings.TrimSpace(task.Description) != "" {
		lines = append(lines, "DESCRIPTION:"+escapeICalText(task.Description))
	}
	if task.Priority > 0 {
		lines = append(lines, fmt.Sprintf("PRIORITY:%d", task.Priority))
	}
	if task.PercentComplete > 0 {
		lines = append(lines, fmt.Sprintf("PERCENT-COMPLETE:%d", task.PercentComplete))
	}
	if len(task.Categories) > 0 {
		cats := make([]string, 0, len(task.Categories))
		for _, cat := range task.Categories {
			if c := escapeICalText(cat); c != "" {
				cats = append(cats, c)
			}
		}
		if len(cats) > 0 {
			lines = append(lines, "CATEGORIES:"+strings.Join(cats, ","))
		}
	}
	if task.Due != nil {
		if task.DueKind == "date" {
			lines = append(lines, "DUE;VALUE=DATE:"+task.Due.Format("20060102"))
		} else {
			lines = append(lines, "DUE:"+task.Due.UTC().Format("20060102T150405Z"))
		}
	}
	lines = append(lines, "END:VTODO", "END:VCALENDAR", "")
	return strings.Join(lines, "\r\n")
}

func escapeICalText(v string) string {
	r := strings.NewReplacer("\\", "\\\\", ";", "\\;", ",", "\\,", "\n", "\\n", "\r", "")
	return r.Replace(strings.TrimSpace(v))
}

func (r *TasksRepo) fetchTasks(ctx context.Context, collectionURL, username, password string, collection Collection) ([]domain.Task, error) {
	reqBody := `<?xml version="1.0" encoding="utf-8"?>
<c:calendar-query xmlns:d="DAV:" xmlns:c="urn:ietf:params:xml:ns:caldav">
  <d:prop>
    <d:getetag/>
    <c:calendar-data/>
  </d:prop>
  <c:filter>
    <c:comp-filter name="VCALENDAR">
      <c:comp-filter name="VTODO"/>
    </c:comp-filter>
  </c:filter>
</c:calendar-query>`

	req, err := http.NewRequestWithContext(ctx, "REPORT", collectionURL, strings.NewReader(reqBody))
	if err != nil {
		return nil, fmt.Errorf("CalDAV REPORT request erstellen: %w", err)
	}
	req.SetBasicAuth(username, password)
	req.Header.Set("Depth", "1")
	req.Header.Set("Content-Type", "application/xml; charset=utf-8")

	resp, err := r.client.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("CalDAV REPORT ausführen: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusMultiStatus {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 2<<10))
		return nil, fmt.Errorf("CalDAV REPORT fehlgeschlagen (%d): %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	var multiStatus reportMultiStatus
	if err := xml.NewDecoder(resp.Body).Decode(&multiStatus); err != nil {
		return nil, fmt.Errorf("CalDAV REPORT Antwort lesen: %w", err)
	}

	tasks := make([]domain.Task, 0, len(multiStatus.Responses))
	for _, resp := range multiStatus.Responses {
		etag, data, ok := resp.calendarDataAndETag()
		if !ok || strings.TrimSpace(data) == "" {
			continue
		}

		task := parseVTODO(data)
		if task.UID == "" {
			continue
		}
		task.CollectionID = collection.ID
		task.CollectionHref = collection.Href
		task.Href = strings.TrimSpace(resp.Href)
		task.ETag = etag
		tasks = append(tasks, task)
	}

	return tasks, nil
}

type reportMultiStatus struct {
	Responses []reportResponse `xml:"response"`
}

type reportResponse struct {
	Href      string           `xml:"href"`
	PropStats []reportPropStat `xml:"propstat"`
}

type reportPropStat struct {
	Status string     `xml:"status"`
	Prop   reportProp `xml:"prop"`
}

type reportProp struct {
	ETag         string `xml:"getetag"`
	CalendarData string `xml:"calendar-data"`
}

func (r reportResponse) calendarDataAndETag() (etag, calendarData string, ok bool) {
	for _, propStat := range r.PropStats {
		if !strings.Contains(propStat.Status, " 200 ") {
			continue
		}
		if strings.TrimSpace(propStat.Prop.CalendarData) == "" {
			continue
		}
		return strings.TrimSpace(propStat.Prop.ETag), propStat.Prop.CalendarData, true
	}
	return "", "", false
}

func parseVTODO(calendarData string) domain.Task {
	var task domain.Task
	for _, line := range unfoldICalLines(calendarData) {
		if line == "" {
			continue
		}
		name, params, value, ok := parseICalProperty(line)
		if !ok {
			continue
		}
		switch name {
		case "UID":
			task.UID = strings.TrimSpace(value)
		case "SUMMARY":
			task.Summary = strings.TrimSpace(value)
		case "DESCRIPTION":
			task.Description = strings.TrimSpace(value)
		case "STATUS":
			task.Status = strings.TrimSpace(value)
		case "PRIORITY":
			task.Priority, _ = strconv.Atoi(strings.TrimSpace(value))
		case "PERCENT-COMPLETE":
			task.PercentComplete, _ = strconv.Atoi(strings.TrimSpace(value))
		case "CATEGORIES":
			for _, cat := range strings.Split(value, ",") {
				cat = strings.TrimSpace(cat)
				if cat != "" {
					task.Categories = append(task.Categories, cat)
				}
			}
		case "DUE":
			if due, dueKind, ok := parseICalTime(strings.TrimSpace(value), params["TZID"]); ok {
				task.Due = &due
				task.DueKind = dueKind
			}
		case "CREATED":
			if created, _, ok := parseICalTime(strings.TrimSpace(value), params["TZID"]); ok {
				task.CreatedAt = &created
			}
		case "LAST-MODIFIED":
			if modified, _, ok := parseICalTime(strings.TrimSpace(value), params["TZID"]); ok {
				task.UpdatedAt = &modified
			}
		}
	}
	if task.Summary == "" {
		task.Summary = task.UID
	}
	return task
}

func parseICalTime(value, tzid string) (time.Time, string, bool) {
	if value == "" {
		return time.Time{}, "", false
	}
	if len(value) == len("20060102") {
		t, err := time.Parse("20060102", value)
		if err != nil {
			return time.Time{}, "", false
		}
		return t, "date", true
	}
	layout := "20060102T150405"
	if strings.HasSuffix(value, "Z") {
		layout += "Z"
		t, err := time.Parse(layout, value)
		if err != nil {
			return time.Time{}, "", false
		}
		return t, "datetime", true
	}

	loc := time.Local
	if strings.TrimSpace(tzid) != "" {
		loadedLoc, err := time.LoadLocation(strings.TrimSpace(tzid))
		if err == nil {
			loc = loadedLoc
		}
	}

	t, err := time.ParseInLocation(layout, value, loc)
	if err != nil {
		return time.Time{}, "", false
	}
	return t, "datetime", true
}

func unfoldICalLines(calendarData string) []string {
	rawLines := strings.Split(calendarData, "\n")
	if len(rawLines) == 0 {
		return nil
	}

	lines := make([]string, 0, len(rawLines))
	for _, rawLine := range rawLines {
		line := strings.TrimRight(rawLine, "\r")
		if len(line) > 0 && (line[0] == ' ' || line[0] == '\t') {
			if len(lines) == 0 {
				continue
			}
			lines[len(lines)-1] += strings.TrimLeft(line, " \t")
			continue
		}
		lines = append(lines, strings.TrimSpace(line))
	}

	return lines
}

func parseICalProperty(line string) (name string, params map[string]string, value string, ok bool) {
	left, value, ok := strings.Cut(line, ":")
	if !ok {
		return "", nil, "", false
	}

	parts := strings.Split(left, ";")
	if len(parts) == 0 {
		return "", nil, "", false
	}

	name = strings.ToUpper(strings.TrimSpace(parts[0]))
	if name == "" {
		return "", nil, "", false
	}

	params = map[string]string{}
	for _, part := range parts[1:] {
		k, v, found := strings.Cut(part, "=")
		if !found {
			continue
		}
		key := strings.ToUpper(strings.TrimSpace(k))
		val := strings.TrimSpace(v)
		if key != "" && val != "" {
			params[key] = strings.Trim(val, `"`)
		}
	}

	return name, params, value, true
}

func resolveCollectionURL(serverURL, href string) (string, error) {
	base, err := url.Parse(strings.TrimSpace(serverURL))
	if err != nil {
		return "", fmt.Errorf("CalDAV server URL ungültig: %w", err)
	}
	rel, err := url.Parse(strings.TrimSpace(href))
	if err != nil {
		return "", fmt.Errorf("CalDAV Collection-URL ungültig: %w", err)
	}
	return base.ResolveReference(rel).String(), nil
}
