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
)

const (
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
	executor *retryExecutor
}

// NewTodoClient constructs a TodoClient with sane defaults.
func NewTodoClient(httpClient *http.Client) *TodoClient {
	if httpClient == nil {
		httpClient = &http.Client{}
	}

	return &TodoClient{executor: newRetryExecutor(httpClient)}
}

// ListVTODOs performs a full calendar-query and returns all remote VTODO objects for the calendar href.
func (c *TodoClient) ListVTODOs(ctx context.Context, credentials Credentials, calendarHref string) ([]CalendarObject, error) {
	calendarURL, err := resolveCalendarURL(credentials.URL, calendarHref)
	if err != nil {
		return nil, fmt.Errorf("list vtodos: resolve calendar url: %w", err)
	}

	response, err := c.executor.do(ctx, operationPolicy{
		timeout:      timeoutFullScan,
		retryEnabled: true,
	}, func(requestCtx context.Context) (*http.Request, error) {
		request, reqErr := http.NewRequestWithContext(requestCtx, "REPORT", calendarURL, bytes.NewBufferString(vtodoFullScanBody))
		if reqErr != nil {
			return nil, reqErr
		}
		request.SetBasicAuth(credentials.Username, credentials.Password)
		request.Header.Set("Depth", "1")
		request.Header.Set("Content-Type", "application/xml; charset=utf-8")
		return request, nil
	})
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

// GetVTODO fetches one VTODO object body and optional ETag metadata.
func (c *TodoClient) GetVTODO(ctx context.Context, credentials Credentials, todoHref string) (string, string, error) {
	resourceURL, err := resolveCalendarURL(credentials.URL, todoHref)
	if err != nil {
		return "", "", fmt.Errorf("get vtodo: resolve resource url: %w", err)
	}

	response, err := c.executor.do(ctx, operationPolicy{
		timeout:      timeoutGET,
		retryEnabled: true,
	}, func(requestCtx context.Context) (*http.Request, error) {
		request, reqErr := http.NewRequestWithContext(requestCtx, http.MethodGet, resourceURL, nil)
		if reqErr != nil {
			return nil, reqErr
		}
		request.SetBasicAuth(credentials.Username, credentials.Password)
		return request, nil
	})
	if err != nil {
		return "", "", fmt.Errorf("get vtodo: request failed: %w", err)
	}
	defer response.Body.Close()

	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return "", "", fmt.Errorf("get vtodo: unexpected status %d", response.StatusCode)
	}

	body, err := io.ReadAll(io.LimitReader(response.Body, maxTodoScanResponseBytes))
	if err != nil {
		return "", "", fmt.Errorf("get vtodo: read response: %w", err)
	}

	return string(body), strings.TrimSpace(response.Header.Get("ETag")), nil
}

// PutVTODOCreate creates a new VTODO object at the target href without retries.
func (c *TodoClient) PutVTODOCreate(ctx context.Context, credentials Credentials, todoHref string, rawVTODO string) (string, error) {
	return c.putVTODO(ctx, credentials, todoHref, rawVTODO, "", false)
}

// PutVTODOUpdate updates an existing VTODO object using If-Match and retries.
func (c *TodoClient) PutVTODOUpdate(ctx context.Context, credentials Credentials, todoHref string, rawVTODO string, etag string) (string, error) {
	return c.putVTODO(ctx, credentials, todoHref, rawVTODO, etag, true)
}

// DeleteVTODO deletes a VTODO object and treats 404 as a successful outcome.
func (c *TodoClient) DeleteVTODO(ctx context.Context, credentials Credentials, todoHref string, etag string) error {
	resourceURL, err := resolveCalendarURL(credentials.URL, todoHref)
	if err != nil {
		return fmt.Errorf("delete vtodo: resolve resource url: %w", err)
	}

	response, err := c.executor.do(ctx, operationPolicy{
		timeout:      timeoutDELETE,
		retryEnabled: true,
	}, func(requestCtx context.Context) (*http.Request, error) {
		request, reqErr := http.NewRequestWithContext(requestCtx, http.MethodDelete, resourceURL, nil)
		if reqErr != nil {
			return nil, reqErr
		}
		request.SetBasicAuth(credentials.Username, credentials.Password)
		if strings.TrimSpace(etag) != "" {
			request.Header.Set("If-Match", etag)
		}
		return request, nil
	})
	if err != nil {
		return fmt.Errorf("delete vtodo: request failed: %w", err)
	}
	defer response.Body.Close()

	if response.StatusCode == http.StatusNotFound {
		return nil
	}
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return fmt.Errorf("delete vtodo: unexpected status %d", response.StatusCode)
	}

	return nil
}

func (c *TodoClient) putVTODO(ctx context.Context, credentials Credentials, todoHref string, rawVTODO string, etag string, update bool) (string, error) {
	resourceURL, err := resolveCalendarURL(credentials.URL, todoHref)
	if err != nil {
		return "", fmt.Errorf("put vtodo: resolve resource url: %w", err)
	}

	response, err := c.executor.do(ctx, operationPolicy{
		timeout:      timeoutPUT,
		retryEnabled: update,
	}, func(requestCtx context.Context) (*http.Request, error) {
		request, reqErr := http.NewRequestWithContext(requestCtx, http.MethodPut, resourceURL, bytes.NewBufferString(rawVTODO))
		if reqErr != nil {
			return nil, reqErr
		}
		request.SetBasicAuth(credentials.Username, credentials.Password)
		request.Header.Set("Content-Type", "text/calendar; charset=utf-8")
		if update {
			request.Header.Set("If-Match", etag)
		}
		return request, nil
	})
	if err != nil {
		return "", fmt.Errorf("put vtodo: request failed: %w", err)
	}
	defer response.Body.Close()

	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return "", fmt.Errorf("put vtodo: unexpected status %d", response.StatusCode)
	}

	return strings.TrimSpace(response.Header.Get("ETag")), nil
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
