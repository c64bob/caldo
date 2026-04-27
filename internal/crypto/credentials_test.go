package crypto

import (
	"bytes"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
	"testing"
)

func TestEncryptDecryptCredentialRoundtrip(t *testing.T) {
	t.Parallel()

	key := bytes.Repeat([]byte{0x2A}, 32)
	plaintext := "top-secret"

	payload, err := EncryptCredential(key, plaintext)
	if err != nil {
		t.Fatalf("encrypt credential: %v", err)
	}

	parts := strings.Split(payload, ":")
	if len(parts) != 3 {
		t.Fatalf("payload format mismatch: got %q", payload)
	}
	if parts[0] != "v1" {
		t.Fatalf("unexpected version: got %q want %q", parts[0], "v1")
	}

	decrypted, err := DecryptCredential(key, payload)
	if err != nil {
		t.Fatalf("decrypt credential: %v", err)
	}
	if decrypted != plaintext {
		t.Fatalf("unexpected plaintext: got %q want %q", decrypted, plaintext)
	}
}

func TestDecryptCredentialRejectsMalformedPayload(t *testing.T) {
	t.Parallel()

	key := bytes.Repeat([]byte{0x2A}, 32)

	_, err := DecryptCredential(key, "invalid")
	if !errors.Is(err, ErrInvalidCredentialCiphertext) {
		t.Fatalf("expected malformed payload error, got %v", err)
	}
}

func TestDecryptCredentialRejectsInvalidNonceSize(t *testing.T) {
	t.Parallel()

	key := bytes.Repeat([]byte{0x2A}, 32)
	payload := fmt.Sprintf(
		"v1:%s:%s",
		base64.StdEncoding.EncodeToString([]byte{0x01}),
		base64.StdEncoding.EncodeToString([]byte("ciphertext")),
	)

	_, err := DecryptCredential(key, payload)
	if !errors.Is(err, ErrInvalidCredentialCiphertext) {
		t.Fatalf("expected malformed payload error, got %v", err)
	}
}

func TestDecryptCredentialWithWrongKeyFails(t *testing.T) {
	t.Parallel()

	goodKey := bytes.Repeat([]byte{0x11}, 32)
	wrongKey := bytes.Repeat([]byte{0x22}, 32)

	payload, err := EncryptCredential(goodKey, "top-secret")
	if err != nil {
		t.Fatalf("encrypt credential: %v", err)
	}

	_, err = DecryptCredential(wrongKey, payload)
	if err == nil {
		t.Fatal("expected wrong key to fail decryption")
	}
	if errors.Is(err, ErrInvalidCredentialCiphertext) {
		t.Fatalf("expected auth failure, got malformed payload error: %v", err)
	}
}
