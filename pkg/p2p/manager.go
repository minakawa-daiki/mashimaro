package p2p

import (
	"context"
	"log"
	"sync"
)

type Manager struct {
	idcnt  int
	peers  map[int]*Peer
	mu     sync.RWMutex
	peerch chan *Peer
}

func NewManager() *Manager {
	return &Manager{peers: make(map[int]*Peer), peerch: make(chan *Peer)}
}

func (m *Manager) AddPeer(p *Peer) {
	m.peerch <- p
}

func (m *Manager) Start(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case peer := <-m.peerch:
			m.idcnt++
			m.mu.Lock()
			m.peers[m.idcnt] = peer
			m.mu.Unlock()

			go func(id int) {
				defer func() {
					m.mu.Lock()
					defer m.mu.Unlock()
					delete(m.peers, id)
					log.Printf("removed peer(id: %v)", id)
				}()
				if err := peer.Start(); err != nil {
					log.Printf("failed to serve peer: %+v", err)
				}
			}(m.idcnt)
			log.Printf("added new peer(id: %v)", m.idcnt)
		}
	}
}
