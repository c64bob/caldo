package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
)

const credentialFormatVersion = "v1"

var (
	// ErrInvalidCredentialCiphertext indicates that encrypted credential payload is malformed.
	ErrInvalidCredentialCiphertext = errors.New("invalid credential ciphertext")
)

// EncryptCredential encrypts a secret with AES-256-GCM and returns a versioned payload.
func EncryptCredential(key []byte, plaintext string) (string, error) {
	if len(key) != 32 {
		return "", fmt.Errorf("invalid key length: got %d want 32", len(key))
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", fmt.Errorf("create aes cipher: %w", err)
	}

	aead, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("create gcm: %w", err)
	}

	nonce := make([]byte, aead.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return "", fmt.Errorf("generate nonce: %w", err)
	}

	ciphertext := aead.Seal(nil, nonce, []byte(plaintext), nil)

	return strings.Join([]string{
		credentialFormatVersion,
		base64.StdEncoding.EncodeToString(nonce),
		base64.StdEncoding.EncodeToString(ciphertext),
	}, ":"), nil
}

// DecryptCredential decrypts a versioned AES-256-GCM payload.
func DecryptCredential(key []byte, payload string) (string, error) {
	if len(key) != 32 {
		return "", fmt.Errorf("invalid key length: got %d want 32", len(key))
	}

	parts := strings.Split(payload, ":")
	if len(parts) != 3 {
		return "", ErrInvalidCredentialCiphertext
	}
	if parts[0] != credentialFormatVersion {
		return "", ErrInvalidCredentialCiphertext
	}

	nonce, err := base64.StdEncoding.DecodeString(parts[1])
	if err != nil {
		return "", ErrInvalidCredentialCiphertext
	}

	ciphertext, err := base64.StdEncoding.DecodeString(parts[2])
	if err != nil {
		return "", ErrInvalidCredentialCiphertext
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", fmt.Errorf("create aes cipher: %w", err)
	}

	aead, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("create gcm: %w", err)
	}

	plaintext, err := aead.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", fmt.Errorf("decrypt payload: %w", err)
	}

	return string(plaintext), nil
}
