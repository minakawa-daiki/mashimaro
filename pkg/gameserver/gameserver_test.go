package gameserver

import (
	"context"
	"encoding/json"
	"log"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/castaneai/mashimaro/pkg/encoder"

	"github.com/castaneai/mashimaro/pkg/allocator"

	"github.com/pkg/errors"

	"github.com/BurntSushi/xgb/xproto"

	"github.com/castaneai/mashimaro/pkg/ayame"
	"github.com/castaneai/mashimaro/pkg/transport"
	"github.com/pion/webrtc/v3"

	"github.com/stretchr/testify/assert"

	"github.com/castaneai/mashimaro/pkg/gamemetadata"

	"github.com/castaneai/mashimaro/pkg/broker"
	"github.com/castaneai/mashimaro/pkg/gameprocess"
	"github.com/castaneai/mashimaro/pkg/gamesession"
	"github.com/castaneai/mashimaro/pkg/proto"
	"github.com/castaneai/mashimaro/pkg/testutils"
	"google.golang.org/grpc"
)

func newTestInternalBrokerClient(t *testing.T, sstore gamesession.Store, mstore gamemetadata.Store) proto.BrokerClient {
	lis := testutils.ListenTCPWithRandomPort(t)
	s := grpc.NewServer()
	proto.RegisterBrokerServer(s, broker.NewInternalBroker(sstore, mstore))
	go s.Serve(lis)
	cc, err := grpc.Dial(lis.Addr().String(), grpc.WithInsecure())
	if err != nil {
		t.Fatalf("failed to dial to internal broker: %+v", err)
	}
	return proto.NewBrokerClient(cc)
}

func newTestGameProcessClient(t *testing.T) proto.GameProcessClient {
	lis := testutils.ListenTCPWithRandomPort(t)
	s := grpc.NewServer()
	proto.RegisterGameProcessServer(s, gameprocess.NewGameProcessServer())
	go s.Serve(lis)
	cc, err := grpc.Dial(lis.Addr().String(), grpc.WithInsecure())
	if err != nil {
		t.Fatalf("failed to dial to game process: %+v", err)
	}
	return proto.NewGameProcessClient(cc)
}

func newTestEncoderClient(t *testing.T) proto.EncoderClient {
	lis := testutils.ListenTCPWithRandomPort(t)
	s := grpc.NewServer()
	proto.RegisterEncoderServer(s, encoder.NewEncoderServer())
	go s.Serve(lis)
	cc, err := grpc.Dial(lis.Addr().String(), grpc.WithInsecure())
	if err != nil {
		t.Fatalf("failed to dial to game process: %+v", err)
	}
	return proto.NewEncoderClient(cc)
}

func sendMoveMessage(t *testing.T, conn transport.PlayerConn, msg *MoveMessage) {
	bodyb, err := json.Marshal(msg)
	if err != nil {
		t.Fatal(err)
	}
	b, err := json.Marshal(&Message{Type: MessageTypeMove, Body: bodyb})
	if err != nil {
		t.Fatal(err)
	}
	if err := conn.SendMessage(context.Background(), b); err != nil {
		t.Fatal(err)
	}
}

func sendMouseMessage(t *testing.T, conn transport.PlayerConn, button xproto.Button, isDown bool) {
	mtype := MessageTypeMouseDown
	var msg interface{}
	if isDown {
		mtype = MessageTypeMouseDown
		msg = &MouseDownMessage{Button: int(button)}
	} else {
		mtype = MessageTypeMouseUp
		msg = MouseUpMessage{Button: int(button)}
	}
	bodyb, err := json.Marshal(msg)
	if err != nil {
		t.Fatal(err)
	}
	b, err := json.Marshal(&Message{Type: mtype, Body: bodyb})
	if err != nil {
		t.Fatal(err)
	}
	if err := conn.SendMessage(context.Background(), b); err != nil {
		t.Fatal(err)
	}
}

