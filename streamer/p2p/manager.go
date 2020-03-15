package p2p

import (
	"context"
	"log"
)

type Manager struct {
	idcnt  int
	peers  map[int]*Peer
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
			if err := peer.Start(); err != nil {
				log.Printf("failed to start peer: %+v", err)
				continue
			}
			m.idcnt++
			m.peers[m.idcnt] = peer
		}
	}
}
