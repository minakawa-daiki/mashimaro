package gameserver

import (
	"context"
	"log"
	"sync"

	"github.com/pion/webrtc/v3"
)

type connEventHandler struct {
	InitConnection  func(pc *webrtc.PeerConnection) error
	OfferCandidate  chan webrtc.ICECandidateInit
	AnswerCandidate chan webrtc.ICECandidateInit
}

func startSignaling(ctx context.Context, offer webrtc.SessionDescription, handler *connEventHandler) (*webrtc.SessionDescription, error) {
	pcAnswer, err := webrtc.NewPeerConnection(webrtc.Configuration{
		ICEServers: []webrtc.ICEServer{
			{URLs: []string{"stun:stun.l.google.com:19302"}},
		},
	})
	if err != nil {
		return nil, err
	}
	if err := handler.InitConnection(pcAnswer); err != nil {
		return nil, err
	}
	pcAnswer.OnICEConnectionStateChange(func(state webrtc.ICEConnectionState) {
		log.Printf("[pcAnswer] ICE state has changed: %s", state)
	})

	pcAnswer.OnICECandidate(func(candidate *webrtc.ICECandidate) {
		log.Printf("[pcAnswer] new ICE candidate: %v", candidate)
		if candidate != nil && pcAnswer.RemoteDescription() != nil {
			handler.AnswerCandidate <- candidate.ToJSON()
		}
	})

	var pendingCandidates []webrtc.ICECandidateInit
	var pendingMu sync.Mutex
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case candidate := <-handler.OfferCandidate:
				if pcAnswer.RemoteDescription() != nil {
					if err := pcAnswer.AddICECandidate(candidate); err != nil {
						log.Printf("failed to add ice candidate from pcOffer: %+v", err)
					}
				} else {
					pendingMu.Lock()
					pendingCandidates = append(pendingCandidates, candidate)
					pendingMu.Unlock()
				}
			}
		}
	}()
	if err := pcAnswer.SetRemoteDescription(offer); err != nil {
		return nil, err
	}
	pendingMu.Lock()
	defer pendingMu.Unlock()
	for _, candidate := range pendingCandidates {
		if err := pcAnswer.AddICECandidate(candidate); err != nil {
			return nil, err
		}
	}
	answer, err := pcAnswer.CreateAnswer(nil)
	if err != nil {
		return nil, err
	}
	if err := pcAnswer.SetLocalDescription(answer); err != nil {
		return nil, err
	}
	return pcAnswer.LocalDescription(), nil
}
