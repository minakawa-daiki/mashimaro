package gamesession

import (
	"context"
	"fmt"
	"sync"

	"google.golang.org/grpc"

	"github.com/castaneai/mashimaro/pkg/proto"

	"github.com/google/uuid"
)

type SessionID string

type Manager struct {
	allocator Allocator
	sessions  map[SessionID]*Session
	mu        sync.RWMutex
}

func NewManager(allocator Allocator) *Manager {
	return &Manager{
		allocator: allocator,
		sessions:  make(map[SessionID]*Session),
		mu:        sync.RWMutex{},
	}
}

func (m *Manager) NewSession(ctx context.Context, gameID string) (*Session, error) {
	sid := SessionID(uuid.Must(uuid.NewRandom()).String())
	gs, err := m.allocator.Allocate(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to allocate gameserver: %+v", err)
	}
	cc, err := grpc.Dial(gs.Addr, grpc.WithInsecure())
	if err != nil {
		return nil, fmt.Errorf("failed to connect to gameserver: %+v", err)
	}
	rpcc := proto.NewGameServerClient(cc)

	m.mu.Lock()
	defer m.mu.Unlock()
	m.sessions[sid] = &Session{SessionID: sid, GameID: gameID, GameServer: gs, RPCClient: rpcc}
	return m.sessions[sid], nil
}

func (m *Manager) GetSession(sid SessionID) (*Session, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	ss, ok := m.sessions[sid]
	return ss, ok
}

func (m *Manager) ExitSession(sid SessionID) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.sessions, sid)
}

type Session struct {
	SessionID  SessionID
	GameID     string
	GameServer *GameServer
	RPCClient  proto.GameServerClient
}
