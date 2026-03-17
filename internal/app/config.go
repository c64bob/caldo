package app

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

type Config struct {
	Server struct {
		Port       int
		AuthHeader string
	}
	CalDAV struct {
		ServerURL   string
		DefaultList string
	}
	Security struct {
		EncryptionKeyFile string
	}
	Database struct {
		Path string
	}
	Sync struct {
		Enabled          bool
		IntervalSeconds  int
		DefaultPrincipal string
	}
}

func LoadConfig() (Config, error) {
	cfg := Config{}
	cfg.Server.Port = 8080
	cfg.Server.AuthHeader = "X-Forwarded-User"
	cfg.CalDAV.DefaultList = "Tasks"
	cfg.Database.Path = "./data/caldo.db"
	cfg.Sync.IntervalSeconds = 300

	path := os.Getenv("CALDO_CONFIG")
	if path == "" {
		path = "configs/config.example.yaml"
	}
	path = filepath.Clean(path)
	f, err := os.Open(path)
	if err != nil {
		return Config{}, fmt.Errorf("read config %q: %w", path, err)
	}
	defer f.Close()

	section := ""
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if strings.HasSuffix(line, ":") && !strings.Contains(line, " ") {
			section = strings.TrimSuffix(line, ":")
			continue
		}
		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		value := strings.Trim(strings.TrimSpace(parts[1]), "\"")
		switch section {
		case "server":
			switch key {
			case "port":
				if n, err := strconv.Atoi(value); err == nil {
					cfg.Server.Port = n
				}
			case "auth_header":
				cfg.Server.AuthHeader = value
			}
		case "caldav":
			switch key {
			case "server_url":
				cfg.CalDAV.ServerURL = value
			case "default_list":
				cfg.CalDAV.DefaultList = value
			}
		case "security":
			if key == "encryption_key_file" {
				cfg.Security.EncryptionKeyFile = value
			}
		case "database":
			if key == "path" {
				cfg.Database.Path = value
			}
		case "sync":
			switch key {
			case "enabled":
				cfg.Sync.Enabled = strings.EqualFold(value, "true")
			case "interval_seconds":
				if n, err := strconv.Atoi(value); err == nil && n > 0 {
					cfg.Sync.IntervalSeconds = n
				}
			case "default_principal":
				cfg.Sync.DefaultPrincipal = value
			}
		}
	}
	if err := scanner.Err(); err != nil {
		return Config{}, fmt.Errorf("parse config %q: %w", path, err)
	}

	overridden, err := applyEnvironmentOverrides(&cfg)
	if err != nil {
		return Config{}, err
	}
	if overridden {
		if err := writeConfig(path, cfg); err != nil {
			return Config{}, fmt.Errorf("persist env config %q: %w", path, err)
		}
	}
	return cfg, nil
}

func applyEnvironmentOverrides(cfg *Config) (bool, error) {
	overridden := false

	if v := os.Getenv("CALDO_SERVER_PORT"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil {
			return false, fmt.Errorf("parse CALDO_SERVER_PORT: %w", err)
		}
		cfg.Server.Port = n
		overridden = true
	}
	if v := os.Getenv("CALDO_SERVER_AUTH_HEADER"); v != "" {
		cfg.Server.AuthHeader = v
		overridden = true
	}
	if v := os.Getenv("CALDO_CALDAV_SERVER_URL"); v != "" {
		cfg.CalDAV.ServerURL = v
		overridden = true
	}
	if v := os.Getenv("CALDO_CALDAV_DEFAULT_LIST"); v != "" {
		cfg.CalDAV.DefaultList = v
		overridden = true
	}
	if v := os.Getenv("CALDO_SECURITY_ENCRYPTION_KEY_FILE"); v != "" {
		cfg.Security.EncryptionKeyFile = v
		overridden = true
	}
	if v := os.Getenv("CALDO_DATABASE_PATH"); v != "" {
		cfg.Database.Path = v
		overridden = true
	}
	if v := os.Getenv("CALDO_SYNC_ENABLED"); v != "" {
		b, err := strconv.ParseBool(v)
		if err != nil {
			return false, fmt.Errorf("parse CALDO_SYNC_ENABLED: %w", err)
		}
		cfg.Sync.Enabled = b
		overridden = true
	}
	if v := os.Getenv("CALDO_SYNC_INTERVAL_SECONDS"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil {
			return false, fmt.Errorf("parse CALDO_SYNC_INTERVAL_SECONDS: %w", err)
		}
		if n > 0 {
			cfg.Sync.IntervalSeconds = n
		}
		overridden = true
	}
	if v := os.Getenv("CALDO_SYNC_DEFAULT_PRINCIPAL"); v != "" {
		cfg.Sync.DefaultPrincipal = v
		overridden = true
	}

	return overridden, nil
}

func writeConfig(path string, cfg Config) error {
	content := fmt.Sprintf(`server:
  port: %d
  auth_header: %q

caldav:
  server_url: %q
  default_list: %q

security:
  encryption_key_file: %q

database:
  path: %q

sync:
  enabled: %t
  interval_seconds: %d
  default_principal: %q
`, cfg.Server.Port, cfg.Server.AuthHeader, cfg.CalDAV.ServerURL, cfg.CalDAV.DefaultList,
		cfg.Security.EncryptionKeyFile, cfg.Database.Path,
		cfg.Sync.Enabled, cfg.Sync.IntervalSeconds, cfg.Sync.DefaultPrincipal)

	return os.WriteFile(path, []byte(content), 0o600)
}
