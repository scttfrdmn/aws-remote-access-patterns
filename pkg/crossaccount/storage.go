// Package crossaccount storage provides secure credential storage implementations
package crossaccount

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/scttfrdmn/aws-remote-access-patterns/pkg/encryption"
)

// CredentialStorage defines the interface for storing and retrieving credentials
type CredentialStorage interface {
	// Store saves credentials with the given key
	Store(ctx context.Context, key string, credentials *StoredCredentials) error
	
	// Retrieve gets credentials by key
	Retrieve(ctx context.Context, key string) (*StoredCredentials, error)
	
	// Delete removes credentials by key
	Delete(ctx context.Context, key string) error
	
	// List returns all stored credential keys
	List(ctx context.Context) ([]string, error)
	
	// Close closes the storage and releases resources
	Close() error
}

// StoredCredentials represents credentials stored in the cache
type StoredCredentials struct {
	AccessKeyID     string    `json:"access_key_id"`
	SecretAccessKey string    `json:"secret_access_key"`
	SessionToken    string    `json:"session_token"`
	Expiration      time.Time `json:"expiration"`
	RoleARN         string    `json:"role_arn"`
	ExternalID      string    `json:"external_id"`
	SessionName     string    `json:"session_name"`
	CreatedAt       time.Time `json:"created_at"`
	LastUsed        time.Time `json:"last_used"`
}

// IsValid returns true if the credentials are still valid
func (c *StoredCredentials) IsValid() bool {
	if c.Expiration.IsZero() {
		return false
	}
	
	// Consider credentials expired 5 minutes before actual expiration
	return time.Now().Before(c.Expiration.Add(-5 * time.Minute))
}

// TimeUntilExpiration returns the duration until credentials expire
func (c *StoredCredentials) TimeUntilExpiration() time.Duration {
	if c.Expiration.IsZero() {
		return 0
	}
	return time.Until(c.Expiration)
}

// FileStorage implements credential storage using encrypted files
type FileStorage struct {
	baseDir   string
	encryptor *encryption.Encryptor
	mu        sync.RWMutex
}

// NewFileStorage creates a new file-based credential storage
func NewFileStorage(baseDir, password string) (*FileStorage, error) {
	// Create base directory if it doesn't exist
	if err := os.MkdirAll(baseDir, 0700); err != nil {
		return nil, fmt.Errorf("failed to create storage directory: %w", err)
	}
	
	var encryptor *encryption.Encryptor
	var err error
	
	if password != "" {
		// Use provided password
		encryptor = encryption.NewEncryptor(password)
	} else {
		// Use environment-based encryption
		encryptor, err = encryption.NewEncryptorFromEnv()
		if err != nil {
			return nil, fmt.Errorf("failed to create encryptor: %w", err)
		}
	}
	
	return &FileStorage{
		baseDir:   baseDir,
		encryptor: encryptor,
	}, nil
}

// Store saves encrypted credentials to a file
func (fs *FileStorage) Store(ctx context.Context, key string, credentials *StoredCredentials) error {
	fs.mu.Lock()
	defer fs.mu.Unlock()
	
	if err := validateCredentialKey(key); err != nil {
		return fmt.Errorf("invalid credential key: %w", err)
	}
	
	// Update timestamps
	now := time.Now()
	credentials.CreatedAt = now
	credentials.LastUsed = now
	
	// Serialize credentials
	data, err := json.Marshal(credentials)
	if err != nil {
		return fmt.Errorf("failed to marshal credentials: %w", err)
	}
	
	// Encrypt credentials
	encrypted, err := fs.encryptor.EncryptString(string(data))
	if err != nil {
		return fmt.Errorf("failed to encrypt credentials: %w", err)
	}
	
	// Write to file
	filePath := fs.getFilePath(key)
	if err := os.WriteFile(filePath, []byte(encrypted), 0600); err != nil {
		return fmt.Errorf("failed to write credentials file: %w", err)
	}
	
	return nil
}

// Retrieve gets and decrypts credentials from a file
func (fs *FileStorage) Retrieve(ctx context.Context, key string) (*StoredCredentials, error) {
	fs.mu.RLock()
	defer fs.mu.RUnlock()
	
	if err := validateCredentialKey(key); err != nil {
		return nil, fmt.Errorf("invalid credential key: %w", err)
	}
	
	filePath := fs.getFilePath(key)
	
	// Check if file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return nil, fmt.Errorf("credentials not found for key: %s", key)
	}
	
	// Read encrypted data
	encryptedData, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read credentials file: %w", err)
	}
	
	// Decrypt data
	decryptedData, err := fs.encryptor.DecryptString(string(encryptedData))
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt credentials: %w", err)
	}
	
	// Unmarshal credentials
	var credentials StoredCredentials
	if err := json.Unmarshal([]byte(decryptedData), &credentials); err != nil {
		return nil, fmt.Errorf("failed to unmarshal credentials: %w", err)
	}
	
	// Update last used time
	credentials.LastUsed = time.Now()
	
	// Save updated credentials back (async to avoid blocking)
	go func() {
		fs.Store(context.Background(), key, &credentials)
	}()
	
	return &credentials, nil
}

// Delete removes credentials file
func (fs *FileStorage) Delete(ctx context.Context, key string) error {
	fs.mu.Lock()
	defer fs.mu.Unlock()
	
	if err := validateCredentialKey(key); err != nil {
		return fmt.Errorf("invalid credential key: %w", err)
	}
	
	filePath := fs.getFilePath(key)
	
	if err := os.Remove(filePath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to delete credentials file: %w", err)
	}
	
	return nil
}

