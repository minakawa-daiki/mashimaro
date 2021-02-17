package gamemetadata

import (
	"context"
	"errors"
	"sync"
)

var (
	ErrMetadataNotFound = errors.New("session not found")
)

type Store interface {
	GetGameMetadata(ctx context.Context, gameID string) (*Metadata, error)
}

type InMemoryStore struct {
	metas map[string]*Metadata
	mu    sync.RWMutex
}

func NewInMemoryMetadataStore() *InMemoryStore {
	return &InMemoryStore{
		metas: make(map[string]*Metadata),
	}
}

func (s *InMemoryStore) AddGameMetadata(ctx context.Context, gameID string, metadata *Metadata) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.metas[gameID] = metadata
	return nil
}

func (s *InMemoryStore) GetGameMetadata(ctx context.Context, gameID string) (*Metadata, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	m, ok := s.metas[gameID]
	if !ok {
		return nil, ErrMetadataNotFound
	}
	return m, nil
}
