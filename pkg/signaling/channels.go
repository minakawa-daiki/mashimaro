package signaling

import (
	"sync"

	"github.com/castaneai/mashimaro/pkg/gamesession"
)

const (
	channelBuffer = 10
)

// For communication between external server and internal server
type Channels struct {
	offerChs           map[gamesession.SessionID]chan string
	answerChs          map[gamesession.SessionID]chan string
	offerCandidateChs  map[gamesession.SessionID]chan string
	answerCandidateChs map[gamesession.SessionID]chan string
	mu                 sync.RWMutex
}

func (c *Channels) getOrCreateCh(m map[gamesession.SessionID]chan string, sid gamesession.SessionID) chan string {
	c.mu.Lock()
	defer c.mu.Unlock()
	ch, ok := m[sid]
	if !ok {
		ch = make(chan string, channelBuffer)
		m[sid] = ch
	}
	return ch
}

func (c *Channels) OfferCh(sid gamesession.SessionID) chan string {
	return c.getOrCreateCh(c.offerChs, sid)
}

func (c *Channels) AnswerCh(sid gamesession.SessionID) chan string {
	return c.getOrCreateCh(c.answerChs, sid)
}

func (c *Channels) OfferCandidateCh(sid gamesession.SessionID) chan string {
	return c.getOrCreateCh(c.offerCandidateChs, sid)
}

func (c *Channels) AnswerCandidateCh(sid gamesession.SessionID) chan string {
	return c.getOrCreateCh(c.answerCandidateChs, sid)
}

func NewChannels() *Channels {
	return &Channels{
		offerChs:           make(map[gamesession.SessionID]chan string),
		answerChs:          make(map[gamesession.SessionID]chan string),
		offerCandidateChs:  make(map[gamesession.SessionID]chan string),
		answerCandidateChs: make(map[gamesession.SessionID]chan string),
	}
}
