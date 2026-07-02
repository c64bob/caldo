package caldav

import (
	"bytes"
	"context"
	"encoding/xml"
	"errors"
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

// ErrSyncCollectionUnsupported indicates that a calendar cannot use WebDAV Sync.
var ErrSyncCollectionUnsupported = errors.New("caldav sync collection unsupported")

// ErrCTagUnsupported indicates that a calendar cannot use CTag/ETag sync.
var ErrCTagUnsupported = errors.New("caldav ctag etag sync unsupported")

// CalendarObject represents one remote VTODO resource including raw payload and metadata.
type CalendarObject struct {
	Href     string
	ETag     string
	RawVTODO string
}

// SyncCollectionResult contains one incremental WebDAV Sync response.
type SyncCollectionResult struct {
	SyncToken    string
	Changed      []CalendarObject
	DeletedHrefs []string
}

// TodoClient loads VTODO resources from a calendar.
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

// SyncCollection performs a WebDAV sync-collection REPORT for one calendar.
func (c *TodoClient) SyncCollection(ctx context.Context, credentials Credentials, calendarHref string, syncToken string) (SyncCollectionResult, error) {
	calendarURL, err := resolveCalendarURL(credentials.URL, calendarHref)
	if err != nil {
		return SyncCollectionResult{}, fmt.Errorf("sync collection: resolve calendar url: %w", err)
	}

	response, err := c.executor.do(ctx, operationPolicy{
		timeout:      timeoutREPORT,
		retryEnabled: true,
	}, func(requestCtx context.Context) (*http.Request, error) {
		request, reqErr := http.NewRequestWithContext(requestCtx, "REPORT", calendarURL, bytes.NewBufferString(syncCollectionBody(syncToken)))
		if reqErr != nil {
			return nil, reqErr
		}
		request.SetBasicAuth(credentials.Username, credentials.Password)
		request.Header.Set("Depth", "1")
		request.Header.Set("Content-Type", "application/xml; charset=utf-8")
		return request, nil
	})
	if err != nil {
		return SyncCollectionResult{}, fmt.Errorf("sync collection: request failed: %w", err)
	}
	defer response.Body.Close()

	if response.StatusCode == http.StatusForbidden ||
		response.StatusCode == http.StatusNotFound ||
		response.StatusCode == http.StatusMethodNotAllowed ||
		response.StatusCode == http.StatusConflict ||
		response.StatusCode == http.StatusNotImplemented {
		return SyncCollectionResult{}, ErrSyncCollectionUnsupported
	}
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return SyncCollectionResult{}, fmt.Errorf("sync collection: unexpected status %d", response.StatusCode)
	}

	body, err := io.ReadAll(io.LimitReader(response.Body, maxTodoScanResponseBytes))
	if err != nil {
		return SyncCollectionResult{}, fmt.Errorf("sync collection: read response: %w", err)
	}

	var parsed syncCollectionMultistatus
	if err := xml.Unmarshal(body, &parsed); err != nil {
		return SyncCollectionResult{}, fmt.Errorf("sync collection: parse response: %w", err)
	}

	result := SyncCollectionResult{SyncToken: strings.TrimSpace(parsed.SyncToken)}
	if result.SyncToken == "" {
		return SyncCollectionResult{}, ErrSyncCollectionUnsupported
	}

	for _, responseEntry := range parsed.Responses {
		href := strings.TrimSpace(responseEntry.Href)
		if href == "" {
			continue
		}
		if syncResponseDeleted(responseEntry) {
			result.DeletedHrefs = append(result.DeletedHrefs, href)
			continue
		}

		object := CalendarObject{Href: href}
		foundOKPropstat := false
		for _, propstat := range responseEntry.Propstats {
			statusCode, statusErr := parseWebDAVStatusCode(propstat.Status)
			if statusErr == nil && statusCode == http.StatusNotFound {
				result.DeletedHrefs = append(result.DeletedHrefs, href)
				foundOKPropstat = false
				break
			}
			if statusErr == nil && (statusCode < http.StatusOK || statusCode >= http.StatusMultipleChoices) {
				continue
			}
			foundOKPropstat = true
			object.ETag = strings.TrimSpace(propstat.Prop.ETag)
			object.RawVTODO = strings.TrimSpace(propstat.Prop.CalendarData)
			break
		}
		if !foundOKPropstat {
			continue
		}
		if object.RawVTODO == "" {
			raw, etag, getErr := c.GetVTODO(ctx, credentials, href)
			if getErr != nil {
				return SyncCollectionResult{}, fmt.Errorf("sync collection: fetch changed vtodo: %w", getErr)
			}
			object.RawVTODO = strings.TrimSpace(raw)
			if strings.TrimSpace(object.ETag) == "" {
				object.ETag = strings.TrimSpace(etag)
			}
		}
		if !strings.Contains(strings.ToUpper(object.RawVTODO), "BEGIN:VTODO") {
			continue
		}
		result.Changed = append(result.Changed, object)
	}

	return result, nil
}

