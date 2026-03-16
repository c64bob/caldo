package service

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"caldo/internal/security"
	"caldo/internal/store/sqlite"
)

type SettingsService struct {
	repo *sqlite.DAVAccountsRepo
	key  []byte
}

func NewSettingsService(repo *sqlite.DAVAccountsRepo, key []byte) *SettingsService {
	return &SettingsService{repo: repo, key: key}
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
	if err := testConnectivity(ctx, in.ServerURL, in.Username, in.Password); err != nil {
		return err
	}

	enc, err := security.EncryptAESGCM(s.key, []byte(in.Password))
	if err != nil {
		return fmt.Errorf("encrypt password: %w", err)
	}
	return s.repo.Upsert(ctx, sqlite.DAVAccount{
		PrincipalID:       in.PrincipalID,
		ServerURL:         strings.TrimSpace(in.ServerURL),
		Username:          strings.TrimSpace(in.Username),
		PasswordEncrypted: enc,
	})
}

func testConnectivity(ctx context.Context, serverURL, username, password string) error {
	reqBody := []byte(`<?xml version="1.0"?><d:propfind xmlns:d="DAV:"><d:prop><d:current-user-principal/></d:prop></d:propfind>`)
	req, err := http.NewRequestWithContext(ctx, "PROPFIND", strings.TrimSpace(serverURL), bytes.NewReader(reqBody))
	if err != nil {
		return fmt.Errorf("invalid server URL")
	}
	req.SetBasicAuth(username, password)
	req.Header.Set("Depth", "0")
	req.Header.Set("Content-Type", "application/xml")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("CalDAV-Server nicht erreichbar")
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		return fmt.Errorf("CalDAV-Anmeldung fehlgeschlagen")
	}
	if resp.StatusCode >= 400 {
		return fmt.Errorf("CalDAV-Server antwortet mit HTTP %d", resp.StatusCode)
	}
	return nil
}
