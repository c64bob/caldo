package caldav

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
)

const (
	maxCapabilityProbeBodyBytes = 1 << 20
)

var (
	// ErrConnectionTestFailed indicates the CalDAV capability probe failed.
	ErrConnectionTestFailed = errors.New("caldav connection test failed")
)

// Credentials contains values used for CalDAV authentication.
type Credentials struct {
	URL      string
	Username string
	Password string
}

// ServerCapabilities describes globally detected CalDAV/WebDAV capabilities.
type ServerCapabilities struct {
	WebDAVSync bool `json:"webdav_sync"`
	CTag       bool `json:"ctag"`
	ETag       bool `json:"etag"`
	FullScan   bool `json:"fullscan"`
}

// ConnectionTester executes a live WebDAV request to verify connectivity and detect capabilities.
type ConnectionTester struct {
	executor *retryExecutor
}

// NewConnectionTester creates a tester with defaults that satisfy architecture timeout constraints.
func NewConnectionTester(httpClient *http.Client) *ConnectionTester {
	if httpClient == nil {
		httpClient = &http.Client{}
	}

	return &ConnectionTester{
		executor: newRetryExecutor(httpClient),
	}
}

// TestConnection runs a PROPFIND request and derives global CalDAV capability flags.
func (t *ConnectionTester) TestConnection(ctx context.Context, credentials Credentials) (ServerCapabilities, error) {
	if strings.TrimSpace(credentials.URL) == "" {
		return ServerCapabilities{}, fmt.Errorf("%w: missing caldav url", ErrConnectionTestFailed)
	}
	if strings.TrimSpace(credentials.Username) == "" {
		return ServerCapabilities{}, fmt.Errorf("%w: missing username", ErrConnectionTestFailed)
	}
	if credentials.Password == "" {
		return ServerCapabilities{}, fmt.Errorf("%w: missing password", ErrConnectionTestFailed)
	}

	response, err := t.executor.do(ctx, operationPolicy{
		timeout:      timeoutPROPFIND,
		retryEnabled: true,
	}, func(requestCtx context.Context) (*http.Request, error) {
		request, reqErr := http.NewRequestWithContext(requestCtx, "PROPFIND", credentials.URL, bytes.NewBufferString(capabilityProbeBody))
		if reqErr != nil {
			return nil, reqErr
		}
		request.SetBasicAuth(credentials.Username, credentials.Password)
		request.Header.Set("Depth", "0")
		request.Header.Set("Content-Type", "application/xml; charset=utf-8")
		return request, nil
	})
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			return ServerCapabilities{}, fmt.Errorf("%w: timeout", ErrConnectionTestFailed)
		}
		return ServerCapabilities{}, fmt.Errorf("%w: request failed", ErrConnectionTestFailed)
	}
	defer response.Body.Close()

	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return ServerCapabilities{}, fmt.Errorf("%w: unexpected status %d", ErrConnectionTestFailed, response.StatusCode)
	}

	bodyBytes, err := io.ReadAll(io.LimitReader(response.Body, maxCapabilityProbeBodyBytes))
	if err != nil {
		return ServerCapabilities{}, fmt.Errorf("%w: read response", ErrConnectionTestFailed)
	}
	if !isCalDAVResponse(response.Header.Get("DAV"), response.Header.Get("Content-Type"), string(bodyBytes)) {
		return ServerCapabilities{}, fmt.Errorf("%w: response is not a caldav endpoint", ErrConnectionTestFailed)
	}

	return detectCapabilities(response.Header.Get("DAV"), string(bodyBytes)), nil
}

func isCalDAVResponse(davHeader string, contentType string, responseBody string) bool {
	lowerDAV := strings.ToLower(davHeader)
	lowerContentType := strings.ToLower(contentType)
	lowerBody := strings.ToLower(responseBody)

	if !strings.Contains(lowerContentType, "xml") {
		return false
	}

	if strings.Contains(lowerDAV, "calendar-access") || strings.Contains(lowerDAV, "addressbook") {
		return true
	}

	return strings.Contains(lowerBody, "multistatus") &&
		(strings.Contains(lowerBody, "xmlns:d=\"dav:\"") || strings.Contains(lowerBody, "xmlns=\"dav:\""))
}

func detectCapabilities(davHeader string, responseBody string) ServerCapabilities {
	lowerDAV := strings.ToLower(davHeader)
	lowerBody := strings.ToLower(responseBody)

	return ServerCapabilities{
		WebDAVSync: strings.Contains(lowerDAV, "sync-collection"),
		CTag:       strings.Contains(lowerBody, "getctag"),
		ETag:       strings.Contains(lowerBody, "getetag"),
		FullScan:   true,
	}
}

const capabilityProbeBody = `<?xml version="1.0" encoding="utf-8"?>
<d:propfind xmlns:d="DAV:" xmlns:c="urn:ietf:params:xml:ns:caldav" xmlns:cs="http://calendarserver.org/ns/">
  <d:prop>
    <d:current-user-principal/>
    <d:principal-URL/>
    <c:calendar-home-set/>
    <d:supported-report-set/>
    <cs:getctag/>
    <d:getetag/>
  </d:prop>
</d:propfind>`
