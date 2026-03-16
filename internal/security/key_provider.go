package security

import (
	"encoding/base64"
	"errors"
	"fmt"
	"os"
	"strings"
)

const keySize = 32

func LoadMasterKey(keyFile string) ([]byte, error) {
	if env := strings.TrimSpace(os.Getenv("CALDO_MASTER_KEY")); env != "" {
		key, err := decodeKey(env)
		if err != nil {
			return nil, fmt.Errorf("decode CALDO_MASTER_KEY: %w", err)
		}
		return key, nil
	}

	if strings.TrimSpace(keyFile) == "" {
		return nil, errors.New("missing master key: set CALDO_MASTER_KEY or security.encryption_key_file")
	}

	data, err := os.ReadFile(keyFile)
	if err != nil {
		return nil, fmt.Errorf("read encryption key file: %w", err)
	}

	key, err := decodeKey(strings.TrimSpace(string(data)))
	if err != nil {
		return nil, fmt.Errorf("decode encryption key file: %w", err)
	}
	return key, nil
}

func decodeKey(raw string) ([]byte, error) {
	k, err := base64.StdEncoding.DecodeString(raw)
	if err != nil {
		return nil, err
	}
	if len(k) != keySize {
		return nil, fmt.Errorf("invalid key length %d, expected %d", len(k), keySize)
	}
	return k, nil
}
