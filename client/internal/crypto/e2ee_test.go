package crypto

import (
	"bytes"
	"path/filepath"
	"testing"
)

func TestGenerateAndPersistKey(t *testing.T) {
	keyPath := filepath.Join(t.TempDir(), "Kizuna", DefaultKeyFileName)

	key1, err := LoadOrGenerateKey(keyPath)
	if err != nil {
		t.Fatalf("failed to generate key: %v", err)
	}
	if len(key1) != KeySize {
		t.Fatalf("expected %d-byte key, got %d", KeySize, len(key1))
	}

	// Subsequent load from same path must return the exact same key
	key2, err := LoadOrGenerateKey(keyPath)
	if err != nil {
		t.Fatalf("failed to reload existing key: %v", err)
	}
	if !bytes.Equal(key1, key2) {
		t.Fatalf("expected persisted key to match generated key")
	}
}

func TestEncryptAndDecrypt_AES256GCM(t *testing.T) {
	key, err := GenerateKey()
	if err != nil {
		t.Fatalf("failed to generate key: %v", err)
	}

	secretComposition := []byte(`{"title":"Zero Knowledge Song","tracks":[{"name":"Guitarra 1","data_content":":4 0.1 2.2"}]}`)

	ciphertext, err := Encrypt(secretComposition, key)
	if err != nil {
		t.Fatalf("encryption failed: %v", err)
	}
	if ciphertext == "" || bytes.Contains([]byte(ciphertext), []byte("Zero Knowledge Song")) {
		t.Fatalf("ciphertext must not expose plaintext composition")
	}

	decrypted, err := Decrypt(ciphertext, key)
	if err != nil {
		t.Fatalf("decryption failed: %v", err)
	}
	if !bytes.Equal(decrypted, secretComposition) {
		t.Fatalf("decrypted payload mismatch: got %s", string(decrypted))
	}

	// Wrong key must fail authentication
	wrongKey, _ := GenerateKey()
	if _, err := Decrypt(ciphertext, wrongKey); err == nil {
		t.Fatalf("expected decryption with wrong key to fail")
	}
}
