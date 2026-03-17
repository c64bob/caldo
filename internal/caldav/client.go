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

type Client struct {
	httpClient *http.Client
}

func NewClient() *Client {
	return &Client{httpClient: &http.Client{Timeout: 15 * time.Second}}
}

func (c *Client) CreateCollection(ctx context.Context, serverURL, username, password, homeSetHref, collectionID, displayName string) error {
	homeSetURL, err := resolveCollectionURL(serverURL, homeSetHref)
	if err != nil {
		return err
	}
	u, err := url.Parse(homeSetURL)
	if err != nil {
		return fmt.Errorf("CalDAV Collection-URL parsen: %w", err)
	}
	name := strings.Trim(strings.TrimSpace(collectionID), "/")
	if name == "" {
		name = strings.Trim(strings.TrimSpace(displayName), "/")
	}
	if name == "" {
		return fmt.Errorf("Collection-Name fehlt")
	}
	u.Path = path.Join(u.Path, name) + "/"
	escapedDisplayName, err := xmlEscapeText(strings.TrimSpace(displayName))
	if err != nil {
		return fmt.Errorf("CalDAV displayname escapen: %w", err)
	}
	body := []byte(`<?xml version="1.0" encoding="utf-8"?>
<c:mkcalendar xmlns:d="DAV:" xmlns:c="urn:ietf:params:xml:ns:caldav">
  <d:set>
    <d:prop>
      <d:displayname>` + escapedDisplayName + `</d:displayname>
      <c:supported-calendar-component-set><c:comp name="VTODO"/></c:supported-calendar-component-set>
    </d:prop>
  </d:set>
</c:mkcalendar>`)
	req, err := http.NewRequestWithContext(ctx, "MKCALENDAR", u.String(), bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("CalDAV MKCALENDAR request erstellen: %w", err)
	}
	req.SetBasicAuth(username, password)
	req.Header.Set("Content-Type", "application/xml; charset=utf-8")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("CalDAV MKCALENDAR ausführen: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusNoContent {
		raw, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return fmt.Errorf("CalDAV MKCALENDAR fehlgeschlagen (%d): %s", resp.StatusCode, strings.TrimSpace(string(raw)))
	}
	return nil
}

func xmlEscapeText(value string) (string, error) {
	var b bytes.Buffer
	if err := xml.EscapeText(&b, []byte(value)); err != nil {
		return "", err
	}
	return b.String(), nil
}
