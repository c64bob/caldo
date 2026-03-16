package sqlite

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

type DAVAccount struct {
	PrincipalID       string `json:"principal_id"`
	ServerURL         string `json:"server_url"`
	Username          string `json:"username"`
	PasswordEncrypted []byte `json:"password_encrypted"`
}

type DAVAccountsRepo struct {
	filePath string
	mu       sync.Mutex
}

func NewDAVAccountsRepo(db *DB) *DAVAccountsRepo {
	return &DAVAccountsRepo{filePath: db.Path + ".dav_accounts.json"}
}

func (r *DAVAccountsRepo) Upsert(_ context.Context, account DAVAccount) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	accounts := map[string]DAVAccount{}
	if data, err := os.ReadFile(r.filePath); err == nil && len(data) > 0 {
		if err := json.Unmarshal(data, &accounts); err != nil {
			return fmt.Errorf("decode account store: %w", err)
		}
	}
	accounts[account.PrincipalID] = account

	encoded, err := json.MarshalIndent(accounts, "", "  ")
	if err != nil {
		return fmt.Errorf("encode account store: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(r.filePath), 0o755); err != nil {
		return fmt.Errorf("create account store dir: %w", err)
	}
	if err := os.WriteFile(r.filePath, encoded, 0o600); err != nil {
		return fmt.Errorf("write account store: %w", err)
	}
	return nil
}
