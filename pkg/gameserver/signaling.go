package gameserver

import (
	"context"
	"log"

	"github.com/pion/webrtc/v3"
)

type ICEChannels struct {
	OfferCandidate  chan webrtc.ICECandidateInit
	AnswerCandidate chan webrtc.ICECandidateInit
}

func NewICEChannels() *ICEChannels {
	return &ICEChannels{
		OfferCandidate:  make(chan webrtc.ICECandidateInit),
		AnswerCandidate: make(chan webrtc.ICECandidateInit),
	}
}

func StartSignaling(ctx context.Context, offer webrtc.SessionDescription, iceChannels *ICEChannels) (*webrtc.SessionDescription, error) {
	pcAnswer, err := webrtc.NewPeerConnection(webrtc.Configuration{
		ICEServers: []webrtc.ICEServer{
			{URLs: []string{"stun:stun.l.google.com:19302"}},
		},
	})
	if err != nil {
		return nil, err
	}
	pcAnswer.OnICEConnectionStateChange(func(state webrtc.ICEConnectionState) {
		log.Printf("[pcAnswer] ICE state has changed: %s", state)
	})
	pcAnswer.OnConnectionStateChange(func(state webrtc.PeerConnectionState) {
		log.Printf("[pcAnswer] connection state has changed: %s", state)
	})
	pcAnswer.OnICECandidate(func(candidate *webrtc.ICECandidate) {
		log.Printf("[pcAnswer] new ICE candidate: %v", candidate)
		if candidate != nil && pcAnswer.RemoteDescription() != nil {
			iceChannels.AnswerCandidate <- candidate.ToJSON()
		}
	})
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case candidate := <-iceChannels.OfferCandidate:
				if err := pcAnswer.AddICECandidate(candidate); err != nil {
					log.Printf("failed to add ice candidate from pcOffer: %+v", err)
				}
			}
		}
	}()
	if err := pcAnswer.SetRemoteDescription(offer); err != nil {
		return nil, err
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
