package gameagent

import (
	"context"
	"fmt"
	"io"
	"log"

	"github.com/castaneai/mashimaro/pkg/gamesession"

	"github.com/castaneai/mashimaro/pkg/webrtcutil"

	"github.com/castaneai/mashimaro/pkg/proto"

	"github.com/pion/webrtc/v3"
)

func newPeerConnection() (*webrtc.PeerConnection, error) {
	pcAnswer, err := webrtc.NewPeerConnection(webrtc.Configuration{
		ICEServers: []webrtc.ICEServer{
			// TODO: stun server from config
			{URLs: []string{"stun:stun.l.google.com:19302"}},
		},
	})
	if err != nil {
		return nil, err
	}

	pcAnswer.OnICEConnectionStateChange(func(state webrtc.ICEConnectionState) {
		log.Printf("[pcAnswer] ICE state has changed: %s", state)
	})
	return pcAnswer, nil
}

func startSignaling(ctx context.Context, sid gamesession.SessionID, c proto.SignalingClient, pcAnswer *webrtc.PeerConnection, offer *webrtc.SessionDescription) (*webrtc.SessionDescription, error) {
	trickleStream, err := c.TrickleSignaling(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to call trickle signaling: %+v", err)
	}
	answerCandidateCh := make(chan *webrtc.ICECandidate)
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case answerCandidate := <-answerCandidateCh:
				j := answerCandidate.ToJSON()
				answerBody, err := webrtcutil.EncodeICECandidate(&j)
				if err != nil {
					log.Printf("failed to encode ICE candidate: %+v", err)
					continue
				}
				if err := trickleStream.Send(&proto.TrickleSignalingRequest{
					SessionId: string(sid),
					Candidate: &proto.ICECandidate{
						Body: answerBody,
					},
				}); err != nil {
					return
				}
			}
		}
	}()
	pcAnswer.OnICECandidate(func(candidate *webrtc.ICECandidate) {
		log.Printf("[pcAnswer] new ICE candidate: %v", candidate)
		if candidate != nil && pcAnswer.RemoteDescription() != nil {
			answerCandidateCh <- candidate
		}
	})

	if err := pcAnswer.SetRemoteDescription(*offer); err != nil {
		return nil, err
	}
	answer, err := pcAnswer.CreateAnswer(nil)
	if err != nil {
		return nil, err
	}
	if err := pcAnswer.SetLocalDescription(answer); err != nil {
		return nil, err
	}

	go func() {
		for {
			resp, err := trickleStream.Recv()
			if err == io.EOF {
				log.Printf("stream EOF received")
				break
			}
			if err != nil {
				return
			}
			offerCandidate, err := webrtcutil.DecodeICECandidate(resp.Candidate.Body)
			if err != nil {
				log.Printf("failed to decode ICE candidate: %+v", err)
				return
			}
			if err := pcAnswer.AddICECandidate(*offerCandidate); err != nil {
				log.Printf("failed to add ice candidate from pcOffer: %+v", err)
			}
		}
	}()

	return pcAnswer.LocalDescription(), nil
}