// List returns all credential keys
func (fs *FileStorage) List(ctx context.Context) ([]string, error) {
	fs.mu.RLock()
	defer fs.mu.RUnlock()
	
	entries, err := os.ReadDir(fs.baseDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read storage directory: %w", err)
	}
	
	var keys []string
	for _, entry := range entries {
		if !entry.IsDir() && hasCredentialExtension(entry.Name()) {
			key := removeCredentialExtension(entry.Name())
			keys = append(keys, key)
		}
	}
	
	return keys, nil
}

// Close cleans up the file storage
func (fs *FileStorage) Close() error {
	// No cleanup needed for file storage
	return nil
}

// getFilePath returns the file path for a given key
func (fs *FileStorage) getFilePath(key string) string {
	filename := sanitizeFilename(key) + ".cred"
	return filepath.Join(fs.baseDir, filename)
}

// MemoryStorage implements in-memory credential storage (for testing)
type MemoryStorage struct {
	credentials map[string]*StoredCredentials
	mu          sync.RWMutex
}

// NewMemoryStorage creates a new in-memory credential storage
func NewMemoryStorage() *MemoryStorage {
	return &MemoryStorage{
		credentials: make(map[string]*StoredCredentials),
	}
}

// Store saves credentials in memory
func (ms *MemoryStorage) Store(ctx context.Context, key string, credentials *StoredCredentials) error {
	ms.mu.Lock()
	defer ms.mu.Unlock()
	
	if err := validateCredentialKey(key); err != nil {
		return fmt.Errorf("invalid credential key: %w", err)
	}
	
	// Update timestamps
	now := time.Now()
	credentials.CreatedAt = now
	credentials.LastUsed = now
	
	// Store a copy to prevent external modifications
	credsCopy := *credentials
	ms.credentials[key] = &credsCopy
	
	return nil
}

// Retrieve gets credentials from memory
func (ms *MemoryStorage) Retrieve(ctx context.Context, key string) (*StoredCredentials, error) {
	ms.mu.RLock()
	defer ms.mu.RUnlock()
	
	if err := validateCredentialKey(key); err != nil {
		return nil, fmt.Errorf("invalid credential key: %w", err)
	}
	
	credentials, exists := ms.credentials[key]
	if !exists {
		return nil, fmt.Errorf("credentials not found for key: %s", key)
	}
	
	// Return a copy and update last used time
	credsCopy := *credentials
	credsCopy.LastUsed = time.Now()
	
	// Update the stored copy with new last used time
	credentials.LastUsed = credsCopy.LastUsed
	
	return &credsCopy, nil
}

// Delete removes credentials from memory
func (ms *MemoryStorage) Delete(ctx context.Context, key string) error {
	ms.mu.Lock()
	defer ms.mu.Unlock()
	
	if err := validateCredentialKey(key); err != nil {
		return fmt.Errorf("invalid credential key: %w", err)
	}
	
	delete(ms.credentials, key)
	return nil
}

// List returns all credential keys
func (ms *MemoryStorage) List(ctx context.Context) ([]string, error) {
	ms.mu.RLock()
	defer ms.mu.RUnlock()
	
	keys := make([]string, 0, len(ms.credentials))
	for key := range ms.credentials {
		keys = append(keys, key)
	}
	
	return keys, nil
}

// Close cleans up memory storage
func (ms *MemoryStorage) Close() error {
	ms.mu.Lock()
	defer ms.mu.Unlock()
	
	// Clear all credentials
	ms.credentials = make(map[string]*StoredCredentials)
	return nil
}

// Helper functions

func validateCredentialKey(key string) error {
	if key == "" {
		return fmt.Errorf("credential key cannot be empty")
	}
	
	if len(key) > 100 {
		return fmt.Errorf("credential key too long (max 100 characters)")
	}
	
	// Check for valid characters (alphanumeric, dash, underscore, dot)
	for _, ch := range key {
		if !isValidKeyChar(ch) {
			return fmt.Errorf("credential key contains invalid character: %c", ch)
		}
	}
	
	return nil
}

func isValidKeyChar(ch rune) bool {
	return (ch >= 'a' && ch <= 'z') ||
		   (ch >= 'A' && ch <= 'Z') ||
		   (ch >= '0' && ch <= '9') ||
		   ch == '-' || ch == '_' || ch == '.'
}

func sanitizeFilename(key string) string {
	// Replace any potentially problematic characters
	safe := make([]rune, 0, len(key))
	for _, ch := range key {
		if isValidKeyChar(ch) {
			safe = append(safe, ch)
		} else {
			safe = append(safe, '_')
		}
	}
	return string(safe)
}

func hasCredentialExtension(filename string) bool {
	return len(filename) > 5 && filename[len(filename)-5:] == ".cred"
}

func removeCredentialExtension(filename string) string {
	if hasCredentialExtension(filename) {
		return filename[:len(filename)-5]
	}
	return filename
}

// CleanupExpiredCredentials removes expired credentials from storage
func CleanupExpiredCredentials(ctx context.Context, storage CredentialStorage) error {
	keys, err := storage.List(ctx)
	if err != nil {
		return fmt.Errorf("failed to list credentials: %w", err)
	}
	
	var errors []error
	for _, key := range keys {
		credentials, err := storage.Retrieve(ctx, key)
		if err != nil {
			errors = append(errors, fmt.Errorf("failed to retrieve credentials for key %s: %w", key, err))
			continue
		}
		
		if !credentials.IsValid() {
			if err := storage.Delete(ctx, key); err != nil {
				errors = append(errors, fmt.Errorf("failed to delete expired credentials for key %s: %w", key, err))
			}
		}
	}
	
	if len(errors) > 0 {
		return fmt.Errorf("cleanup completed with %d errors: %v", len(errors), errors[0])
	}
	
	return nil
}