package caldav

import (
	"context"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"strconv"
	"strings"
)

type DiscoveryResult struct {
	PrincipalHref   string
	CalendarHomeSet string
	Collections     []Collection
}

func (c *Client) DiscoverTaskCollections(ctx context.Context, serverURL, username, password, defaultList string) (DiscoveryResult, error) {
	if strings.TrimSpace(serverURL) == "" {
		return DiscoveryResult{}, fmt.Errorf("CalDAV server URL fehlt")
	}
	if strings.TrimSpace(username) == "" {
		return DiscoveryResult{}, fmt.Errorf("CalDAV Benutzername fehlt")
	}
	if strings.TrimSpace(password) == "" {
		return DiscoveryResult{}, fmt.Errorf("CalDAV Passwort fehlt")
	}

	principalHref, err := c.discoverPrincipalHref(ctx, serverURL, username, password)
	if err != nil {
		return DiscoveryResult{}, err
	}

	homeSetHref, err := c.discoverCalendarHomeSet(ctx, serverURL, principalHref, username, password)
	if err != nil {
		return DiscoveryResult{}, err
	}

	collections, err := c.discoverCollections(ctx, serverURL, homeSetHref, username, password)
	if err != nil {
		return DiscoveryResult{}, err
	}

	if len(collections) > 1 {
		wanted := strings.TrimSpace(defaultList)
		if wanted != "" {
			for i, collection := range collections {
				if strings.EqualFold(strings.TrimSpace(collection.DisplayName), wanted) {
					collections[0], collections[i] = collections[i], collections[0]
					break
				}
			}
		}
	}

	return DiscoveryResult{
		PrincipalHref:   principalHref,
		CalendarHomeSet: homeSetHref,
		Collections:     collections,
	}, nil
}

func (c *Client) discoverPrincipalHref(ctx context.Context, serverURL, username, password string) (string, error) {
	body := `<?xml version="1.0" encoding="utf-8"?>
<d:propfind xmlns:d="DAV:">
  <d:prop><d:current-user-principal/></d:prop>
</d:propfind>`
	result, err := c.doPropfind(ctx, serverURL, username, password, "0", body)
	if err != nil {
		return "", err
	}
	for _, response := range result.Responses {
		for _, propstat := range response.PropStats {
			if !strings.Contains(propstat.Status, " 200 ") {
				continue
			}
			href := strings.TrimSpace(propstat.Prop.CurrentUserPrincipal.Href)
			if href != "" {
				return href, nil
			}
		}
	}
	return "", fmt.Errorf("CalDAV current-user-principal nicht gefunden")
}

func (c *Client) discoverCalendarHomeSet(ctx context.Context, serverURL, principalHref, username, password string) (string, error) {
	principalURL, err := resolveCollectionURL(serverURL, principalHref)
	if err != nil {
		return "", err
	}
	body := `<?xml version="1.0" encoding="utf-8"?>
<d:propfind xmlns:d="DAV:" xmlns:c="urn:ietf:params:xml:ns:caldav">
  <d:prop><c:calendar-home-set/></d:prop>
</d:propfind>`
	result, err := c.doPropfind(ctx, principalURL, username, password, "0", body)
	if err != nil {
		return "", err
	}
	for _, response := range result.Responses {
		for _, propstat := range response.PropStats {
			if !strings.Contains(propstat.Status, " 200 ") {
				continue
			}
			href := strings.TrimSpace(propstat.Prop.CalendarHomeSet.Href)
			if href != "" {
				return href, nil
			}
		}
	}
	return "", fmt.Errorf("CalDAV calendar-home-set nicht gefunden")
}

func (c *Client) discoverCollections(ctx context.Context, serverURL, homeSetHref, username, password string) ([]Collection, error) {
	homeSetURL, err := resolveCollectionURL(serverURL, homeSetHref)
	if err != nil {
		return nil, err
	}
	body := `<?xml version="1.0" encoding="utf-8"?>
<d:propfind xmlns:d="DAV:" xmlns:c="urn:ietf:params:xml:ns:caldav">
  <d:prop>
    <d:displayname/>
    <d:resourcetype/>
    <c:supported-calendar-component-set/>
  </d:prop>
</d:propfind>`
	result, err := c.doPropfind(ctx, homeSetURL, username, password, "1", body)
	if err != nil {
		return nil, err
	}
	collections := make([]Collection, 0, len(result.Responses))
	usedIDs := map[string]struct{}{}
	baseIDCounts := map[string]int{}
	for _, response := range result.Responses {
		href := strings.TrimSpace(response.Href)
		if href == "" {
			continue
		}
		for _, propstat := range response.PropStats {
			if !strings.Contains(propstat.Status, " 200 ") || !propstat.Prop.ResourceType.HasCalendar() {
				continue
			}
			supportsVTODO := propstat.Prop.SupportedCalendarComponentSet.Supports("VTODO")
			if !supportsVTODO {
				continue
			}
			baseID := deriveCollectionID(href, len(collections)+1)
			id := uniqueCollectionID(baseID, usedIDs, baseIDCounts)
			displayName := strings.TrimSpace(propstat.Prop.DisplayName)
			if displayName == "" {
				displayName = id
			}
			collections = append(collections, Collection{ID: id, DisplayName: displayName, Href: href, SupportsVTODO: true})
			break
		}
	}
	return collections, nil
}