// CalendarCTag loads the current calendar collection CTag.
func (c *TodoClient) CalendarCTag(ctx context.Context, credentials Credentials, calendarHref string) (string, error) {
	calendarURL, err := resolveCalendarURL(credentials.URL, calendarHref)
	if err != nil {
		return "", fmt.Errorf("calendar ctag: resolve calendar url: %w", err)
	}

	response, err := c.executor.do(ctx, operationPolicy{
		timeout:      timeoutPROPFIND,
		retryEnabled: true,
	}, func(requestCtx context.Context) (*http.Request, error) {
		request, reqErr := http.NewRequestWithContext(requestCtx, "PROPFIND", calendarURL, bytes.NewBufferString(calendarCTagBody))
		if reqErr != nil {
			return nil, reqErr
		}
		request.SetBasicAuth(credentials.Username, credentials.Password)
		request.Header.Set("Depth", "0")
		request.Header.Set("Content-Type", "application/xml; charset=utf-8")
		return request, nil
	})
	if err != nil {
		return "", fmt.Errorf("calendar ctag: request failed: %w", err)
	}
	defer response.Body.Close()

	if response.StatusCode == http.StatusForbidden ||
		response.StatusCode == http.StatusNotFound ||
		response.StatusCode == http.StatusMethodNotAllowed ||
		response.StatusCode == http.StatusNotImplemented {
		return "", ErrCTagUnsupported
	}
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return "", fmt.Errorf("calendar ctag: unexpected status %d", response.StatusCode)
	}

	body, err := io.ReadAll(io.LimitReader(response.Body, maxTodoScanResponseBytes))
	if err != nil {
		return "", fmt.Errorf("calendar ctag: read response: %w", err)
	}

	var parsed ctagMultistatus
	if err := xml.Unmarshal(body, &parsed); err != nil {
		return "", fmt.Errorf("calendar ctag: parse response: %w", err)
	}
	for _, responseEntry := range parsed.Responses {
		for _, propstat := range responseEntry.Propstats {
			if !propstatOK(propstat.Status) {
				continue
			}
			ctag := strings.TrimSpace(propstat.Prop.CTag)
			if ctag != "" {
				return ctag, nil
			}
		}
	}
	return "", ErrCTagUnsupported
}

