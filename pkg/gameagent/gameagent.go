package gameagent

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/pkg/errors"

	"github.com/castaneai/mashimaro/pkg/streamer"

	"github.com/pion/webrtc/v3/pkg/media"

	"golang.org/x/sync/errgroup"

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
	// TODO: connection timed out
	<-connected

	log.Printf("[agent] connected!")

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

	videoStream, err := streamer.NewX11VideoStream(a.streamingConfig.XDisplay)
	if err != nil {
		return errors.Wrap(err, "failed to get x11 video stream")
	}
	defer videoStream.Close()
	audioStream, err := streamer.NewPulseAudioStream(a.streamingConfig.PulseAddr)
	if err != nil {
		return errors.Wrap(err, "failed to get pulse audio stream")
	}
	defer audioStream.Close()

	log.Printf("start streaming media")
	videoStream.Start()
	audioStream.Start()
	eg := &errgroup.Group{}
	eg.Go(func() error {
		if err := startStreamingMedia(ctx, mediaTracks.VideoTrack, videoStream); err != nil {
			return errors.Wrap(err, "failed to stream video")
		}
		return nil
	})
	eg.Go(func() error {
		if err := startStreamingMedia(ctx, mediaTracks.AudioTrack, audioStream); err != nil {
			return errors.Wrap(err, "failed to stream audio")
		}
		return nil
	})
	return eg.Wait()
}

func waitForSession(ctx context.Context, c proto.BrokerClient, gsName string) (*proto.Session, error) {
	ticker := time.NewTicker(1 * time.Second)
	for {
		select {
		case <-ctx.Done():
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

func startStreamingMedia(ctx context.Context, track *webrtc.TrackLocalStaticSample, stream streamer.MediaStream) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			chunk, err := stream.ReadChunk()
			if err != nil {
				return fmt.Errorf("failed to read chunk from stream: %+v", err)
			}
			if err := track.WriteSample(media.Sample{
				Data:     chunk.Data,
				Duration: chunk.Duration,
			}); err != nil {
				return fmt.Errorf("failed to write sample to track: %+v", err)
			}
		}
	}
}
