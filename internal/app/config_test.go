package app

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadConfig_UsesDefaultsWhenValuesMissing(t *testing.T) {
	tmp := t.TempDir()
	cfgPath := filepath.Join(tmp, "config.yaml")
	if err := os.WriteFile(cfgPath, []byte("caldav:\n  server_url: \"https://example.com\"\n"), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
	t.Setenv("CALDO_CONFIG", cfgPath)

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if cfg.Server.Port != 8080 {
		t.Fatalf("expected default port 8080, got %d", cfg.Server.Port)
	}
	if cfg.Server.AuthHeader != "X-Forwarded-User" {
		t.Fatalf("expected default auth header, got %q", cfg.Server.AuthHeader)
	}
	if cfg.CalDAV.DefaultList != "Tasks" {
		t.Fatalf("expected default list Tasks, got %q", cfg.CalDAV.DefaultList)
	}
	if cfg.Database.Path != "./data/caldo.db" {
		t.Fatalf("expected default db path, got %q", cfg.Database.Path)
	}
}

func TestLoadConfig_ParsesConfiguredValues(t *testing.T) {
	tmp := t.TempDir()
	cfgPath := filepath.Join(tmp, "config.yaml")
	content := `server:
  port: 9090
  auth_header: "X-Test-User"
caldav:
  server_url: "https://nextcloud.example.com"
  default_list: "Inbox"
security:
  encryption_key_file: "/tmp/key"
database:
  path: "/tmp/caldo.db"
`
	if err := os.WriteFile(cfgPath, []byte(content), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
	t.Setenv("CALDO_CONFIG", cfgPath)

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if cfg.Server.Port != 9090 || cfg.Server.AuthHeader != "X-Test-User" {
		t.Fatalf("unexpected server config: %+v", cfg.Server)
	}
	if cfg.CalDAV.ServerURL != "https://nextcloud.example.com" || cfg.CalDAV.DefaultList != "Inbox" {
		t.Fatalf("unexpected caldav config: %+v", cfg.CalDAV)
	}
	if cfg.Security.EncryptionKeyFile != "/tmp/key" {
		t.Fatalf("unexpected key file: %q", cfg.Security.EncryptionKeyFile)
	}
	if cfg.Database.Path != "/tmp/caldo.db" {
		t.Fatalf("unexpected db path: %q", cfg.Database.Path)
	}
}

func TestLoadConfig_MissingFileFails(t *testing.T) {
	t.Setenv("CALDO_CONFIG", filepath.Join(t.TempDir(), "missing.yaml"))
	if _, err := LoadConfig(); err == nil {
		t.Fatal("expected missing file error")
	}
}
