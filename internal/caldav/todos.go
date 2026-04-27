package caldav

import (
	"bytes"
	"context"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"strings"
	"time"
)

const (
	defaultTodoScanTimeout   = 30 * time.Second
	maxTodoScanResponseBytes = 8 << 20
)

// CalendarObject represents one remote VTODO resource including raw payload and metadata.
type CalendarObject struct {
	Href     string
	ETag     string
	RawVTODO string
}

// TodoClient loads VTODO resources from a calendar using a full-scan REPORT.
type TodoClient struct {
	httpClient *http.Client
	timeout    time.Duration
}

// NewTodoClient constructs a TodoClient with sane defaults.
func NewTodoClient(httpClient *http.Client) *TodoClient {
	if httpClient == nil {
		httpClient = &http.Client{}
	}

	return &TodoClient{httpClient: httpClient, timeout: defaultTodoScanTimeout}
}

// ListVTODOs performs a full calendar-query and returns all remote VTODO objects for the calendar href.
func (c *TodoClient) ListVTODOs(ctx context.Context, credentials Credentials, calendarHref string) ([]CalendarObject, error) {
	calendarURL, err := resolveCalendarURL(credentials.URL, calendarHref)
	if err != nil {
		return nil, fmt.Errorf("list vtodos: resolve calendar url: %w", err)
	}

	requestCtx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	request, err := http.NewRequestWithContext(requestCtx, "REPORT", calendarURL, bytes.NewBufferString(vtodoFullScanBody))
	if err != nil {
		return nil, fmt.Errorf("list vtodos: create request: %w", err)
	}
	request.SetBasicAuth(credentials.Username, credentials.Password)
	request.Header.Set("Depth", "1")
	request.Header.Set("Content-Type", "application/xml; charset=utf-8")

	response, err := c.httpClient.Do(request)
	if err != nil {
		return nil, fmt.Errorf("list vtodos: request failed: %w", err)
	}
	defer response.Body.Close()

	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return nil, fmt.Errorf("list vtodos: unexpected status %d", response.StatusCode)
	}

	body, err := io.ReadAll(io.LimitReader(response.Body, maxTodoScanResponseBytes))
	if err != nil {
		return nil, fmt.Errorf("list vtodos: read response: %w", err)
	}

	var parsed todoMultistatus
	if err := xml.Unmarshal(body, &parsed); err != nil {
		return nil, fmt.Errorf("list vtodos: parse response: %w", err)
	}

	objects := make([]CalendarObject, 0, len(parsed.Responses))
	for _, responseEntry := range parsed.Responses {
		raw := strings.TrimSpace(responseEntry.Propstat.Prop.CalendarData)
		if raw == "" || !strings.Contains(strings.ToUpper(raw), "BEGIN:VTODO") {
			continue
		}
		objects = append(objects, CalendarObject{
			Href:     strings.TrimSpace(responseEntry.Href),
			ETag:     strings.TrimSpace(responseEntry.Propstat.Prop.ETag),
			RawVTODO: raw,
		})
	}

	return objects, nil
}

func resolveCalendarURL(baseURL string, calendarHref string) (string, error) {
	base, err := url.Parse(strings.TrimSpace(baseURL))
	if err != nil {
		return "", fmt.Errorf("invalid base url")
	}

	href := strings.TrimSpace(calendarHref)
	if href == "" {
		return "", fmt.Errorf("calendar href is required")
	}

	if strings.HasPrefix(href, "http://") || strings.HasPrefix(href, "https://") {
		return href, nil
	}

	if !strings.HasPrefix(href, "/") {
		href = path.Join(base.Path, href)
	}

	relative, err := url.Parse(href)
	if err != nil {
		return "", fmt.Errorf("invalid calendar href")
	}
	return base.ResolveReference(relative).String(), nil
}

type todoMultistatus struct {
	Responses []todoResponse `xml:"response"`
}

type todoResponse struct {
	Href     string       `xml:"href"`
	Propstat todoPropstat `xml:"propstat"`
}

type todoPropstat struct {
	Prop todoProp `xml:"prop"`
}

type todoProp struct {
	ETag         string `xml:"getetag"`
	CalendarData string `xml:"calendar-data"`
}

const vtodoFullScanBody = `<?xml version="1.0" encoding="utf-8"?>
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
