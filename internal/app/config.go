package app

import (
	"bufio"
	"fmt"
	"os"
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
	return cfg, nil
}
