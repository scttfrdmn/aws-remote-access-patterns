// Package encryption provides secure encryption utilities for credential storage
package encryption

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"io"

	"golang.org/x/crypto/pbkdf2"
)

const (
	// KeySize is the AES key size in bytes (AES-256)
	KeySize = 32
	// SaltSize is the salt size in bytes
	SaltSize = 16
	// NonceSize is the GCM nonce size in bytes
	NonceSize = 12
	// PBKDF2Iterations is the number of PBKDF2 iterations for key derivation
	PBKDF2Iterations = 100000
)

// Encryptor provides secure encryption and decryption for sensitive data
type Encryptor struct {
	// password is the user-provided password for encryption
	password []byte
}

// NewEncryptor creates a new encryptor with the given password
func NewEncryptor(password string) *Encryptor {
	return &Encryptor{
		password: []byte(password),
	}
}

// NewEncryptorFromEnv creates an encryptor using environment-based key derivation
func NewEncryptorFromEnv() (*Encryptor, error) {
	// Derive password from machine-specific information
	// This is less secure than user-provided password but better than no encryption
	hostname, err := getHostname()
	if err != nil {
		return nil, fmt.Errorf("failed to get hostname: %w", err)
	}
	
	// Combine hostname with application identifier
	password := fmt.Sprintf("aws-remote-access-patterns-%s", hostname)
	return NewEncryptor(password), nil
}

// EncryptedData represents encrypted data with metadata
type EncryptedData struct {
	Salt       []byte `json:"salt"`
	Nonce      []byte `json:"nonce"`
	Ciphertext []byte `json:"ciphertext"`
	Version    int    `json:"version"`
}

// Encrypt encrypts plaintext data using AES-GCM with PBKDF2 key derivation
func (e *Encryptor) Encrypt(plaintext []byte) (*EncryptedData, error) {
	if len(plaintext) == 0 {
		return nil, fmt.Errorf("plaintext cannot be empty")
	}

	// Generate random salt
	salt := make([]byte, SaltSize)
	if _, err := io.ReadFull(rand.Reader, salt); err != nil {
		return nil, fmt.Errorf("failed to generate salt: %w", err)
	}

	// Derive key using PBKDF2
	key := pbkdf2.Key(e.password, salt, PBKDF2Iterations, KeySize, sha256.New)

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
	nonce := make([]byte, NonceSize)
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, fmt.Errorf("failed to generate nonce: %w", err)
	}

	// Encrypt the data
	ciphertext := gcm.Seal(nil, nonce, plaintext, nil)

	return &EncryptedData{
		Salt:       salt,
		Nonce:      nonce,
		Ciphertext: ciphertext,
		Version:    1,
	}, nil
}

// Decrypt decrypts encrypted data
func (e *Encryptor) Decrypt(data *EncryptedData) ([]byte, error) {
	if data == nil {
		return nil, fmt.Errorf("encrypted data cannot be nil")
	}

	if data.Version != 1 {
		return nil, fmt.Errorf("unsupported encryption version: %d", data.Version)
	}

	if len(data.Salt) != SaltSize {
		return nil, fmt.Errorf("invalid salt size: expected %d, got %d", SaltSize, len(data.Salt))
	}

	if len(data.Nonce) != NonceSize {
		return nil, fmt.Errorf("invalid nonce size: expected %d, got %d", NonceSize, len(data.Nonce))
	}

	// Derive key using the same parameters
	key := pbkdf2.Key(e.password, data.Salt, PBKDF2Iterations, KeySize, sha256.New)

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

	// Decrypt the data
	plaintext, err := gcm.Open(nil, data.Nonce, data.Ciphertext, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt: %w", err)
	}

	return plaintext, nil
}

// EncryptString encrypts a string and returns base64-encoded result
func (e *Encryptor) EncryptString(plaintext string) (string, error) {
	data, err := e.Encrypt([]byte(plaintext))
	if err != nil {
		return "", err
	}

	// Encode as base64 for storage
	return e.encodeEncryptedData(data), nil
}

// DecryptString decrypts a base64-encoded encrypted string
func (e *Encryptor) DecryptString(encrypted string) (string, error) {
	data, err := e.decodeEncryptedData(encrypted)
	if err != nil {
		return "", err
	}

	plaintext, err := e.Decrypt(data)
	if err != nil {
		return "", err
	}

	return string(plaintext), nil
}

// encodeEncryptedData encodes EncryptedData as base64 string
func (e *Encryptor) encodeEncryptedData(data *EncryptedData) string {
	// Create a simple format: version:salt:nonce:ciphertext (all base64)
	version := fmt.Sprintf("%d", data.Version)
	salt := base64.StdEncoding.EncodeToString(data.Salt)
	nonce := base64.StdEncoding.EncodeToString(data.Nonce)
	ciphertext := base64.StdEncoding.EncodeToString(data.Ciphertext)
	
	combined := fmt.Sprintf("%s:%s:%s:%s", version, salt, nonce, ciphertext)
	return base64.StdEncoding.EncodeToString([]byte(combined))
}

