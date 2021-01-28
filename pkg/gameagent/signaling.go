package gameagent

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"sync"

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

func startTrickleICE(ctx context.Context, sid gamesession.SessionID, c proto.SignalingClient, pcAnswer *webrtc.PeerConnection, remoteFound chan struct{}) error {
	trickleStream, err := c.TrickleSignaling(ctx)
	if err != nil {
		return fmt.Errorf("failed to call trickle signaling: %+v", err)
	}
	answerGatheringCtx, cancelAnswerGathering := context.WithCancel(ctx)
	answerCandidateCh := make(chan *webrtc.ICECandidate)
	pcAnswer.OnICECandidate(func(candidate *webrtc.ICECandidate) {
		log.Printf("[pcAnswer] new ICE candidate: %v", candidate)
		answerCandidateCh <- candidate
	})
	go func() {
		for {
			select {
			case <-answerGatheringCtx.Done():
				log.Printf("[pcAnswer] stopped goroutine to gathering ICE candidates")
				return
			case answerCandidate := <-answerCandidateCh:
				answerBody := ""
				if answerCandidate != nil {
					j := answerCandidate.ToJSON()
					b, err := webrtcutil.EncodeICECandidate(&j)
					if err != nil {
						log.Printf("failed to encode ICE candidate: %+v", err)
						continue
					}
					answerBody = b
				}
				if err := trickleStream.Send(&proto.TrickleSignalingRequest{
					SessionId: string(sid),
					Candidate: &proto.ICECandidate{
						Body: answerBody,
					},
				}); err != nil {
					return
				}
				if answerBody == "" {
					log.Printf("[pcAnswer] finished gathering ICE candidates")
					cancelAnswerGathering()
				}
			}
		}
	}()

	offerGatheringCtx, cancelOfferGathering := context.WithCancel(ctx)
	var pendingCandidates []webrtc.ICECandidateInit
	var pendingMu sync.Mutex
	go func() {
		for {
			resp, err := trickleStream.Recv()
			if errors.Is(err, io.EOF) {
				log.Printf("stream EOF received")
				break
			}
			if err != nil {
				return
			}
			if resp.Candidate.Body == "" {
				log.Printf("[agent] received end of gathering offer ICE candidates")
				cancelOfferGathering()
				return
			}
			offerCandidate, err := webrtcutil.DecodeICECandidate(resp.Candidate.Body)
			if err != nil {
				log.Printf("failed to decode ICE candidate: %+v", err)
				return
			}
			if pcAnswer.RemoteDescription() != nil {
				if err := pcAnswer.AddICECandidate(*offerCandidate); err != nil {
					log.Printf("failed to add ice candidate from pcOffer: %+v", err)
				}
			} else {
				pendingMu.Lock()
				pendingCandidates = append(pendingCandidates, *offerCandidate)
				pendingMu.Unlock()
			}
		}
	}()
	go func() {
		for {
			select {
			case <-offerGatheringCtx.Done():
				return
			case <-remoteFound:
				(func() {
					pendingMu.Lock()
					defer pendingMu.Unlock()
					if len(pendingCandidates) > 0 {
						log.Printf("[agent] remote SDP found, adding pending ICE candidates to pcAnswer")
					}
					for _, candidate := range pendingCandidates {
						if err := pcAnswer.AddICECandidate(candidate); err != nil {
							log.Printf("failed to add ICE candidate: %+v", err)
						}
					}
				})()
				return
			}
		}
	}()

	return nil
}

func createAnswer(ctx context.Context, pcAnswer *webrtc.PeerConnection, offer *webrtc.SessionDescription, remoteFound chan struct{}) (*webrtc.SessionDescription, error) {
	if err := pcAnswer.SetRemoteDescription(*offer); err != nil {
		return nil, err
	}
	close(remoteFound)
	answer, err := pcAnswer.CreateAnswer(nil)
	if err != nil {
		return nil, err
	}
	if err := pcAnswer.SetLocalDescription(answer); err != nil {
		return nil, err
	}
	return pcAnswer.LocalDescription(), nil
}
