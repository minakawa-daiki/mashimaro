package gameagent

import (
	"context"
	"fmt"
	"log"
	"strings"
	"sync"

	"github.com/castaneai/mashimaro/pkg/transport"

	"github.com/pkg/errors"

	"github.com/castaneai/mashimaro/pkg/streamer"

	"github.com/castaneai/mashimaro/pkg/gamesession"

	"github.com/pion/webrtc/v3"

	"github.com/castaneai/mashimaro/pkg/gamemetadata"
	"github.com/goccy/go-yaml"

	"github.com/castaneai/mashimaro/pkg/proto"
)

const (
	msgChBufferSize   = 50
	defaultX264Params = "speed-preset=ultrafast tune=zerolatency byte-stream=true intra-refresh=true"
)

var (
	defaultWebRTCConfiguration = webrtc.Configuration{
		ICEServers: []webrtc.ICEServer{
			{
				URLs: []string{"stun:stun.l.google.com:19302"},
			},
		},
	}
)

type Agent struct {
	brokerClient      proto.BrokerClient
	gameWrapperClient proto.GameWrapperClient
	signaler          Signaler
	onExit            func()
	callbackMu        sync.Mutex
}

func NewAgent(brokerClient proto.BrokerClient, gameWrapperClient proto.GameWrapperClient, signaler Signaler) *Agent {
	return &Agent{
		brokerClient:      brokerClient,
		gameWrapperClient: gameWrapperClient,
		signaler:          signaler,
	}
}

func (a *Agent) OnExit(f func()) {
	a.callbackMu.Lock()
	defer a.callbackMu.Unlock()
	a.onExit = f
}

func (a *Agent) Run(ctx context.Context, gsName string, videoConf *streamer.VideoConfig, audioConf *streamer.AudioConfig) error {
	defer a.handleExit()
	if videoConf.X264Param == "" {
		log.Printf("X264Param is empty, defaults to x264 params: %s", defaultX264Params)
		videoConf.X264Param = defaultX264Params
	}

	log.Printf("waiting for session...")
	ss, err := waitForSession(ctx, a.brokerClient, gsName)
	if err != nil {
		return err
	}
	sid := gamesession.SessionID(ss.SessionId)
	var metadata gamemetadata.Metadata
	if err := yaml.Unmarshal([]byte(ss.GameMetadata.Body), &metadata); err != nil {
		return err
	}
	log.Printf("game metadata loaded: %+v", metadata)

	// TODO: provisioning game data and ready to start process
	log.Printf("--- (TODO) provisioning game...")

	log.Printf("--- initializing connection...")
	conn, err := transport.NewWebRTCStreamerConn(defaultWebRTCConfiguration)
	if err != nil {
		return errors.Wrap(err, "failed to new webrtc streamer conn")
	}
	connected := make(chan struct{})
	conn.OnConnect(func() { close(connected) })
	disconnected := make(chan struct{})
	conn.OnDisconnect(func() { close(disconnected) })
	msgCh := make(chan []byte, msgChBufferSize)
	conn.OnMessage(func(data []byte) {
		msgCh <- data
	})

	log.Printf("start signaling...")
	if err := a.signaler.Signaling(ctx, conn, string(sid), "streamer"); err != nil {
		return err
	}

	// TODO: connection timed out
	log.Printf("waiting for connection...")
	<-connected
	log.Printf("connected!")

	if err := a.startGame(ctx, &metadata); err != nil {
		return err
	}

	errCh := make(chan error)
	captureAreaChanged := newCaptureAreaPubSub()
	go func() {
		captureAreaChanged.Start(ctx)
	}()
	go func() {
		errCh <- a.startStreaming(ctx, conn, videoConf, audioConf, captureAreaChanged.Subscribe())
	}()
	go func() {
		errCh <- a.startMessaging(ctx, videoConf.CaptureDisplay, msgCh, captureAreaChanged.Subscribe())
	}()
	go func() {
		errCh <- a.startWatchGame(ctx, captureAreaChanged)
	}()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case err := <-errCh:
			return err
		case <-disconnected:
			return fmt.Errorf("disconnected")
		}
	}
}

func (a *Agent) startGame(ctx context.Context, metadata *gamemetadata.Metadata) error {
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

func (a *Agent) handleExit() {
	a.callbackMu.Lock()
	onExit := a.onExit
	a.callbackMu.Unlock()
	if onExit != nil {
		onExit()
	}
}

type captureAreaPubSub struct {
	publishCh   chan *streamer.CaptureArea
	subscribeCh chan chan *streamer.CaptureArea
}

func newCaptureAreaPubSub() *captureAreaPubSub {
	return &captureAreaPubSub{
		publishCh:   make(chan *streamer.CaptureArea),
		subscribeCh: make(chan chan *streamer.CaptureArea),
	}
}

func (b *captureAreaPubSub) Start(ctx context.Context) {
	subscribers := make(map[chan *streamer.CaptureArea]struct{})
	for {
		select {
		case <-ctx.Done():
			return
		case area := <-b.publishCh:
			for sub := range subscribers {
				select {
				case sub <- area:
				default:
				}
			}
		case ch := <-b.subscribeCh:
			subscribers[ch] = struct{}{}
		}
	}
}

func (b *captureAreaPubSub) Subscribe() <-chan *streamer.CaptureArea {
	ch := make(chan *streamer.CaptureArea)
	b.subscribeCh <- ch
	return ch
}

func (b *captureAreaPubSub) Publish(area *streamer.CaptureArea) {
	b.publishCh <- area
}