// ListVTODOETags lists resource hrefs and ETags for one calendar without loading bodies.
func (c *TodoClient) ListVTODOETags(ctx context.Context, credentials Credentials, calendarHref string) ([]CalendarObject, error) {
	calendarURL, err := resolveCalendarURL(credentials.URL, calendarHref)
	if err != nil {
		return nil, fmt.Errorf("list vtodo etags: resolve calendar url: %w", err)
	}

	response, err := c.executor.do(ctx, operationPolicy{
		timeout:      timeoutPROPFIND,
		retryEnabled: true,
	}, func(requestCtx context.Context) (*http.Request, error) {
		request, reqErr := http.NewRequestWithContext(requestCtx, "PROPFIND", calendarURL, bytes.NewBufferString(vtodoETagBody))
		if reqErr != nil {
			return nil, reqErr
		}
		request.SetBasicAuth(credentials.Username, credentials.Password)
		request.Header.Set("Depth", "1")
		request.Header.Set("Content-Type", "application/xml; charset=utf-8")
		return request, nil
	})
	if err != nil {
		return nil, fmt.Errorf("list vtodo etags: request failed: %w", err)
	}
	defer response.Body.Close()

	if response.StatusCode == http.StatusForbidden ||
		response.StatusCode == http.StatusNotFound ||
		response.StatusCode == http.StatusMethodNotAllowed ||
		response.StatusCode == http.StatusNotImplemented {
		return nil, ErrCTagUnsupported
	}
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return nil, fmt.Errorf("list vtodo etags: unexpected status %d", response.StatusCode)
	}

	body, err := io.ReadAll(io.LimitReader(response.Body, maxTodoScanResponseBytes))
	if err != nil {
		return nil, fmt.Errorf("list vtodo etags: read response: %w", err)
	}

	var parsed etagMultistatus
	if err := xml.Unmarshal(body, &parsed); err != nil {
		return nil, fmt.Errorf("list vtodo etags: parse response: %w", err)
	}

	objects := make([]CalendarObject, 0, len(parsed.Responses))
	for _, responseEntry := range parsed.Responses {
		href := strings.TrimSpace(responseEntry.Href)
		if href == "" || normalizeHrefForCompare(href) == normalizeHrefForCompare(calendarHref) {
			continue
		}

		var etag string
		for _, propstat := range responseEntry.Propstats {
			if !propstatOK(propstat.Status) {
				continue
			}
			etag = strings.TrimSpace(propstat.Prop.ETag)
			break
		}
		if etag == "" {
			return nil, ErrCTagUnsupported
		}
		objects = append(objects, CalendarObject{Href: href, ETag: etag})
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

type syncCollectionMultistatus struct {
	SyncToken string                   `xml:"sync-token"`
	Responses []syncCollectionResponse `xml:"response"`
}

type syncCollectionResponse struct {
	Href      string                   `xml:"href"`
	Status    string                   `xml:"status"`
	Propstats []syncCollectionPropstat `xml:"propstat"`
}

type syncCollectionPropstat struct {
	Status string             `xml:"status"`
	Prop   syncCollectionProp `xml:"prop"`
}

type syncCollectionProp struct {
	ETag         string `xml:"getetag"`
	CalendarData string `xml:"calendar-data"`
}

type ctagMultistatus struct {
	Responses []ctagResponse `xml:"response"`
}

type ctagResponse struct {
	Href      string         `xml:"href"`
	Propstats []ctagPropstat `xml:"propstat"`
}

type ctagPropstat struct {
	Status string   `xml:"status"`
	Prop   ctagProp `xml:"prop"`
}

type ctagProp struct {
	CTag string `xml:"getctag"`
}

type etagMultistatus struct {
	Responses []etagResponse `xml:"response"`
}

type etagResponse struct {
	Href      string         `xml:"href"`
	Propstats []etagPropstat `xml:"propstat"`
}

type etagPropstat struct {
	Status string   `xml:"status"`
	Prop   etagProp `xml:"prop"`
}

type etagProp struct {
	ETag string `xml:"getetag"`
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

const calendarCTagBody = `<?xml version="1.0" encoding="utf-8"?>
<d:propfind xmlns:d="DAV:" xmlns:cs="http://calendarserver.org/ns/">
  <d:prop>
    <cs:getctag/>
  </d:prop>
</d:propfind>`

const vtodoETagBody = `<?xml version="1.0" encoding="utf-8"?>
<d:propfind xmlns:d="DAV:">
  <d:prop>
    <d:getetag/>
  </d:prop>
</d:propfind>`

func syncCollectionBody(syncToken string) string {
	tokenElement := ""
	if strings.TrimSpace(syncToken) != "" {
		tokenElement = "<d:sync-token>" + xmlEscape(syncToken) + "</d:sync-token>"
	}
	return `<?xml version="1.0" encoding="utf-8"?>
<d:sync-collection xmlns:d="DAV:" xmlns:c="urn:ietf:params:xml:ns:caldav">
  ` + tokenElement + `
  <d:sync-level>1</d:sync-level>
  <d:prop>
    <d:getetag/>
    <c:calendar-data/>
  </d:prop>
</d:sync-collection>`
}

func syncResponseDeleted(response syncCollectionResponse) bool {
	if statusCode, err := parseWebDAVStatusCode(response.Status); err == nil && statusCode == http.StatusNotFound {
		return true
	}
	return false
}

func propstatOK(status string) bool {
	status = strings.TrimSpace(status)
	if status == "" {
		return true
	}
	statusCode, err := parseWebDAVStatusCode(status)
	return err == nil && statusCode >= http.StatusOK && statusCode < http.StatusMultipleChoices
}

func normalizeHrefForCompare(href string) string {
	trimmed := strings.TrimSpace(href)
	if trimmed == "" {
		return ""
	}
	parsed, err := url.Parse(trimmed)
	if err == nil && parsed.Path != "" {
		trimmed = parsed.Path
	}
	if !strings.HasSuffix(trimmed, "/") {
		return trimmed
	}
	return strings.TrimRight(trimmed, "/") + "/"
}
