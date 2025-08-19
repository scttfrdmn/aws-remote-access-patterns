package crossaccount

import (
	"context"
	"fmt"
	"sync"
)

// CredentialStorage defines how customer credentials are stored
// This interface allows you to use different storage backends
type CredentialStorage interface {
	Store(ctx context.Context, customerID string, creds CustomerCredentials) error
	Get(ctx context.Context, customerID string) (*CustomerCredentials, error)
	Delete(ctx context.Context, customerID string) error
	List(ctx context.Context) ([]string, error)
}

// MemoryStorage provides simple in-memory storage for development
// For production, implement with encrypted database storage
type MemoryStorage struct {
	mu    sync.RWMutex
	creds map[string]CustomerCredentials
}

// NewMemoryStorage creates a new in-memory storage
func NewMemoryStorage() *MemoryStorage {
	return &MemoryStorage{
		creds: make(map[string]CustomerCredentials),
	}
}

func (m *MemoryStorage) Store(ctx context.Context, customerID string, creds CustomerCredentials) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	m.creds[customerID] = creds
	return nil
}

func (m *MemoryStorage) Get(ctx context.Context, customerID string) (*CustomerCredentials, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	creds, exists := m.creds[customerID]
	if !exists {
		return nil, fmt.Errorf("customer %s not found", customerID)
	}
	
	return &creds, nil
}

func (m *MemoryStorage) Delete(ctx context.Context, customerID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	delete(m.creds, customerID)
	return nil
}

func (m *MemoryStorage) List(ctx context.Context) ([]string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	customers := make([]string, 0, len(m.creds))
	for customerID := range m.creds {
		customers = append(customers, customerID)
	}
	
	return customers, nil
}