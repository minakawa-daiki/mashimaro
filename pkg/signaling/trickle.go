package signaling

import (
	"context"
	"fmt"
	"log"
	"sync"

	"github.com/castaneai/mashimaro/pkg/proto"

	"github.com/pion/webrtc/v3"

	"github.com/castaneai/mashimaro/pkg/gamesession"
)

type trickleSession struct {
	stream proto.GameServer_TrickleSignalingClient
}

type trickleManager struct {
	sessions map[gamesession.SessionID]*trickleSession
	mu       sync.RWMutex
}

func newTrickleManager() *trickleManager {
	return &trickleManager{
		sessions: make(map[gamesession.SessionID]*trickleSession),
		mu:       sync.RWMutex{},
	}
}

func (m *trickleManager) NewSession(ctx context.Context, ss *gamesession.Session, onAnswerICECandidate func(init webrtc.ICECandidateInit)) {
	stream, err := ss.RPCClient.TrickleSignaling(ctx)
	if err != nil {
		log.Printf("failed to call trickle signaling: %+v", err)
		return
	}
	go func() {
		defer func() {
			m.DeleteSession(ss.SessionID)
		}()
		for {
			select {
			case <-ctx.Done():
				return
			default:
				recv, err := stream.Recv()
				if err != nil {
					log.Printf("failed to recv from trickle signaling stream: %+v", err)
					return
				}
				onAnswerICECandidate(decodeICECandidate(recv.Body))
			}
		}
	}()

	m.mu.Lock()
	defer m.mu.Unlock()
	m.sessions[ss.SessionID] = &trickleSession{
		stream: stream,
	}
}

func (m *trickleManager) AddICECandidate(sid gamesession.SessionID, candidate webrtc.ICECandidateInit) error {
	m.mu.RLock()
	defer m.mu.RUnlock()
	ts, ok := m.sessions[sid]
	if !ok {
		return fmt.Errorf("trickle session not found (sid: %s)", sid)
	}
	if err := ts.stream.Send(&proto.ICECandidate{Body: encodeICECandidate(candidate)}); err != nil {
		return fmt.Errorf("failed to send ice candidate: %+v", err)
	}
	return nil
}

func (m *trickleManager) DeleteSession(sid gamesession.SessionID) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.sessions, sid)
}
