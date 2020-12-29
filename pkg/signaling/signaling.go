package signaling

import (
	"context"
	"log"

	"github.com/pion/webrtc/v3"
)

type iceChannels struct {
	offerCandidate  chan webrtc.ICECandidateInit
	answerCandidate chan webrtc.ICECandidateInit
}

func startSignaling(ctx context.Context, offer webrtc.SessionDescription) (*webrtc.SessionDescription, *iceChannels, error) {
	pcAnswer, err := webrtc.NewPeerConnection(webrtc.Configuration{
		ICEServers: []webrtc.ICEServer{
			{URLs: []string{"stun:stun.l.google.com:19302"}},
		},
	})
	if err != nil {
		return nil, nil, err
	}
	iceChannels := &iceChannels{
		offerCandidate:  make(chan webrtc.ICECandidateInit),
		answerCandidate: make(chan webrtc.ICECandidateInit),
	}
	pcAnswer.OnICECandidate(func(candidate *webrtc.ICECandidate) {
		log.Printf("new ICE candidate: %v", candidate)
		if candidate != nil && pcAnswer.RemoteDescription() != nil {
			iceChannels.answerCandidate <- candidate.ToJSON()
		}
	})
	go func() {
		log.Printf("started signaling with trickle ICE")
		for {
			select {
			case <-ctx.Done():
				return
			case candidate := <-iceChannels.offerCandidate:
				if err := pcAnswer.AddICECandidate(candidate); err != nil {
					log.Printf("failed to add ice candidate from pcOffer: %+v", err)
				}
			}
		}
	}()
	if err := pcAnswer.SetRemoteDescription(offer); err != nil {
		return nil, nil, err
	}
	answer, err := pcAnswer.CreateAnswer(nil)
	if err != nil {
		return nil, nil, err
	}
	if err := pcAnswer.SetLocalDescription(answer); err != nil {
		return nil, nil, err
	}
	return pcAnswer.LocalDescription(), iceChannels, nil
}
