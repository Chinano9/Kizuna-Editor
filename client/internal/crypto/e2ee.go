/*
Kizuna Editor - A local-first songwriting environment.
Copyright (C) 2025 Fernando Ponce Solis (@Chinano9)

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program.  If not, see <https://www.gnu.org/licenses/>.
*/

package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"kizuna/client/internal/database"
	"os"
	"strings"
)

const (
	// KeySize is 32 bytes (256 bits) for AES-256-GCM.
	KeySize = 32
	// AlgorithmAES256GCM identifies the E2EE authenticated encryption scheme.
	AlgorithmAES256GCM = "AES-256-GCM"
	// DefaultKeyFileName is the local file name where the client's master E2EE key is stored.
	DefaultKeyFileName = "e2ee_master.key"
)

var (
	ErrInvalidKeySize    = errors.New("invalid encryption key size: must be 32 bytes (256 bits)")
	ErrCiphertextTooShort = errors.New("ciphertext too short to contain valid nonce and tag")
)

// GenerateKey generates a cryptographically secure 256-bit (32-byte) key on the client.
func GenerateKey() ([]byte, error) {
	key := make([]byte, KeySize)
	if _, err := io.ReadFull(rand.Reader, key); err != nil {
		return nil, fmt.Errorf("failed to generate random cryptographic key: %w", err)
	}
	return key, nil
}

// ResolveDefaultKeyPath returns the OS-specific local path for the user's E2EE key.
func ResolveDefaultKeyPath() (string, error) {
	return database.ResolveDBPath(database.DefaultAppFolder, DefaultKeyFileName)
}

// LoadOrGenerateKey loads an existing local 256-bit key from disk or generates and saves a new one.
// If the target directory is not writable (e.g. in an isolated test environment), it returns the generated
// in-memory key so client encryption remains functional.
func LoadOrGenerateKey(keyPath string) ([]byte, error) {
	if strings.TrimSpace(keyPath) == "" {
		resolved, err := ResolveDefaultKeyPath()
		if err == nil {
			keyPath = resolved
		}
	}

	if keyPath != "" {
		if raw, err := os.ReadFile(keyPath); err == nil {
			trimmed := strings.TrimSpace(string(raw))
			if decoded, err := hex.DecodeString(trimmed); err == nil && len(decoded) == KeySize {
				return decoded, nil
			}
			if decoded, err := base64.StdEncoding.DecodeString(trimmed); err == nil && len(decoded) == KeySize {
				return decoded, nil
			}
			if len(raw) == KeySize {
				key := make([]byte, KeySize)
				copy(key, raw)
				return key, nil
			}
		}
	}

	key, err := GenerateKey()
	if err != nil {
		return nil, err
	}

	if keyPath != "" {
		if err := database.EnsureDirectory(keyPath); err == nil {
			_ = os.WriteFile(keyPath, []byte(hex.EncodeToString(key)), 0o600)
		}
	}

	return key, nil
}

// Encrypt encrypts plaintext bytes using AES-256-GCM with a random 12-byte nonce
// and returns a Base64-encoded string containing [nonce || ciphertext || tag].
func Encrypt(plaintext []byte, key []byte) (string, error) {
	if len(key) != KeySize {
		return "", ErrInvalidKeySize
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", fmt.Errorf("failed to initialize AES cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("failed to initialize GCM mode: %w", err)
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("failed to generate GCM nonce: %w", err)
	}

	sealed := gcm.Seal(nonce, nonce, plaintext, nil)
	return base64.StdEncoding.EncodeToString(sealed), nil
}

// Decrypt decodes a Base64-encoded [nonce || ciphertext || tag] payload and
// authenticates/decrypts it using AES-256-GCM with the provided 256-bit local key.
func Decrypt(encodedCiphertext string, key []byte) ([]byte, error) {
	if len(key) != KeySize {
		return nil, ErrInvalidKeySize
	}

	raw, err := base64.StdEncoding.DecodeString(strings.TrimSpace(encodedCiphertext))
	if err != nil {
		return nil, fmt.Errorf("invalid base64 encrypted payload: %w", err)
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize AES cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize GCM mode: %w", err)
	}

	nonceSize := gcm.NonceSize()
	if len(raw) < nonceSize+gcm.Overhead() {
		return nil, ErrCiphertextTooShort
	}

	nonce := raw[:nonceSize]
	ciphertext := raw[nonceSize:]

	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt payload with local key (authentication failed): %w", err)
	}

	return plaintext, nil
}

// KeyFingerprint returns a short SHA-256 hex digest of the key for diagnostics without exposing key material.
func KeyFingerprint(key []byte) string {
	sum := sha256.Sum256(key)
	return hex.EncodeToString(sum[:8])
}
