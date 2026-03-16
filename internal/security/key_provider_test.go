package security

import (
	"encoding/base64"
	"os"
	"path/filepath"
	"testing"
)

func TestLoadMasterKey_FromEnv(t *testing.T) {
	t.Setenv("CALDO_MASTER_KEY", base64.StdEncoding.EncodeToString([]byte("0123456789abcdef0123456789abcdef")))
	key, err := LoadMasterKey("")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(key) != 32 {
		t.Fatalf("expected key len 32, got %d", len(key))
	}
}

func TestLoadMasterKey_FromFile(t *testing.T) {
	t.Setenv("CALDO_MASTER_KEY", "")
	tmp := t.TempDir()
	path := filepath.Join(tmp, "key")
	encoded := base64.StdEncoding.EncodeToString([]byte("0123456789abcdef0123456789abcdef"))
	if err := os.WriteFile(path, []byte(encoded), 0o600); err != nil {
		t.Fatalf("write key file: %v", err)
	}

	key, err := LoadMasterKey(path)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(key) != 32 {
		t.Fatalf("expected key len 32, got %d", len(key))
	}
}

func TestLoadMasterKey_InvalidBase64Fails(t *testing.T) {
	t.Setenv("CALDO_MASTER_KEY", "not-base64")
	if _, err := LoadMasterKey(""); err == nil {
		t.Fatal("expected invalid base64 error")
	}
}

func TestLoadMasterKey_WrongLengthFails(t *testing.T) {
	t.Setenv("CALDO_MASTER_KEY", base64.StdEncoding.EncodeToString([]byte("short")))
	if _, err := LoadMasterKey(""); err == nil {
		t.Fatal("expected wrong key length error")
	}
}