func uniqueCollectionID(baseID string, usedIDs map[string]struct{}, baseIDCounts map[string]int) string {
	baseIDCounts[baseID]++
	sequence := baseIDCounts[baseID]
	candidate := baseID
	if sequence > 1 {
		candidate = baseID + "-" + strconv.Itoa(sequence)
	}
	for {
		if _, exists := usedIDs[candidate]; !exists {
			usedIDs[candidate] = struct{}{}
			baseIDCounts[baseID] = sequence
			return candidate
		}
		sequence++
		candidate = baseID + "-" + strconv.Itoa(sequence)
	}
}

func deriveCollectionID(href string, idx int) string {
	u, err := url.Parse(strings.TrimSpace(href))
	if err == nil {
		candidate := path.Base(strings.TrimRight(u.Path, "/"))
		if value, decodeErr := url.PathUnescape(candidate); decodeErr == nil {
			candidate = value
		}
		candidate = strings.TrimSpace(candidate)
		if candidate != "" && candidate != "." && candidate != "/" {
			return candidate
		}
	}
	return "list-" + strconv.Itoa(idx)
}

func (c *Client) doPropfind(ctx context.Context, endpoint, username, password, depth, body string) (discoveryMultiStatus, error) {
	req, err := http.NewRequestWithContext(ctx, "PROPFIND", endpoint, strings.NewReader(body))
	if err != nil {
		return discoveryMultiStatus{}, fmt.Errorf("CalDAV PROPFIND request erstellen: %w", err)
	}
	req.SetBasicAuth(username, password)
	req.Header.Set("Depth", depth)
	req.Header.Set("Content-Type", "application/xml; charset=utf-8")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return discoveryMultiStatus{}, fmt.Errorf("CalDAV PROPFIND ausführen: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusMultiStatus {
		bodyBytes, _ := io.ReadAll(io.LimitReader(resp.Body, 2<<10))
		return discoveryMultiStatus{}, fmt.Errorf("CalDAV PROPFIND fehlgeschlagen (%d): %s", resp.StatusCode, strings.TrimSpace(string(bodyBytes)))
	}

	var result discoveryMultiStatus
	if err := xml.NewDecoder(resp.Body).Decode(&result); err != nil {
		return discoveryMultiStatus{}, fmt.Errorf("CalDAV PROPFIND Antwort lesen: %w", err)
	}
	return result, nil
}

type discoveryMultiStatus struct {
	Responses []discoveryResponse `xml:"response"`
}

type discoveryResponse struct {
	Href      string              `xml:"href"`
	PropStats []discoveryPropStat `xml:"propstat"`
}

type discoveryPropStat struct {
	Status string        `xml:"status"`
	Prop   discoveryProp `xml:"prop"`
}

type discoveryProp struct {
	DisplayName                   string                                 `xml:"displayname"`
	CurrentUserPrincipal          discoveryHrefProp                      `xml:"current-user-principal"`
	CalendarHomeSet               discoveryHrefProp                      `xml:"calendar-home-set"`
	ResourceType                  discoveryResourceType                  `xml:"resourcetype"`
	SupportedCalendarComponentSet discoverySupportedCalendarComponentSet `xml:"supported-calendar-component-set"`
}

type discoveryHrefProp struct {
	Href string `xml:"href"`
}

type discoveryResourceType struct {
	Calendar *struct{} `xml:"calendar"`
}

func (r discoveryResourceType) HasCalendar() bool {
	return r.Calendar != nil
}

type discoverySupportedCalendarComponentSet struct {
	Components []discoveryCalendarComponent `xml:"comp"`
}

type discoveryCalendarComponent struct {
	Name string `xml:"name,attr"`
}

func (s discoverySupportedCalendarComponentSet) Supports(component string) bool {
	for _, comp := range s.Components {
		if strings.EqualFold(strings.TrimSpace(comp.Name), component) {
			return true
		}
	}
	return false
}
