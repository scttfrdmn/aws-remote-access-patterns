// Package cache provides secure credential caching for the AWS CLI helper
package cache

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"golang.org/x/crypto/pbkdf2"
)

// Credentials represents cached AWS credentials
type Credentials struct {
	AccessKeyID     string    `json:"access_key_id"`
	SecretAccessKey string    `json:"secret_access_key"`
	SessionToken    string    `json:"session_token,omitempty"`
	ExpiresAt       time.Time `json:"expires_at"`
	Region          string    `json:"region,omitempty"`
	CachedAt        time.Time `json:"cached_at"`
}

// Cache provides secure credential caching
type Cache struct {
	directory string
	maxAge    time.Duration
	key       []byte
}

// New creates a new cache instance
func New(directory string, maxAge time.Duration) (*Cache, error) {
	// Create cache directory if it doesn't exist
	if err := os.MkdirAll(directory, 0700); err != nil {
		return nil, fmt.Errorf("failed to create cache directory: %w", err)
	}

	// Generate or load encryption key
	key, err := getOrCreateKey(directory)
	if err != nil {
		return nil, fmt.Errorf("failed to get encryption key: %w", err)
	}

	return &Cache{
		directory: directory,
		maxAge:    maxAge,
		key:       key,
	}, nil
}

// Set stores credentials in the cache
func (c *Cache) Set(profile string, creds *Credentials) error {
	// Set cached timestamp
	creds.CachedAt = time.Now()

	// Marshal to JSON
	data, err := json.Marshal(creds)
	if err != nil {
		return fmt.Errorf("failed to marshal credentials: %w", err)
	}

	// Encrypt the data
	encrypted, err := c.encrypt(data)
	if err != nil {
		return fmt.Errorf("failed to encrypt credentials: %w", err)
	}

	// Write to cache file
	cacheFile := c.getCacheFile(profile)
	if err := os.WriteFile(cacheFile, encrypted, 0600); err != nil {
		return fmt.Errorf("failed to write cache file: %w", err)
	}

	return nil
}

// Get retrieves credentials from the cache
func (c *Cache) Get(profile string) *Credentials {
	cacheFile := c.getCacheFile(profile)

	// Check if cache file exists
	if _, err := os.Stat(cacheFile); os.IsNotExist(err) {
		return nil
	}

	// Read cache file
	encrypted, err := os.ReadFile(cacheFile)
	if err != nil {
		return nil
	}

	// Decrypt the data
	data, err := c.decrypt(encrypted)
	if err != nil {
		// If decryption fails, remove the corrupted cache file
		os.Remove(cacheFile)
		return nil
	}

	// Unmarshal credentials
	var creds Credentials
	if err := json.Unmarshal(data, &creds); err != nil {
		// If unmarshaling fails, remove the corrupted cache file
		os.Remove(cacheFile)
		return nil
	}

	// Check if cache is too old
	if time.Since(creds.CachedAt) > c.maxAge {
		c.Delete(profile)
		return nil
	}

	return &creds
}

// Delete removes credentials from the cache
func (c *Cache) Delete(profile string) error {
	cacheFile := c.getCacheFile(profile)
	if err := os.Remove(cacheFile); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to delete cache file: %w", err)
	}
	return nil
}

// Clear removes all cached credentials
func (c *Cache) Clear() error {
	entries, err := os.ReadDir(c.directory)
	if err != nil {
		return fmt.Errorf("failed to read cache directory: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() || entry.Name() == "key" {
			continue
		}
		
		filePath := filepath.Join(c.directory, entry.Name())
		if err := os.Remove(filePath); err != nil {
			return fmt.Errorf("failed to remove cache file %s: %w", entry.Name(), err)
		}
	}

	return nil
}

