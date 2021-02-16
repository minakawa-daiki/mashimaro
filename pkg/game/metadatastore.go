package game

import (
	"context"
	"errors"
	"sync"
)

var (
	ErrMetadataNotFound = errors.New("session not found")
)

type MetadataStore interface {
	GetGameMetadata(ctx context.Context, gameID string) (*Metadata, error)
}

type MockMetadataStore struct {
	metadatas map[string]*Metadata
	mu        sync.RWMutex
}

func NewMockMetadataStore() *MockMetadataStore {
	return &MockMetadataStore{
		metadatas: make(map[string]*Metadata),
	}
}

func (s *MockMetadataStore) AddGameMetadata(ctx context.Context, gameID string, metadata *Metadata) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.metadatas[gameID] = metadata
	return nil
}

func (s *MockMetadataStore) GetGameMetadata(ctx context.Context, gameID string) (*Metadata, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	m, ok := s.metadatas[gameID]
	if !ok {
		return nil, ErrMetadataNotFound
	}
	return m, nil
}
