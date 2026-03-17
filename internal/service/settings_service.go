package service

import (
	"bytes"
	"context"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"caldo/internal/security"
	"caldo/internal/store/sqlite"
)

type SettingsService struct {
	repo            *sqlite.DAVAccountsRepo
	key             []byte
	allowedHostPort string
}

func NewSettingsService(repo *sqlite.DAVAccountsRepo, key []byte, configuredServerURL string) *SettingsService {
	allowedHostPort := ""
	if u, err := parseAndValidateServerURL(configuredServerURL); err == nil {
		allowedHostPort = u.Host
	}
	return &SettingsService{repo: repo, key: key, allowedHostPort: allowedHostPort}
}

type SaveDAVAccountInput struct {
	PrincipalID string
	ServerURL   string
	Username    string
	Password    string
}

func (s *SettingsService) SaveDAVAccount(ctx context.Context, in SaveDAVAccountInput) error {
	if strings.TrimSpace(in.PrincipalID) == "" {
		return fmt.Errorf("missing principal")
	}
	if strings.TrimSpace(in.ServerURL) == "" || strings.TrimSpace(in.Username) == "" || strings.TrimSpace(in.Password) == "" {
		return fmt.Errorf("server URL, Benutzername und Passwort sind erforderlich")
	}

	normalizedURL, err := s.validateSubmittedServerURL(in.ServerURL)
	if err != nil {
		return err
	}

	if err := testConnectivity(ctx, normalizedURL, in.Username, in.Password); err != nil {
		return err
	}

	enc, err := security.EncryptAESGCM(s.key, []byte(in.Password))
	if err != nil {
		return fmt.Errorf("encrypt password: %w", err)
	}
	return s.repo.Upsert(ctx, sqlite.DAVAccount{
		PrincipalID:       in.PrincipalID,
		ServerURL:         normalizedURL,
		Username:          strings.TrimSpace(in.Username),
		PasswordEncrypted: enc,
	})
}

func (s *SettingsService) GetDAVAccount(ctx context.Context, principalID string) (sqlite.DAVAccount, bool, error) {
	return s.repo.GetByPrincipal(ctx, principalID)
}

func (s *SettingsService) validateSubmittedServerURL(serverURL string) (string, error) {
	u, err := parseAndValidateServerURL(serverURL)
	if err != nil {
		return "", err
	}
	if s.allowedHostPort != "" && !strings.EqualFold(u.Host, s.allowedHostPort) {
		return "", fmt.Errorf("server URL muss auf den konfigurierten CalDAV-Host zeigen")
	}
	return u.String(), nil
}

func parseAndValidateServerURL(serverURL string) (*url.URL, error) {
	u, err := url.Parse(strings.TrimSpace(serverURL))
	if err != nil {
		return nil, fmt.Errorf("invalid server URL")
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return nil, fmt.Errorf("server URL muss mit http:// oder https:// beginnen")
	}
	if u.Host == "" {
		return nil, fmt.Errorf("invalid server URL")
	}
	if u.User != nil {
		return nil, fmt.Errorf("server URL darf keine eingebetteten Zugangsdaten enthalten")
	}
	if ip := net.ParseIP(u.Hostname()); ip != nil && ip.IsUnspecified() {
		return nil, fmt.Errorf("invalid server URL")
	}
	u.Fragment = ""
	return u, nil
}

func testConnectivity(ctx context.Context, serverURL, username, password string) error {
	reqBody := []byte(`<?xml version="1.0"?><d:propfind xmlns:d="DAV:"><d:prop><d:current-user-principal/></d:prop></d:propfind>`)
	req, err := http.NewRequestWithContext(ctx, "PROPFIND", serverURL, bytes.NewReader(reqBody))
	if err != nil {
		return fmt.Errorf("invalid server URL")
	}
	req.SetBasicAuth(username, password)
	req.Header.Set("Depth", "0")
	req.Header.Set("Content-Type", "application/xml")

	client := &http.Client{
		Timeout: 10 * time.Second,
		CheckRedirect: func(_ *http.Request, _ []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("CalDAV-Server nicht erreichbar")
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		return fmt.Errorf("CalDAV-Anmeldung fehlgeschlagen")
	}
	if resp.StatusCode != http.StatusMultiStatus {
		return fmt.Errorf("CalDAV-Validierung fehlgeschlagen (erwartet HTTP 207, erhalten %d)", resp.StatusCode)
	}
	return nil
}
