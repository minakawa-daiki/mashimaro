package gamesession

import (
	"context"
	"errors"
	"log"
	"sync"

	"github.com/castaneai/mashimaro/pkg/gameserver"

	"github.com/google/uuid"
)

type Store interface {
	NewSession(ctx context.Context, req *NewSessionRequest) (*Session, error)
	GetSession(ctx context.Context, sid SessionID) (*Session, error)
	GetSessionByGameServerName(ctx context.Context, gsName string) (*Session, error)
	UpdateSessionState(ctx context.Context, sid SessionID, newState State) error
}

type NewSessionRequest struct {
	GameID     string
	GameServer *gameserver.GameServer
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
		SessionID:      sid,
		State:          StateWaitingForSession,
		GameID:         req.GameID,
		GameServerName: req.GameServer.Name,
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

func (s *InMemoryStore) GetSessionByGameServerName(ctx context.Context, gsName string) (*Session, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, ss := range s.sessions {
		if ss.GameServerName == gsName {
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
