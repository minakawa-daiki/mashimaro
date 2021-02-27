package gameserver

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/castaneai/mashimaro/pkg/gamemetadata"

	"github.com/castaneai/mashimaro/pkg/allocator"

	"github.com/castaneai/mashimaro/pkg/transport"

	"github.com/pkg/errors"

	"github.com/castaneai/mashimaro/pkg/gamesession"

	"github.com/pion/webrtc/v3"

	"github.com/castaneai/mashimaro/pkg/proto"
)

const (
	receivedMessageBufferSize = 50
	connectTimeout            = 10 * time.Second
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

type GameServer struct {
	allocatedServer *allocator.AllocatedServer
	broker          proto.BrokerClient
	gameProcess     proto.GameProcessClient
	encoder         proto.EncoderClient
	signaler        transport.WebRTCSignaler
	onShutdown      func()
	callbackMu      sync.Mutex
}

func NewGameServer(allocatedServer *allocator.AllocatedServer, broker proto.BrokerClient, gameProcess proto.GameProcessClient, encoder proto.EncoderClient, signaler transport.WebRTCSignaler) *GameServer {
	return &GameServer{
		allocatedServer: allocatedServer,
		broker:          broker,
		gameProcess:     gameProcess,
		encoder:         encoder,
		signaler:        signaler,
	}
}

func (s *GameServer) OnShutdown(f func()) {
	s.callbackMu.Lock()
	defer s.callbackMu.Unlock()
	s.onShutdown = f
}

func (s *GameServer) Serve(ctx context.Context) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	defer s.shutdown()

	errCh := make(chan error)
	sessionCreated := make(chan *gamesession.Session)
	sessionDeleted := make(chan struct{})
	go func() {
		errCh <- s.startWatchSession(ctx, sessionCreated, sessionDeleted)
	}()

	log.Printf("waiting for new session for allocated server: %v", s.allocatedServer)
	var session *gamesession.Session
	select {
	case <-ctx.Done():
		return ctx.Err()
	case err := <-errCh:
		return fmt.Errorf("error occured while waiting for new session: %+v", err)
	case session = <-sessionCreated:
	}
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		log.Printf("request delete session")
		if _, err := s.broker.DeleteSession(ctx, &proto.DeleteSessionRequest{SessionId: string(session.SessionID), AllocatedServerId: s.allocatedServer.ID}); err != nil {
			log.Printf("failed to delete session: %+v", err)
		}
	}()

	log.Printf("--- initializing connection...")
	conn, err := transport.NewWebRTCStreamerConn(defaultWebRTCConfiguration)
	if err != nil {
		return errors.Wrap(err, "failed to new webrtc streamer conn")
	}
	// TODO: reconnect
	connected := make(chan struct{})
	conn.OnConnect(func() { close(connected) })
	disconnected := make(chan struct{})
	var dOnce sync.Once
	conn.OnDisconnect(func() {
		dOnce.Do(func() {
			close(disconnected)
		})
	})
	messageReceived := make(chan []byte, receivedMessageBufferSize)
	conn.OnMessage(func(data []byte) {
		messageReceived <- data
	})

	roomID := string(session.SessionID)
	connector := transport.NewWebRTCConnector(s.signaler, roomID, "streamer")
	if err := connector.Connect(ctx, conn); err != nil {
		return err
	}

	log.Printf("waiting for connection...")
	select {
	case err := <-errCh:
		return fmt.Errorf("error occured while waiting for connection established: %+v", err)
	case <-connected:
	case <-time.After(connectTimeout):
		return fmt.Errorf("connection timed out")
	}
	log.Printf("connected!")

	log.Printf("start game process")
	if err := s.startGame(ctx, session.GameID); err != nil {
		return err
	}
	defer func() {
		log.Printf("clean-up game process")
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second) // TODO: from config
		defer cancel()
		if _, err := s.gameProcess.ExitGame(ctx, &proto.ExitGameRequest{}); err != nil {
			log.Printf("failed to exit game request: %+v", err)
		}
	}()

	captureRectChanged := newCaptureRectPubSub()
	go func() { captureRectChanged.Start(ctx) }()
	go func() { errCh <- s.startStreaming(ctx, conn, captureRectChanged.Subscribe()) }()
	go func() { errCh <- s.startController(ctx, messageReceived, captureRectChanged.Subscribe()) }()
	go func() { errCh <- s.startWatchGame(ctx, captureRectChanged) }()
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case err := <-errCh:
			return err
		case <-disconnected:
			return fmt.Errorf("disconnected")
		case <-sessionDeleted:
			return fmt.Errorf("session deleted")
		}
	}
}

func (s *GameServer) startGame(ctx context.Context, gameID string) error {
	resp, err := s.broker.GetGameMetadata(ctx, &proto.GetGameMetadataRequest{GameId: gameID})
	if err != nil {
		return err
	}
	var metadata gamemetadata.Metadata
	if err := gamemetadata.Unmarshal([]byte(resp.GameMetadata.Body), &metadata); err != nil {
		return err
	}
	cmd, args, err := metadata.ParseCommand()
	if err != nil {
		return err
	}
	if _, err := s.gameProcess.StartGame(ctx, &proto.StartGameRequest{
		Command:          cmd,
		Args:             args,
		WorkingDirectory: "", // TODO: workdir
	}); err != nil {
		return err
	}
	return nil
}

func (s *GameServer) shutdown() {
	s.callbackMu.Lock()
	onExit := s.onShutdown
	s.callbackMu.Unlock()
	if onExit != nil {
		onExit()
	}
}

type captureRectPubSub struct {
	publishCh   chan ScreenRect
	subscribeCh chan chan ScreenRect
}

func newCaptureRectPubSub() *captureRectPubSub {
	return &captureRectPubSub{
		publishCh:   make(chan ScreenRect),
		subscribeCh: make(chan chan ScreenRect),
	}
}

func (b *captureRectPubSub) Start(ctx context.Context) {
	subscribers := make(map[chan ScreenRect]struct{})
	for {
		select {
		case <-ctx.Done():
			return
		case rect := <-b.publishCh:
			for sub := range subscribers {
				select {
				case sub <- rect:
				default:
				}
			}
		case ch := <-b.subscribeCh:
			subscribers[ch] = struct{}{}
		}
	}
}

func (b *captureRectPubSub) Subscribe() <-chan ScreenRect {
	ch := make(chan ScreenRect)
	b.subscribeCh <- ch
	return ch
}

func (b *captureRectPubSub) Publish(rect ScreenRect) {
	b.publishCh <- rect
}
