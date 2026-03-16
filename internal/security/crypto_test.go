package security

import (
	"bytes"
	"testing"
)

func TestEncryptDecryptAESGCM_Roundtrip(t *testing.T) {
	key := []byte("0123456789abcdef0123456789abcdef")
	plain := []byte("super-secret")

	ciphertext, err := EncryptAESGCM(key, plain)
	if err != nil {
		t.Fatalf("encrypt failed: %v", err)
	}

	decrypted, err := DecryptAESGCM(key, ciphertext)
	if err != nil {
		t.Fatalf("decrypt failed: %v", err)
	}
	if !bytes.Equal(plain, decrypted) {
		t.Fatalf("roundtrip mismatch: got %q want %q", decrypted, plain)
	}
}

func TestEncryptAESGCM_UsesRandomNonce(t *testing.T) {
	key := []byte("0123456789abcdef0123456789abcdef")
	plain := []byte("same")

	c1, err := EncryptAESGCM(key, plain)
	if err != nil {
		t.Fatalf("encrypt #1 failed: %v", err)
	}
	c2, err := EncryptAESGCM(key, plain)
	if err != nil {
		t.Fatalf("encrypt #2 failed: %v", err)
	}
	if bytes.Equal(c1, c2) {
		t.Fatal("expected different ciphertexts due to random nonce")
	}
}

func TestDecryptAESGCM_WrongKeyFails(t *testing.T) {
	key := []byte("0123456789abcdef0123456789abcdef")
	wrong := []byte("abcdef0123456789abcdef0123456789")

	ciphertext, err := EncryptAESGCM(key, []byte("value"))
	if err != nil {
		t.Fatalf("encrypt failed: %v", err)
	}

	if _, err := DecryptAESGCM(wrong, ciphertext); err == nil {
		t.Fatal("expected decrypt with wrong key to fail")
	}
}

func TestDecryptAESGCM_TooShortCiphertextFails(t *testing.T) {
	key := []byte("0123456789abcdef0123456789abcdef")
	if _, err := DecryptAESGCM(key, []byte("short")); err == nil {
		t.Fatal("expected short ciphertext to fail")
	}
}
