package gamesession

import (
	"context"
	"errors"
	"log"
	"sync"

	"github.com/google/uuid"
)

type Store interface {
	NewSession(ctx context.Context, req *NewSessionRequest) (*Session, error)
	GetSession(ctx context.Context, sid SessionID) (*Session, error)
	GetSessionByAllocatedServerID(ctx context.Context, allocatedServerID string) (*Session, error)
	UpdateSessionState(ctx context.Context, sid SessionID, newState State) error
	DeleteSession(ctx context.Context, sid SessionID) error
}

type NewSessionRequest struct {
	GameID            string
	AllocatedServerID string
}

var (
	ErrSessionNotFound = errors.New("session not found")
)

type InMemoryStore struct {
	sessions map[SessionID]*Session
	mu       sync.RWMutex
}

func NewInMemoryStore() *InMemoryStore {
	return &InMemoryStore{
		sessions: make(map[SessionID]*Session),
		mu:       sync.RWMutex{},
	}
}

func (s *InMemoryStore) NewSession(ctx context.Context, req *NewSessionRequest) (*Session, error) {
	sid := SessionID(uuid.Must(uuid.NewRandom()).String())
	ss := &Session{
		SessionID:         sid,
		State:             StateWaitingForSession,
		GameID:            req.GameID,
		AllocatedServerID: req.AllocatedServerID,
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.sessions[sid] = ss
	return ss, nil
}

func (s *InMemoryStore) GetSession(ctx context.Context, sid SessionID) (*Session, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	ss, ok := s.sessions[sid]
	if !ok {
		return nil, ErrSessionNotFound
	}
	return ss, nil
}

func (s *InMemoryStore) GetSessionByAllocatedServerID(ctx context.Context, allocatedServerID string) (*Session, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, ss := range s.sessions {
		if ss.AllocatedServerID == allocatedServerID {
			return ss, nil
		}
	}
	return nil, ErrSessionNotFound
}

func (s *InMemoryStore) UpdateSessionState(ctx context.Context, sid SessionID, newState State) error {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, ss := range s.sessions {
		if ss.SessionID == sid {
			log.Printf("update session state %s -> %s", ss.State, newState)
			s.sessions[sid].State = newState
			return nil
		}
	}
	return ErrSessionNotFound
}

func (s *InMemoryStore) DeleteSession(ctx context.Context, sid SessionID) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.sessions, sid)
	return nil
}
