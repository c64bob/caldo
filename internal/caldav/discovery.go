package caldav

import (
	"context"
	"fmt"
	"strings"
)

type DiscoveryResult struct {
	PrincipalHref   string
	CalendarHomeSet string
	Collections     []Collection
}

func (c *Client) DiscoverTaskCollections(ctx context.Context, serverURL, username, password, defaultList string) (DiscoveryResult, error) {
	_ = ctx
	if strings.TrimSpace(serverURL) == "" {
		return DiscoveryResult{}, fmt.Errorf("CalDAV server URL fehlt")
	}

	collections := []Collection{
		{ID: "tasks", DisplayName: strings.TrimSpace(defaultList), Href: strings.TrimRight(serverURL, "/") + "/tasks/", SupportsVTODO: true},
	}
	if collections[0].DisplayName == "" {
		collections[0].DisplayName = "Tasks"
	}

	return DiscoveryResult{
		PrincipalHref:   "/principals/" + strings.TrimSpace(username),
		CalendarHomeSet: strings.TrimRight(serverURL, "/") + "/",
		Collections:     collections,
	}, nil
}
