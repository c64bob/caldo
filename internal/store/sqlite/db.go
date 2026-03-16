package sqlite

import (
	"fmt"
	"os"
	"path/filepath"
)

type DB struct {
	Path string
}

func Open(path string) (*DB, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, fmt.Errorf("create db directory: %w", err)
	}
	return &DB{Path: path}, nil
}

func (db *DB) Close() error {
	return nil
}
