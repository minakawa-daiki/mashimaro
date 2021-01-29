package gameagent

import (
	"bytes"
	"context"
	"log"
	"time"

	"github.com/castaneai/mashimaro/pkg/ayame"

	"github.com/pion/webrtc/v3/pkg/media"

	"github.com/castaneai/mashimaro/pkg/gamesession"

	"github.com/pion/webrtc/v3"

	"github.com/castaneai/mashimaro/pkg/game"
	"github.com/goccy/go-yaml"

	"github.com/castaneai/mashimaro/pkg/proto"
)

type Agent struct {
	brokerClient    proto.BrokerClient
	signalingConfig *SignalingConfig
}

type SignalingConfig struct {
	AyameURL string
}

func NewAgent(brokerClient proto.BrokerClient, sc *SignalingConfig) *Agent {
	return &Agent{brokerClient: brokerClient, signalingConfig: sc}
}

func (a *Agent) Run(ctx context.Context, gsName string, mediaTracks *MediaTracks) error {
	ss, err := waitForSession(ctx, a.brokerClient, gsName)
	if err != nil {
		return err
	}
	sid := gamesession.SessionID(ss.SessionId)

	// TODO: provisioning game data and ready to start process

	ayamec := ayame.NewClient(ayame.WithInitPeerConnection(func(pc *webrtc.PeerConnection) error {
		if _, err := pc.AddTrack(mediaTracks.VideoTrack); err != nil {
			return err
		}
		if _, err := pc.AddTrack(mediaTracks.AudioTrack); err != nil {
			return err
		}
		return nil
	}))

	connected := make(chan struct{})
	ayamec.OnConnect(func() {
		close(connected)
	})
	if err := ayamec.Connect(ctx, a.signalingConfig.AyameURL, &ayame.ConnectRequest{RoomID: string(sid), ClientID: "streamer"}); err != nil {
		return err
	}
	<-connected

	log.Printf("[agent] connected!")
	ticker := time.NewTicker(1 * time.Second)
	for range ticker.C {
		if err := mediaTracks.VideoTrack.WriteSample(media.Sample{
			Data:     bytes.Repeat([]byte{0, 1, 2}, 100),
			Duration: 1 * time.Second,
		}); err != nil {
			return err
		}
		if err := mediaTracks.AudioTrack.WriteSample(media.Sample{
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
