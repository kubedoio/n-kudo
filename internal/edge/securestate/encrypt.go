package securestate

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"os"
)

const (
	// VersionByte for the encrypted file format
	VersionByte byte = 0x01

	// KeySize is the required size for AES-256 (32 bytes)
	KeySize = 32

	// NonceSize is the required nonce size for GCM (12 bytes)
	NonceSize = 12
)

var (
	// ErrInvalidKey is returned when the encryption key is invalid
	ErrInvalidKey = errors.New("invalid encryption key: must be 32 bytes or base64 encoded 32 bytes")

	// ErrInvalidCiphertext is returned when the ciphertext format is invalid
	ErrInvalidCiphertext = errors.New("invalid ciphertext format")

	// ErrVersionMismatch is returned when the file version is not supported
	ErrVersionMismatch = errors.New("unsupported encrypted file version")
)

// DeriveKey attempts to derive a 32-byte key from various sources:
// 1. NKUDO_STATE_KEY environment variable (raw 32 bytes or base64)
// 2. Returns nil if no key is available (fallback to unencrypted)
func DeriveKey() ([]byte, error) {
	// Try environment variable first
	if keyEnv := os.Getenv("NKUDO_STATE_KEY"); keyEnv != "" {
		key, err := parseKey(keyEnv)
		if err != nil {
			return nil, fmt.Errorf("NKUDO_STATE_KEY: %w", err)
		}
		return key, nil
	}

	// No key available - return nil to indicate unencrypted mode
	return nil, nil
}

// parseKey parses a key from string. It accepts:
// - Raw 32-byte string
// - Base64-encoded 32-byte key
func parseKey(keyStr string) ([]byte, error) {
	// First, try as raw 32-byte key
	if len(keyStr) == KeySize {
		return []byte(keyStr), nil
	}

	// Try base64 decoding
	decoded, err := base64.StdEncoding.DecodeString(keyStr)
	if err != nil {
		return nil, ErrInvalidKey
	}

	if len(decoded) != KeySize {
		return nil, fmt.Errorf("%w: decoded key is %d bytes, expected %d", ErrInvalidKey, len(decoded), KeySize)
	}

	return decoded, nil
}

// Encrypt encrypts plaintext using AES-256-GCM.
// Returns: version byte || nonce || ciphertext || tag
func Encrypt(key, plaintext []byte) ([]byte, error) {
	if len(key) != KeySize {
		return nil, ErrInvalidKey
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("create GCM: %w", err)
	}

	nonce := make([]byte, NonceSize)
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, fmt.Errorf("generate nonce: %w", err)
	}

	// Seal appends the tag to the ciphertext
	ciphertext := gcm.Seal(nil, nonce, plaintext, nil)

	// Format: version byte + nonce + ciphertext (includes tag)
	result := make([]byte, 0, 1+NonceSize+len(ciphertext))
	result = append(result, VersionByte)
	result = append(result, nonce...)
	result = append(result, ciphertext...)

	return result, nil
}

// Decrypt decrypts ciphertext using AES-256-GCM.
// Expects format: version byte || nonce || ciphertext || tag
func Decrypt(key, ciphertext []byte) ([]byte, error) {
	if len(key) != KeySize {
		return nil, ErrInvalidKey
	}

	if len(ciphertext) < 1+NonceSize {
		return nil, ErrInvalidCiphertext
	}

	// Check version byte
	if ciphertext[0] != VersionByte {
		return nil, fmt.Errorf("%w: expected 0x%02x, got 0x%02x", ErrVersionMismatch, VersionByte, ciphertext[0])
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("create GCM: %w", err)
	}

	nonce := ciphertext[1 : 1+NonceSize]
	encryptedData := ciphertext[1+NonceSize:]

	plaintext, err := gcm.Open(nil, nonce, encryptedData, nil)
	if err != nil {
		return nil, fmt.Errorf("decrypt: %w", err)
	}

	return plaintext, nil
}

// IsEncrypted checks if the given data appears to be encrypted
// by checking the version byte
func IsEncrypted(data []byte) bool {
	return len(data) >= 1 && data[0] == VersionByte
}

// GenerateKey generates a new random 32-byte encryption key
func GenerateKey() ([]byte, error) {
	key := make([]byte, KeySize)
	if _, err := io.ReadFull(rand.Reader, key); err != nil {
		return nil, fmt.Errorf("generate key: %w", err)
	}
	return key, nil
}

// GenerateKeyBase64 generates a new random key and returns it as base64
func GenerateKeyBase64() (string, error) {
	key, err := GenerateKey()
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(key), nil
}