func sendKeyMessage(t *testing.T, conn transport.PlayerConn, key int, isDown bool) {
	mtype := MessageTypeKeyDown
	var msg interface{}
	if isDown {
		mtype = MessageTypeKeyDown
		msg = &KeyDownMessage{Key: key}
	} else {
		mtype = MessageTypeKeyUp
		msg = KeyUpMessage{Key: key}
	}
	bodyb, err := json.Marshal(msg)
	if err != nil {
		t.Fatal(err)
	}
	b, err := json.Marshal(&Message{Type: mtype, Body: bodyb})
	if err != nil {
		t.Fatal(err)
	}
	if err := conn.SendMessage(context.Background(), b); err != nil {
		t.Fatal(err)
	}
}

func sendExitGameMessage(t *testing.T, conn transport.PlayerConn) {
	b, err := json.Marshal(&Message{Type: MessageTypeExitGame})
	if err != nil {
		t.Fatal(err)
	}
	if err := conn.SendMessage(context.Background(), b); err != nil {
		t.Fatal(err)
	}
}

func TestGameServerLifecycle(t *testing.T) {
	ayameURL := os.Getenv("AYAME_URL")
	if ayameURL == "" {
		t.Skip("Set AYAME_URL to run this test")
	}

	ctx := context.Background()
	allocatedServer := &allocator.AllocatedServer{
		ID: "test-gs",
	}
	gameMetadata := &gamemetadata.Metadata{
		GameID:  "notepad",
		Command: "wine notepad",
	}
	sstore := gamesession.NewInMemoryStore()
	mstore := gamemetadata.NewInMemoryStore()
	err := mstore.AddGameMetadata(ctx, gameMetadata)
	assert.NoError(t, err)
	brokerClient := newTestInternalBrokerClient(t, sstore, mstore)
	gameProcessClient := newTestGameProcessClient(t)
	encoderClient := newTestEncoderClient(t)
	signaler := transport.NewAyameSignaler(ayameURL)
	ss, err := sstore.NewSession(ctx, &gamesession.NewSessionRequest{
		GameID:            gameMetadata.GameID,
		AllocatedServerID: allocatedServer.ID,
	})
	assert.NoError(t, err)
	gameServer := NewGameServer(allocatedServer, brokerClient, gameProcessClient, encoderClient, signaler)
	shutdown := make(chan struct{})
	gameServer.OnShutdown(func() {
		close(shutdown)
	})
	go func() {
		if err := gameServer.Serve(ctx); err != nil {
			log.Printf("failed to run game server: %+v", err)
		}
	}()

	conn, err := transport.NewWebRTCPlayerConn(webrtc.Configuration{})
	assert.NoError(t, err)
	connected := make(chan struct{})
	conn.OnConnect(func() {
		close(connected)
	})
	disconnected := make(chan struct{})
	var dOnce sync.Once
	conn.OnDisconnect(func() {
		dOnce.Do(func() {
			close(disconnected)
		})
	})
	ayamec := ayame.NewClient(conn.PeerConnection())
	err = ayamec.Connect(ctx, ayameURL, &ayame.ConnectRequest{
		RoomID:   string(ss.SessionID),
		ClientID: "player",
	})
	assert.NoError(t, err)
	<-connected
	// waiting for game process ready
	time.Sleep(1 * time.Second)

	for i := 0; i < 10; i++ {
		sendMoveMessage(t, conn, &MoveMessage{X: i * 10, Y: i * 10})
		time.Sleep(10 * time.Millisecond)
	}
	sendMouseMessage(t, conn, xproto.ButtonIndex3, true)
	sendMouseMessage(t, conn, xproto.ButtonIndex3, false)
	time.Sleep(1 * time.Second)
	sendExitGameMessage(t, conn)
	<-shutdown
	time.Sleep(100 * time.Millisecond)
	_, err = sstore.GetSession(ctx, ss.SessionID)
	assert.True(t, errors.Is(err, gamesession.ErrSessionNotFound))
}
