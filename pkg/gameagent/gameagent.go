package gameagent

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/castaneai/mashimaro/pkg/xorg"

	"github.com/castaneai/mashimaro/pkg/messaging"

	"github.com/castaneai/mashimaro/pkg/transport"

	"github.com/pkg/errors"

	"github.com/castaneai/mashimaro/pkg/streamer"

	"github.com/castaneai/mashimaro/pkg/gamesession"

	"github.com/pion/webrtc/v3"

	"github.com/castaneai/mashimaro/pkg/game"
	"github.com/goccy/go-yaml"

	"github.com/castaneai/mashimaro/pkg/proto"
)

const (
	msgChBufferSize = 50
)

var (
	ErrGameExited = errors.New("game exited")
)

type Agent struct {
	brokerClient      proto.BrokerClient
	gameWrapperClient proto.GameWrapperClient
	signaler          Signaler
	streamingConfig   *StreamingConfig

	onExit     func()
	callbackMu sync.Mutex
}

type StreamingConfig struct {
	XDisplay  string
	PulseAddr string
}

func NewAgent(brokerClient proto.BrokerClient, gameWrapperClient proto.GameWrapperClient, signaler Signaler, streamingConfig *StreamingConfig) *Agent {
	return &Agent{
		brokerClient:      brokerClient,
		gameWrapperClient: gameWrapperClient,
		signaler:          signaler,
		streamingConfig:   streamingConfig,
	}
}

func (a *Agent) OnExit(f func()) {
	a.callbackMu.Lock()
	defer a.callbackMu.Unlock()
	a.onExit = f
}

func (a *Agent) Run(ctx context.Context, gsName string) error {
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
	xorg.Display(a.streamingConfig.XDisplay)
	log.Printf("--- (TODO) provisioning game...")

	conn, err := transport.NewWebRTCStreamerConn(webrtc.Configuration{})
	if err != nil {
		return errors.Wrap(err, "failed to new webrtc streamer conn")
	}
	connected := make(chan struct{})
	conn.OnConnect(func() {
		close(connected)
	})
	msgCh := make(chan []byte, msgChBufferSize)
	conn.OnMessage(func(data []byte) {
		msgCh <- data
	})

	log.Printf("[agent] start signaling...")
	if err := a.signaler.Signaling(ctx, conn, string(sid), "streamer"); err != nil {
		return err
	}

	// TODO: connection timed out
	log.Printf("[agent] waiting for connection...")
	<-connected
	log.Printf("[agent] connected!")

	if err := a.startGame(ctx, &metadata); err != nil {
		return err
	}

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
		streamingErr <- startStreaming(ctx, conn, videoStream, audioStream)
	}()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case err := <-streamingErr:
			return errors.Wrap(err, "failed to streaming media")
		case msg := <-msgCh:
			if err := a.handleMessage(ctx, msg); err != nil {
				if errors.Is(err, ErrGameExited) {
					a.handleExit()
					return nil
				}
				log.Printf("failed to handle message: %+v", err)
			}
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

func (a *Agent) handleMessage(ctx context.Context, data []byte) error {
	var msg messaging.Message
	if err := json.Unmarshal(data, &msg); err != nil {
		return err
	}
	switch msg.Type {
	case messaging.MessageTypeMove:
		var body messaging.MoveMessage
		if err := json.Unmarshal(msg.Body, &body); err != nil {
			return err
		}
		xorg.Move(body.X, body.Y)
		return nil
	case messaging.MessageTypeExitGame:
		if _, err := a.gameWrapperClient.ExitGame(ctx, &proto.ExitGameRequest{}); err != nil {
			return err
		}
		return ErrGameExited
	default:
		return fmt.Errorf("unknown message type: %s", msg.Type)
	}
}

func (a *Agent) handleExit() {
	a.callbackMu.Lock()
	h := a.onExit
	a.callbackMu.Unlock()
	if h != nil {
		h()
	}
}