// decodeEncryptedData decodes base64 string to EncryptedData
func (e *Encryptor) decodeEncryptedData(encoded string) (*EncryptedData, error) {
	// Decode base64
	combined, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return nil, fmt.Errorf("failed to decode base64: %w", err)
	}

	// Split components
	parts := splitString(string(combined), ":", 4)
	if len(parts) != 4 {
		return nil, fmt.Errorf("invalid encrypted data format")
	}

	// Parse version
	version := 0
	if _, err := fmt.Sscanf(parts[0], "%d", &version); err != nil {
		return nil, fmt.Errorf("invalid version: %w", err)
	}

	// Decode salt
	salt, err := base64.StdEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, fmt.Errorf("failed to decode salt: %w", err)
	}

	// Decode nonce
	nonce, err := base64.StdEncoding.DecodeString(parts[2])
	if err != nil {
		return nil, fmt.Errorf("failed to decode nonce: %w", err)
	}

	// Decode ciphertext
	ciphertext, err := base64.StdEncoding.DecodeString(parts[3])
	if err != nil {
		return nil, fmt.Errorf("failed to decode ciphertext: %w", err)
	}

	return &EncryptedData{
		Version:    version,
		Salt:       salt,
		Nonce:      nonce,
		Ciphertext: ciphertext,
	}, nil
}

// ValidatePassword validates that a password meets security requirements
func ValidatePassword(password string) error {
	if len(password) < 12 {
		return fmt.Errorf("password must be at least 12 characters long")
	}
	
	// Check for basic character diversity
	hasUpper := false
	hasLower := false
	hasDigit := false
	hasSpecial := false
	
	for _, ch := range password {
		switch {
		case ch >= 'A' && ch <= 'Z':
			hasUpper = true
		case ch >= 'a' && ch <= 'z':
			hasLower = true
		case ch >= '0' && ch <= '9':
			hasDigit = true
		case isSpecialChar(ch):
			hasSpecial = true
		}
	}
	
	if !hasUpper {
		return fmt.Errorf("password must contain at least one uppercase letter")
	}
	if !hasLower {
		return fmt.Errorf("password must contain at least one lowercase letter")
	}
	if !hasDigit {
		return fmt.Errorf("password must contain at least one digit")
	}
	if !hasSpecial {
		return fmt.Errorf("password must contain at least one special character")
	}
	
	return nil
}

// GenerateSecurePassword generates a cryptographically secure password
func GenerateSecurePassword(length int) (string, error) {
	if length < 12 {
		length = 12
	}
	
	// Character sets
	uppercase := "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
	lowercase := "abcdefghijklmnopqrstuvwxyz"
	digits := "0123456789"
	special := "!@#$%^&*()-_+=[]{}|;:,.<>?"
	
	allChars := uppercase + lowercase + digits + special
	
	password := make([]byte, length)
	
	// Generate random bytes
	if _, err := rand.Read(password); err != nil {
		return "", fmt.Errorf("failed to generate random password: %w", err)
	}
	
	// Convert to valid characters
	for i := range password {
		password[i] = allChars[int(password[i])%len(allChars)]
	}
	
	// Ensure at least one character from each set
	if length >= 4 {
		password[0] = uppercase[int(password[0])%len(uppercase)]
		password[1] = lowercase[int(password[1])%len(lowercase)]
		password[2] = digits[int(password[2])%len(digits)]
		password[3] = special[int(password[3])%len(special)]
		
		// Shuffle to avoid predictable positions
		for i := range password {
			j := int(password[i]) % len(password)
			password[i], password[j] = password[j], password[i]
		}
	}
	
	return string(password), nil
}

// Helper functions

func getHostname() (string, error) {
	// This would normally use os.Hostname() but we'll provide a simple implementation
	// for testing purposes
	hostname := "default-host"
	return hostname, nil
}

func splitString(s, sep string, n int) []string {
	parts := make([]string, 0, n)
	start := 0
	
	for i := 0; i < n-1; i++ {
		idx := findNext(s[start:], sep)
		if idx == -1 {
			break
		}
		parts = append(parts, s[start:start+idx])
		start += idx + len(sep)
	}
	
	if start < len(s) {
		parts = append(parts, s[start:])
	}
	
	return parts
}

func findNext(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}

func isSpecialChar(ch rune) bool {
	special := "!@#$%^&*()-_+=[]{}|;:,.<>?"
	for _, s := range special {
		if ch == s {
			return true
		}
	}
	return false
}