// List returns all cached profile names
func (c *Cache) List() ([]string, error) {
	entries, err := os.ReadDir(c.directory)
	if err != nil {
		return nil, fmt.Errorf("failed to read cache directory: %w", err)
	}

	var profiles []string
	for _, entry := range entries {
		if entry.IsDir() || entry.Name() == "key" {
			continue
		}
		
		// Remove .enc extension
		name := entry.Name()
		if filepath.Ext(name) == ".enc" {
			name = name[:len(name)-4]
		}
		profiles = append(profiles, name)
	}

	return profiles, nil
}

// IsExpired checks if credentials are expired
func (c *Credentials) IsExpired() bool {
	return time.Now().After(c.ExpiresAt)
}

// TimeUntilExpiry returns the time until credentials expire
func (c *Credentials) TimeUntilExpiry() time.Duration {
	return time.Until(c.ExpiresAt)
}

// getCacheFile returns the cache file path for a profile
func (c *Cache) getCacheFile(profile string) string {
	return filepath.Join(c.directory, profile+".enc")
}

// encrypt encrypts data using AES-GCM
func (c *Cache) encrypt(data []byte) ([]byte, error) {
	block, err := aes.NewCipher(c.key)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}

	ciphertext := gcm.Seal(nonce, nonce, data, nil)
	return ciphertext, nil
}

// decrypt decrypts data using AES-GCM
func (c *Cache) decrypt(data []byte) ([]byte, error) {
	block, err := aes.NewCipher(c.key)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonceSize := gcm.NonceSize()
	if len(data) < nonceSize {
		return nil, fmt.Errorf("ciphertext too short")
	}

	nonce, ciphertext := data[:nonceSize], data[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, err
	}

	return plaintext, nil
}

// getOrCreateKey generates or loads an encryption key
func getOrCreateKey(directory string) ([]byte, error) {
	keyFile := filepath.Join(directory, "key")

	// Try to load existing key
	if keyData, err := os.ReadFile(keyFile); err == nil {
		return keyData, nil
	}

	// Generate new key
	password := make([]byte, 32)
	if _, err := rand.Read(password); err != nil {
		return nil, fmt.Errorf("failed to generate random password: %w", err)
	}

	salt := make([]byte, 16)
	if _, err := rand.Read(salt); err != nil {
		return nil, fmt.Errorf("failed to generate salt: %w", err)
	}

	key := pbkdf2.Key(password, salt, 100000, 32, sha256.New)

	// Store the key (in a real implementation, you might want to use the system keyring)
	keyData := append(salt, key...)
	if err := os.WriteFile(keyFile, keyData, 0600); err != nil {
		return nil, fmt.Errorf("failed to write key file: %w", err)
	}

	return key, nil
}

// CleanupExpired removes expired cache entries
func (c *Cache) CleanupExpired() error {
	profiles, err := c.List()
	if err != nil {
		return err
	}

	for _, profile := range profiles {
		creds := c.Get(profile)
		if creds == nil || creds.IsExpired() {
			c.Delete(profile)
		}
	}

	return nil
}

// GetStats returns cache statistics
func (c *Cache) GetStats() (map[string]interface{}, error) {
	profiles, err := c.List()
	if err != nil {
		return nil, err
	}

	stats := map[string]interface{}{
		"total_profiles": len(profiles),
		"valid_cached":   0,
		"expired_cached": 0,
		"cache_size":     0,
	}

	for _, profile := range profiles {
		creds := c.Get(profile)
		if creds == nil {
			continue
		}

		if creds.IsExpired() {
			stats["expired_cached"] = stats["expired_cached"].(int) + 1
		} else {
			stats["valid_cached"] = stats["valid_cached"].(int) + 1
		}
	}

	// Calculate cache directory size
	if entries, err := os.ReadDir(c.directory); err == nil {
		var totalSize int64
		for _, entry := range entries {
			if !entry.IsDir() {
				if info, err := entry.Info(); err == nil {
					totalSize += info.Size()
				}
			}
		}
		stats["cache_size"] = totalSize
	}

	return stats, nil
}