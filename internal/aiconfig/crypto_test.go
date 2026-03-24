package aiconfig

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var testKey = []byte("12345678901234567890123456789012") // 32 bytes

// ---------------------------------------------------------------------------
// Encrypt / Decrypt
// ---------------------------------------------------------------------------

func TestEncryptDecrypt(t *testing.T) {
	plaintext := "sk-test-api-key-1234567890abcdef"

	ciphertext, err := Encrypt(plaintext, testKey, 1)
	require.NoError(t, err)
	assert.NotEmpty(t, ciphertext)

	decrypted, err := Decrypt(ciphertext, testKey)
	require.NoError(t, err)
	assert.Equal(t, plaintext, decrypted)
}

func TestEncryptDecrypt_EmptyPlaintext(t *testing.T) {
	ciphertext, err := Encrypt("", testKey, 1)
	require.NoError(t, err)

	decrypted, err := Decrypt(ciphertext, testKey)
	require.NoError(t, err)
	assert.Equal(t, "", decrypted)
}

func TestEncryptDecrypt_PreservesKeyVersion(t *testing.T) {
	const version byte = 3

	ciphertext, err := Encrypt("secret", testKey, version)
	require.NoError(t, err)
	require.NotEmpty(t, ciphertext)

	// First byte should be the key version.
	assert.Equal(t, version, ciphertext[0])
}

func TestEncryptDifferentNonces(t *testing.T) {
	plaintext := "same-plaintext"

	ct1, err := Encrypt(plaintext, testKey, 1)
	require.NoError(t, err)

	ct2, err := Encrypt(plaintext, testKey, 1)
	require.NoError(t, err)

	// Same plaintext must produce different ciphertexts due to random nonces.
	assert.False(t, bytes.Equal(ct1, ct2), "two encryptions of the same plaintext should differ")
}

func TestDecryptWrongKey(t *testing.T) {
	ciphertext, err := Encrypt("secret", testKey, 1)
	require.NoError(t, err)

	wrongKey := []byte("wrongkey00000000wrongkey00000000") // 32 bytes
	_, err = Decrypt(ciphertext, wrongKey)
	assert.Error(t, err, "decrypting with wrong key should return an error")
}

func TestDecryptCorrupted_TooShort(t *testing.T) {
	// Minimum valid length = 1 (version) + 12 (nonce) + 16 (GCM tag) = 29 bytes.
	short := make([]byte, 10)
	_, err := Decrypt(short, testKey)
	assert.Error(t, err, "too-short data should return an error")
}

func TestDecryptTampered(t *testing.T) {
	ciphertext, err := Encrypt("original text", testKey, 1)
	require.NoError(t, err)

	// Flip a bit near the end of the ciphertext (the GCM tag area).
	tampered := make([]byte, len(ciphertext))
	copy(tampered, ciphertext)
	tampered[len(tampered)-1] ^= 0xFF

	_, err = Decrypt(tampered, testKey)
	assert.Error(t, err, "tampered ciphertext should fail authentication")
}

func TestEncrypt_KeyTooShort(t *testing.T) {
	_, err := Encrypt("secret", []byte("tooshort"), 1)
	assert.Error(t, err, "key shorter than 32 bytes should return an error")
}

func TestEncrypt_KeyTooLong(t *testing.T) {
	longKey := make([]byte, 64)
	_, err := Encrypt("secret", longKey, 1)
	assert.Error(t, err, "key longer than 32 bytes should return an error")
}

// ---------------------------------------------------------------------------
// MaskAPIKey
// ---------------------------------------------------------------------------

func TestMaskAPIKey(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "normal key",
			input:    "sk-1234567890abcdef",
			expected: "sk-...cdef",
		},
		{
			name:     "exactly 9 chars",
			input:    "123456789",
			expected: "123...6789",
		},
		{
			name:     "short key 5 chars",
			input:    "short",
			expected: "****",
		},
		{
			name:     "exactly 8 chars",
			input:    "12345678",
			expected: "****",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "****",
		},
		{
			name:     "long key",
			input:    "sk-proj-abcdefghijklmnopqrstuvwxyz1234567890",
			expected: "sk-...7890",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := MaskAPIKey(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}
