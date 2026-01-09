package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"time"

	"golang.org/x/crypto/hkdf"
)

const (
	// KeyLength is the length of derived encryption keys (32 bytes = 256 bits)
	KeyLength = 32
	// NonceLength is the length of nonces for AES-GCM (12 bytes)
	NonceLength = 12
)

// EncryptedPayload represents the encrypted data format expected by Nexus API
type EncryptedPayload struct {
	Encrypted     bool   `json:"encrypted"`
	KeyDate       string `json:"keyDate"`
	SecretVersion int    `json:"secretVersion"`
	Nonce         string `json:"nonce"`
	Data          string `json:"data"`
}

// EncryptPayload encrypts the data using AES-256-GCM with daily key derivation
// This matches the encryption format used by the Nexus Python SDK
func EncryptPayload(data map[string]interface{}, masterSecretB64 string, appKey string) (*EncryptedPayload, error) {
	// Decode master secret from base64
	masterSecret, err := base64.StdEncoding.DecodeString(masterSecretB64)
	if err != nil {
		return nil, fmt.Errorf("failed to decode master secret: %w", err)
	}

	// Debug: Log the first 8 chars of master secret (matching server debug format)
	log.Printf("DEBUG EncryptPayload: appKey=%s, masterSecretB64(first8)=%s..., decodedLen=%d",
		appKey, masterSecretB64[:min(8, len(masterSecretB64))], len(masterSecret))

	// Get today's date in UTC
	keyDate := time.Now().UTC().Format("2006-01-02")
	log.Printf("DEBUG EncryptPayload: keyDate=%s", keyDate)

	// Derive daily key using HKDF (must match Python SDK)
	key, err := deriveKeyForDate(masterSecret, appKey, keyDate)
	if err != nil {
		return nil, fmt.Errorf("failed to derive key: %w", err)
	}

	// Create AES cipher
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher: %w", err)
	}

	// Create GCM mode
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}

	// Generate random nonce
	nonce := make([]byte, NonceLength)
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, fmt.Errorf("failed to generate nonce: %w", err)
	}

	// Serialize data to JSON
	plaintext, err := json.Marshal(data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal data: %w", err)
	}

	// Encrypt the data
	ciphertext := gcm.Seal(nil, nonce, plaintext, nil)

	// Return the encrypted payload in the expected format
	return &EncryptedPayload{
		Encrypted:     true,
		KeyDate:       keyDate,
		SecretVersion: 1,
		Nonce:         base64.StdEncoding.EncodeToString(nonce),
		Data:          base64.StdEncoding.EncodeToString(ciphertext),
	}, nil
}

// deriveKeyForDate uses HKDF to derive a daily encryption key
// MUST match the Python SDK: salt=None, info="nexus-enigma-{appKey}-{date}"
func deriveKeyForDate(masterSecret []byte, appKey string, date string) ([]byte, error) {
	info := fmt.Sprintf("nexus-enigma-%s-%s", appKey, date)

	// Use HKDF with SHA-256, no salt
	hkdfReader := hkdf.New(sha256.New, masterSecret, nil, []byte(info))

	key := make([]byte, KeyLength)
	if _, err := io.ReadFull(hkdfReader, key); err != nil {
		return nil, fmt.Errorf("failed to derive key: %w", err)
	}

	return key, nil
}

// min returns the smaller of a or b
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
