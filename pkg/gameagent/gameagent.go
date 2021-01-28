package gameagent

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/pion/webrtc/v3/pkg/media"

	"github.com/castaneai/mashimaro/pkg/gamesession"

	"github.com/castaneai/mashimaro/pkg/webrtcutil"

	"github.com/pion/webrtc/v3"

	"github.com/castaneai/mashimaro/pkg/game"
	"github.com/goccy/go-yaml"

	"github.com/castaneai/mashimaro/pkg/proto"
)

type Agent struct {
	brokerClient    proto.BrokerClient
	signalingClient proto.SignalingClient
}

func NewAgent(brokerClient proto.BrokerClient, signalingClient proto.SignalingClient) *Agent {
	return &Agent{brokerClient: brokerClient, signalingClient: signalingClient}
}

func (a *Agent) Run(ctx context.Context, gsName string) error {
	ss, err := waitForSession(ctx, a.brokerClient, gsName)
	if err != nil {
		return err
	}
	sid := gamesession.SessionID(ss.SessionId)

	// TODO: provisioning game data and ready to start process

	offer, err := waitForOffer(ctx, a.signalingClient, gamesession.SessionID(ss.SessionId))
	if err != nil {
		return err
	}
	pcAnswer, err := newPeerConnection()
	if err != nil {
		return err
	}
	tracks, err := NewMediaTracks()
	if err != nil {
		return err
	}
	if _, err := pcAnswer.AddTrack(tracks.VideoTrack); err != nil {
		return err
	}
	if _, err := pcAnswer.AddTrack(tracks.AudioTrack); err != nil {
		return err
	}

	once := &sync.Once{}
	connected := make(chan struct{})
	pcAnswer.OnConnectionStateChange(func(state webrtc.PeerConnectionState) {
		log.Printf("[pcAnswer] connection state has changed: %s", state)
		switch state {
		case webrtc.PeerConnectionStateConnected:
			once.Do(func() {
				close(connected)
			})
		case webrtc.PeerConnectionStateDisconnected:
			// TODO:
		}
	})

	remoteFound := make(chan struct{})
	if err := startTrickleICE(ctx, sid, a.signalingClient, pcAnswer, remoteFound); err != nil {
		return err
	}
	answer, err := createAnswer(ctx, pcAnswer, offer, remoteFound)
	if err != nil {
		return err
	}
	answerBody, err := webrtcutil.EncodeSDP(answer)
	if err != nil {
		return fmt.Errorf("faield to encode answer SDP: %+v", err)
	}
	if _, err := a.signalingClient.SendSignalingAnswer(ctx, &proto.SendSignalingAnswerRequest{
		SessionId:       ss.SessionId,
		SignalingAnswer: &proto.SignalingAnswer{Body: answerBody},
	}); err != nil {
		return err
	}
	log.Printf("[pcAnswer] sent answer")

	// TODO: wait for provisioning game

	<-connected
	log.Printf("[pcAnswer] connected to peer!")
	ticker := time.NewTicker(1 * time.Second)
	for range ticker.C {
		if err := tracks.VideoTrack.WriteSample(media.Sample{
			Data:     bytes.Repeat([]byte{0, 1, 2}, 100),
			Duration: 1 * time.Second,
		}); err != nil {
			return err
		}
		if err := tracks.AudioTrack.WriteSample(media.Sample{
			Data:     bytes.Repeat([]byte{0, 1, 2}, 100),
			Duration: 1 * time.Second,
		}); err != nil {
			return err
		}
	}

	return nil
}

func waitForSession(ctx context.Context, c proto.BrokerClient, gsName string) (*proto.Session, error) {
	ticker := time.NewTicker(1 * time.Second)
	for {
		select {
		case <-ctx.Done():
		case <-ticker.C:
			resp, err := c.FindSession(ctx, &proto.FindSessionRequest{GameserverName: gsName})
			if err != nil {
				return nil, err
			}
			if !resp.Found {
				continue
			}
			var metadata game.Metadata
			if err := yaml.Unmarshal([]byte(resp.Session.GameMetadata.Body), &metadata); err != nil {
				return nil, err
			}
			log.Printf("found session(sid: %s, gameId: %s)", resp.Session.SessionId, metadata.GameID)
			return resp.Session, nil
		}
	}
}

func waitForOffer(ctx context.Context, c proto.SignalingClient, sid gamesession.SessionID) (*webrtc.SessionDescription, error) {
	ticker := time.NewTicker(1 * time.Second)
	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-ticker.C:
			resp, err := c.ReceiveSignalingOffer(ctx, &proto.ReceiveSignalingOfferRequest{SessionId: string(sid)})
			if err != nil {
				return nil, err
			}
			if resp.Found {
				log.Printf("[agent] received offer")
				return webrtcutil.DecodeSDP(resp.Offer.Body)
			}
		}
	}
}
