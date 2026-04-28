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
	"regexp"
	"strconv"
	"strings"
)

const (
	maxCalendarResponseBodyBytes = 2 << 20
)

var (
	// ErrCalendarDiscoveryFailed indicates calendar listing failed.
	ErrCalendarDiscoveryFailed = errors.New("caldav calendar discovery failed")
	// ErrCalendarCreateFailed indicates calendar creation failed.
	ErrCalendarCreateFailed = errors.New("caldav calendar create failed")
	// ErrCalendarRenameFailed indicates calendar rename failed.
	ErrCalendarRenameFailed = errors.New("caldav calendar rename failed")
	// ErrCalendarDeleteFailed indicates calendar deletion failed.
	ErrCalendarDeleteFailed = errors.New("caldav calendar delete failed")
	slugSanitizer           = regexp.MustCompile(`[^a-z0-9]+`)
)

// Calendar contains the minimum metadata needed to map a CalDAV calendar to a project.
type Calendar struct {
	Href        string
	DisplayName string
}

// CalendarClient discovers and creates calendars over CalDAV/WebDAV.
type CalendarClient struct {
	executor *retryExecutor
}

// NewCalendarClient constructs a calendar client with default timeout settings.
func NewCalendarClient(httpClient *http.Client) *CalendarClient {
	if httpClient == nil {
		httpClient = &http.Client{}
	}

	return &CalendarClient{
		executor: newRetryExecutor(httpClient),
	}
}

// ListCalendars returns calendars available under the configured CalDAV URL.
func (c *CalendarClient) ListCalendars(ctx context.Context, credentials Credentials) ([]Calendar, error) {
	response, err := c.executor.do(ctx, operationPolicy{
		timeout:      timeoutPROPFIND,
		retryEnabled: true,
	}, func(requestCtx context.Context) (*http.Request, error) {
		request, reqErr := http.NewRequestWithContext(requestCtx, "PROPFIND", credentials.URL, bytes.NewBufferString(calendarListProbeBody))
		if reqErr != nil {
			return nil, reqErr
		}
		request.SetBasicAuth(credentials.Username, credentials.Password)
		request.Header.Set("Depth", "1")
		request.Header.Set("Content-Type", "application/xml; charset=utf-8")
		return request, nil
	})
	if err != nil {
		return nil, fmt.Errorf("%w: request failed", ErrCalendarDiscoveryFailed)
	}
	defer response.Body.Close()

	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return nil, fmt.Errorf("%w: unexpected status %d", ErrCalendarDiscoveryFailed, response.StatusCode)
	}

	body, err := io.ReadAll(io.LimitReader(response.Body, maxCalendarResponseBodyBytes))
	if err != nil {
		return nil, fmt.Errorf("%w: read response", ErrCalendarDiscoveryFailed)
	}

	var multistatus multistatusResponse
	if err := xml.Unmarshal(body, &multistatus); err != nil {
		return nil, fmt.Errorf("%w: parse response", ErrCalendarDiscoveryFailed)
	}

	calendars := make([]Calendar, 0, len(multistatus.Responses))
	for _, resp := range multistatus.Responses {
		if len(resp.Propstat.Prop.ResourceType.Calendars) == 0 {
			continue
		}
		href := strings.TrimSpace(resp.Href)
		if href == "" {
			continue
		}

		displayName := strings.TrimSpace(resp.Propstat.Prop.DisplayName)
		if displayName == "" {
			displayName = href
		}

		calendars = append(calendars, Calendar{
			Href:        href,
			DisplayName: displayName,
		})
	}

	return calendars, nil
}

