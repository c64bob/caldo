package app

import (
	"os"
	"path/filepath"
	"strings"
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
	if cfg.Sync.IntervalSeconds != 300 || cfg.Sync.Enabled {
		t.Fatalf("unexpected default sync config: %+v", cfg.Sync)
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
sync:
  enabled: true
  interval_seconds: 42
  default_principal: "alice@example.com"
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
	if !cfg.Sync.Enabled || cfg.Sync.IntervalSeconds != 42 || cfg.Sync.DefaultPrincipal != "alice@example.com" {
		t.Fatalf("unexpected sync config: %+v", cfg.Sync)
	}
}

func TestLoadConfig_AppliesAndPersistsEnvironmentOverrides(t *testing.T) {
	tmp := t.TempDir()
	cfgPath := filepath.Join(tmp, "config.yaml")
	content := `server:
  port: 8080
  auth_header: "X-Forwarded-User"
caldav:
  server_url: "https://old.example.com"
  default_list: "Tasks"
security:
  encryption_key_file: "/old/key"
database:
  path: "/old/db.sqlite"
sync:
  enabled: false
  interval_seconds: 300
  default_principal: ""
`
	if err := os.WriteFile(cfgPath, []byte(content), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	t.Setenv("CALDO_CONFIG", cfgPath)
	t.Setenv("CALDO_SERVER_PORT", "8181")
	t.Setenv("CALDO_SERVER_AUTH_HEADER", "X-Auth-User")
	t.Setenv("CALDO_CALDAV_SERVER_URL", "https://env.example.com")
	t.Setenv("CALDO_CALDAV_DEFAULT_LIST", "Inbox")
	t.Setenv("CALDO_SECURITY_ENCRYPTION_KEY_FILE", "/env/key")
	t.Setenv("CALDO_DATABASE_PATH", "/env/caldo.db")
	t.Setenv("CALDO_SYNC_ENABLED", "true")
	t.Setenv("CALDO_SYNC_INTERVAL_SECONDS", "120")
	t.Setenv("CALDO_SYNC_DEFAULT_PRINCIPAL", "bob@example.com")

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("load config: %v", err)
	}

	if cfg.Server.Port != 8181 || cfg.Server.AuthHeader != "X-Auth-User" {
		t.Fatalf("unexpected server config: %+v", cfg.Server)
	}
	if cfg.CalDAV.ServerURL != "https://env.example.com" || cfg.CalDAV.DefaultList != "Inbox" {
		t.Fatalf("unexpected caldav config: %+v", cfg.CalDAV)
	}
	if cfg.Security.EncryptionKeyFile != "/env/key" {
		t.Fatalf("unexpected key file: %q", cfg.Security.EncryptionKeyFile)
	}
	if cfg.Database.Path != "/env/caldo.db" {
		t.Fatalf("unexpected db path: %q", cfg.Database.Path)
	}
	if !cfg.Sync.Enabled || cfg.Sync.IntervalSeconds != 120 || cfg.Sync.DefaultPrincipal != "bob@example.com" {
		t.Fatalf("unexpected sync config: %+v", cfg.Sync)
	}

	persisted, err := os.ReadFile(cfgPath)
	if err != nil {
		t.Fatalf("read persisted config: %v", err)
	}
	text := string(persisted)
	for _, expected := range []string{
		"port: 8181",
		"auth_header: \"X-Auth-User\"",
		"server_url: \"https://env.example.com\"",
		"default_list: \"Inbox\"",
		"encryption_key_file: \"/env/key\"",
		"path: \"/env/caldo.db\"",
		"enabled: true",
		"interval_seconds: 120",
		"default_principal: \"bob@example.com\"",
	} {
		if !strings.Contains(text, expected) {
			t.Fatalf("expected persisted config to contain %q, got:\n%s", expected, text)
		}
	}
}

func TestLoadConfig_InvalidEnvironmentOverrideFails(t *testing.T) {
	tmp := t.TempDir()
	cfgPath := filepath.Join(tmp, "config.yaml")
	if err := os.WriteFile(cfgPath, []byte("server:\n  port: 8080\n"), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	t.Setenv("CALDO_CONFIG", cfgPath)
	t.Setenv("CALDO_SERVER_PORT", "not-a-number")

	if _, err := LoadConfig(); err == nil {
		t.Fatal("expected invalid env override error")
	}
}

func TestLoadConfig_MissingFileFails(t *testing.T) {
	t.Setenv("CALDO_CONFIG", filepath.Join(t.TempDir(), "missing.yaml"))
	if _, err := LoadConfig(); err == nil {
		t.Fatal("expected missing file error")
	}
}
