package aiconfig

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"errors"
	"fmt"
	"io"
)

// minDataLen is the minimum valid length for encrypted data:
// 1 byte (key version) + 12 bytes (GCM nonce) + 16 bytes (GCM tag, empty plaintext) = 29.
const minDataLen = 1 + 12 + 16

// Encrypt encrypts plaintext using AES-256-GCM and returns
// [keyVersion(1) || nonce(12) || ciphertext].
// key must be exactly 32 bytes.
func Encrypt(plaintext string, key []byte, keyVersion byte) ([]byte, error) {
	if len(key) != 32 {
		return nil, fmt.Errorf("aiconfig: encryption key must be 32 bytes, got %d", len(key))
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("aiconfig: create AES cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("aiconfig: create GCM: %w", err)
	}

	nonce := make([]byte, gcm.NonceSize()) // 12 bytes
	if _, err = io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, fmt.Errorf("aiconfig: generate nonce: %w", err)
	}

	ciphertext := gcm.Seal(nil, nonce, []byte(plaintext), nil)

	// Layout: keyVersion || nonce || ciphertext
	result := make([]byte, 1+len(nonce)+len(ciphertext))
	result[0] = keyVersion
	copy(result[1:], nonce)
	copy(result[1+len(nonce):], ciphertext)

	return result, nil
}

// Decrypt decrypts data produced by Encrypt.
// data[0] is keyVersion, data[1:13] is nonce, data[13:] is ciphertext.
func Decrypt(data []byte, key []byte) (string, error) {
	if len(data) < minDataLen {
		return "", errors.New("aiconfig: encrypted data too short")
	}

	// data[0] is keyVersion — reserved for future key rotation.
	nonce := data[1:13]
	ciphertext := data[13:]

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", fmt.Errorf("aiconfig: create AES cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("aiconfig: create GCM: %w", err)
	}

	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", fmt.Errorf("aiconfig: decrypt: %w", err)
	}

	return string(plaintext), nil
}

// MaskAPIKey shows the first 3 chars and last 4 chars of an API key.
// If len(key) <= 8, returns "****".
func MaskAPIKey(key string) string {
	if len(key) <= 8 {
		return "****"
	}
	return key[:3] + "..." + key[len(key)-4:]
}