// CreateCalendar creates a new calendar and returns its href plus requested display name.
func (c *CalendarClient) CreateCalendar(ctx context.Context, credentials Credentials, displayName string) (Calendar, error) {
	projectName := strings.TrimSpace(displayName)
	if projectName == "" {
		return Calendar{}, fmt.Errorf("%w: missing display name", ErrCalendarCreateFailed)
	}

	targetURL, href, err := calendarCreateURL(credentials.URL, projectName)
	if err != nil {
		return Calendar{}, err
	}

	response, err := c.executor.do(ctx, operationPolicy{
		timeout:      timeoutMKCAL,
		retryEnabled: false,
	}, func(requestCtx context.Context) (*http.Request, error) {
		request, reqErr := http.NewRequestWithContext(requestCtx, "MKCALENDAR", targetURL, bytes.NewBufferString(calendarCreateBody(projectName)))
		if reqErr != nil {
			return nil, reqErr
		}
		request.SetBasicAuth(credentials.Username, credentials.Password)
		request.Header.Set("Content-Type", "application/xml; charset=utf-8")
		return request, nil
	})
	if err != nil {
		return Calendar{}, fmt.Errorf("%w: request failed", ErrCalendarCreateFailed)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusCreated && response.StatusCode != http.StatusOK {
		return Calendar{}, fmt.Errorf("%w: unexpected status %d", ErrCalendarCreateFailed, response.StatusCode)
	}

	return Calendar{
		Href:        href,
		DisplayName: projectName,
	}, nil
}

// RenameCalendar updates an existing calendar display name via WebDAV PROPPATCH.
func (c *CalendarClient) RenameCalendar(ctx context.Context, credentials Credentials, calendarHref string, displayName string) (Calendar, error) {
	projectName := strings.TrimSpace(displayName)
	if projectName == "" {
		return Calendar{}, fmt.Errorf("%w: missing display name", ErrCalendarRenameFailed)
	}

	calendarURL, err := resolveCalendarURL(credentials.URL, calendarHref)
	if err != nil {
		return Calendar{}, fmt.Errorf("%w: invalid calendar href", ErrCalendarRenameFailed)
	}

	response, err := c.executor.do(ctx, operationPolicy{
		timeout:      timeoutPROPFIND,
		retryEnabled: false,
	}, func(requestCtx context.Context) (*http.Request, error) {
		request, reqErr := http.NewRequestWithContext(requestCtx, "PROPPATCH", calendarURL, bytes.NewBufferString(calendarRenameBody(projectName)))
		if reqErr != nil {
			return nil, reqErr
		}
		request.SetBasicAuth(credentials.Username, credentials.Password)
		request.Header.Set("Content-Type", "application/xml; charset=utf-8")
		return request, nil
	})
	if err != nil {
		return Calendar{}, fmt.Errorf("%w: request failed", ErrCalendarRenameFailed)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK && response.StatusCode != http.StatusNoContent && response.StatusCode != http.StatusMultiStatus {
		return Calendar{}, fmt.Errorf("%w: unexpected status %d", ErrCalendarRenameFailed, response.StatusCode)
	}
	if response.StatusCode == http.StatusMultiStatus {
		body, err := io.ReadAll(io.LimitReader(response.Body, maxCalendarResponseBodyBytes))
		if err != nil {
			return Calendar{}, fmt.Errorf("%w: read response", ErrCalendarRenameFailed)
		}
		if err := validateCalendarRenameMultiStatus(body); err != nil {
			return Calendar{}, err
		}
	}

	return Calendar{
		Href:        strings.TrimSpace(calendarHref),
		DisplayName: projectName,
	}, nil
}

// DeleteCalendar removes an existing calendar via WebDAV DELETE.
func (c *CalendarClient) DeleteCalendar(ctx context.Context, credentials Credentials, calendarHref string) error {
	trimmedHref := strings.TrimSpace(calendarHref)
	if trimmedHref == "" {
		return fmt.Errorf("%w: missing calendar href", ErrCalendarDeleteFailed)
	}

	calendarURL, err := resolveCalendarURL(credentials.URL, trimmedHref)
	if err != nil {
		return fmt.Errorf("%w: invalid calendar href", ErrCalendarDeleteFailed)
	}

	response, err := c.executor.do(ctx, operationPolicy{
		timeout:      timeoutDELETE,
		retryEnabled: false,
	}, func(requestCtx context.Context) (*http.Request, error) {
		request, reqErr := http.NewRequestWithContext(requestCtx, http.MethodDelete, calendarURL, nil)
		if reqErr != nil {
			return nil, reqErr
		}
		request.SetBasicAuth(credentials.Username, credentials.Password)
		return request, nil
	})
	if err != nil {
		return fmt.Errorf("%w: request failed", ErrCalendarDeleteFailed)
	}
	defer response.Body.Close()

	if response.StatusCode == http.StatusNotFound {
		return nil
	}
	if response.StatusCode != http.StatusNoContent && response.StatusCode != http.StatusOK && response.StatusCode != http.StatusAccepted {
		return fmt.Errorf("%w: unexpected status %d", ErrCalendarDeleteFailed, response.StatusCode)
	}

	return nil
}

func calendarCreateURL(baseURL string, name string) (string, string, error) {
	parsed, err := url.Parse(baseURL)
	if err != nil {
		return "", "", fmt.Errorf("%w: invalid base url", ErrCalendarCreateFailed)
	}

	slug := slugSanitizer.ReplaceAllString(strings.ToLower(strings.TrimSpace(name)), "-")
	slug = strings.Trim(slug, "-")
	if slug == "" {
		slug = "calendar"
	}

	basePath := strings.TrimSuffix(parsed.Path, "/")
	newPath := basePath + "/" + slug + "/"
	parsed.Path = newPath

	return parsed.String(), newPath, nil
}

func calendarCreateBody(displayName string) string {
	return fmt.Sprintf(`<?xml version="1.0" encoding="utf-8"?>
<c:mkcalendar xmlns:d="DAV:" xmlns:c="urn:ietf:params:xml:ns:caldav">
  <d:set>
    <d:prop>
      <d:displayname>%s</d:displayname>
    </d:prop>
  </d:set>
</c:mkcalendar>`, xmlEscape(displayName))
}

func calendarRenameBody(displayName string) string {
	return fmt.Sprintf(`<?xml version="1.0" encoding="utf-8"?>
<d:propertyupdate xmlns:d="DAV:">
  <d:set>
    <d:prop>
      <d:displayname>%s</d:displayname>
    </d:prop>
  </d:set>
</d:propertyupdate>`, xmlEscape(displayName))
}

func xmlEscape(v string) string {
	var b strings.Builder
	_ = xml.EscapeText(&b, []byte(v))
	return b.String()
}

type multistatusResponse struct {
	Responses []propfindResponse `xml:"response"`
}

type propfindResponse struct {
	Href     string         `xml:"href"`
	Propstat propstatRecord `xml:"propstat"`
}

type propstatRecord struct {
	Prop propRecord `xml:"prop"`
}

type propRecord struct {
	DisplayName  string             `xml:"displayname"`
	ResourceType resourceTypeRecord `xml:"resourcetype"`
}

type resourceTypeRecord struct {
	Calendars []struct{} `xml:"calendar"`
}

func validateCalendarRenameMultiStatus(body []byte) error {
	var multistatus propPatchMultiStatusResponse
	if err := xml.Unmarshal(body, &multistatus); err != nil {
		return fmt.Errorf("%w: parse multistatus response", ErrCalendarRenameFailed)
	}

	hasDisplayNameStatus := false
	for _, response := range multistatus.Responses {
		for _, propstat := range response.Propstats {
			if propstat.Prop.DisplayName == nil {
				continue
			}
			hasDisplayNameStatus = true

			statusCode, err := parseWebDAVStatusCode(propstat.Status)
			if err != nil {
				return fmt.Errorf("%w: parse propstat status", ErrCalendarRenameFailed)
			}
			if statusCode < http.StatusOK || statusCode >= http.StatusMultipleChoices {
				return fmt.Errorf("%w: propstat status %d", ErrCalendarRenameFailed, statusCode)
			}
		}
	}

	if !hasDisplayNameStatus {
		return fmt.Errorf("%w: missing displayname propstat", ErrCalendarRenameFailed)
	}

	return nil
}

func parseWebDAVStatusCode(statusLine string) (int, error) {
	fields := strings.Fields(strings.TrimSpace(statusLine))
	if len(fields) < 2 {
		return 0, fmt.Errorf("invalid status line %q", statusLine)
	}
	statusCode, err := strconv.Atoi(fields[1])
	if err != nil {
		return 0, fmt.Errorf("invalid status code %q: %w", fields[1], err)
	}
	return statusCode, nil
}

type propPatchMultiStatusResponse struct {
	Responses []propPatchResponse `xml:"response"`
}

type propPatchResponse struct {
	Propstats []propPatchPropstat `xml:"propstat"`
}

type propPatchPropstat struct {
	Status string        `xml:"status"`
	Prop   propPatchProp `xml:"prop"`
}

type propPatchProp struct {
	DisplayName *string `xml:"displayname"`
}

const calendarListProbeBody = `<?xml version="1.0" encoding="utf-8"?>
<d:propfind xmlns:d="DAV:" xmlns:c="urn:ietf:params:xml:ns:caldav">
  <d:prop>
    <d:displayname/>
    <d:resourcetype/>
    <c:supported-calendar-component-set/>
  </d:prop>
</d:propfind>`
