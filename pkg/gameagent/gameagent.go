package gameagent

import (
	"context"
	"log"
	"strings"
	"time"

	"github.com/pkg/errors"

	"github.com/castaneai/mashimaro/pkg/streamer"

	"github.com/castaneai/mashimaro/pkg/ayame"

	"github.com/castaneai/mashimaro/pkg/gamesession"

	"github.com/pion/webrtc/v3"

	"github.com/castaneai/mashimaro/pkg/game"
	"github.com/goccy/go-yaml"

	"github.com/castaneai/mashimaro/pkg/proto"
)

type Agent struct {
	brokerClient      proto.BrokerClient
	gameWrapperClient proto.GameWrapperClient
	signalingConfig   *SignalingConfig
	streamingConfig   *StreamingConfig
}

type SignalingConfig struct {
	AyameURL string
}

type StreamingConfig struct {
	XDisplay  string
	PulseAddr string
}

func NewAgent(brokerClient proto.BrokerClient, gameWrapperClient proto.GameWrapperClient, signalingConfig *SignalingConfig, streamingConfig *StreamingConfig) *Agent {
	return &Agent{
		brokerClient:      brokerClient,
		gameWrapperClient: gameWrapperClient,
		signalingConfig:   signalingConfig,
		streamingConfig:   streamingConfig,
	}
}

func (a *Agent) Run(ctx context.Context, gsName string, mediaTracks *MediaTracks) error {
	log.Printf("waiting for session...")
	ss, err := waitForSession(ctx, a.brokerClient, gsName)
	if err != nil {
		return err
	}
	sid := gamesession.SessionID(ss.SessionId)
	var metadata game.Metadata
	if err := yaml.Unmarshal([]byte(ss.GameMetadata.Body), &metadata); err != nil {
		return err
	}
	log.Printf("[agent] load metadata: %+v", metadata)

	// TODO: provisioning game data and ready to start process
	log.Printf("--- (TODO) provisioning game...")

	dataChannelCh := make(chan *webrtc.DataChannel)
	dataChannelMessageCh := make(chan webrtc.DataChannelMessage)
	ayamec := ayame.NewClient(ayame.WithInitPeerConnection(func(pc *webrtc.PeerConnection) error {
		pc.OnDataChannel(func(dc *webrtc.DataChannel) {
			dc.OnOpen(func() {
				dataChannelCh <- dc
			})
			dc.OnError(func(err error) {
				log.Printf("data channel error: %+v", err)
			})
			dc.OnClose(func() {
				log.Printf("data channel closed")
			})
			dc.OnMessage(func(msg webrtc.DataChannelMessage) {
				dataChannelMessageCh <- msg
			})
		})
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
	// TODO: connection timed out
	<-connected
	log.Printf("[agent] connected!")

	if err := a.startGame(ctx, &metadata); err != nil {
		return err
	}

	dataChannel := <-dataChannelCh
	log.Printf("dataChannel %s-%d opened", dataChannel.Label(), dataChannel.ID())

	videoStream, err := streamer.NewX11VideoStream(a.streamingConfig.XDisplay)
	if err != nil {
		return errors.Wrap(err, "failed to get x11 video stream")
	}
	audioStream, err := streamer.NewPulseAudioStream(a.streamingConfig.PulseAddr)
	if err != nil {
		return errors.Wrap(err, "failed to get pulse audio stream")
	}

	streamingErr := make(chan error)
	go func() {
		log.Printf("start streaming media")
		streamingErr <- startStreaming(ctx, mediaTracks.VideoTrack, mediaTracks.AudioTrack, videoStream, audioStream)
	}()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case err := <-streamingErr:
			return errors.Wrap(err, "failed to streaming media")
		case msg := <-dataChannelMessageCh:
			log.Printf("msg received: %+v", string(msg.Data))
		}
	}
}

func (a *Agent) startGame(ctx context.Context, metadata *game.Metadata) error {
	cmds := strings.Split(metadata.Command, " ")
	var args []string
	if len(cmds) > 1 {
		args = cmds[1:]
	}
	if _, err := a.gameWrapperClient.StartGame(ctx, &proto.StartGameRequest{
		Command: cmds[0],
		Args:    args,
	}); err != nil {
		return errors.Wrap(err, "failed to start game")
	}
	return nil
}

func waitForSession(ctx context.Context, c proto.BrokerClient, gsName string) (*proto.Session, error) {
	ticker := time.NewTicker(1 * time.Second)
	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-ticker.C:
			resp, err := c.FindSession(ctx, &proto.FindSessionRequest{GameserverName: gsName})
			if err != nil {
				return nil, errors.Wrap(err, "failed to find session by broker")
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
