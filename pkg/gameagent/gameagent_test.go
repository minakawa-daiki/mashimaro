package gameagent_test

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/url"
	"testing"
	"time"

	"github.com/castaneai/mashimaro/pkg/xorg"

	"github.com/castaneai/mashimaro/pkg/messaging"

	"github.com/castaneai/mashimaro/pkg/ayame"
	"github.com/castaneai/mashimaro/pkg/transport"
	"github.com/pion/webrtc/v3"

	"github.com/castaneai/mashimaro/pkg/gameserver"

	"github.com/stretchr/testify/assert"

	"github.com/castaneai/mashimaro/pkg/game"

	"github.com/castaneai/mashimaro/pkg/broker"
	"github.com/castaneai/mashimaro/pkg/gameagent"
	"github.com/castaneai/mashimaro/pkg/gamesession"
	"github.com/castaneai/mashimaro/pkg/gamewrapper"
	"github.com/castaneai/mashimaro/pkg/proto"
	"github.com/castaneai/mashimaro/pkg/testutils"
	"google.golang.org/grpc"
)

const (
	ayameURL = "ws://localhost:3000/signaling"
)

func checkAyame(t *testing.T) {
	u, err := url.Parse(ayameURL)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := net.DialTimeout("tcp", u.Host, 100*time.Millisecond); err != nil {
		t.Skip(fmt.Sprintf("A Test was skipped. Make sure that Ayame is running on %s", ayameURL))
	}
}

func newInternalBrokerClient(t *testing.T, sstore gamesession.Store, mstore game.MetadataStore) proto.BrokerClient {
	lis := testutils.ListenTCPWithRandomPort(t)
	s := grpc.NewServer()
	proto.RegisterBrokerServer(s, broker.NewInternalServer(sstore, mstore))
	go s.Serve(lis)
	cc, err := grpc.Dial(lis.Addr().String(), grpc.WithInsecure())
	if err != nil {
		t.Fatalf("failed to dial to internal broker: %+v", err)
	}
	return proto.NewBrokerClient(cc)
}

func newGameWrapperClient(t *testing.T) proto.GameWrapperClient {
	lis := testutils.ListenTCPWithRandomPort(t)
	s := grpc.NewServer()
	proto.RegisterGameWrapperServer(s, gamewrapper.NewGameWrapperServer())
	go s.Serve(lis)
	cc, err := grpc.Dial(lis.Addr().String(), grpc.WithInsecure())
	if err != nil {
		t.Fatalf("failed to dial to game wrapper: %+v", err)
	}
	return proto.NewGameWrapperClient(cc)
}

func sendMoveMessage(t *testing.T, conn transport.PlayerConn, msg *messaging.MoveMessage) {
	bodyb, err := json.Marshal(msg)
	if err != nil {
		t.Fatal(err)
	}
	b, err := json.Marshal(&messaging.Message{Type: messaging.MessageTypeMove, Body: bodyb})
	if err != nil {
		t.Fatal(err)
	}
	if err := conn.SendMessage(context.Background(), b); err != nil {
		t.Fatal(err)
	}
}

func sendMouseMessage(t *testing.T, conn transport.PlayerConn, button xorg.XButtonCode, isDown bool) {
	mtype := messaging.MessageTypeMouseDown
	var msg interface{}
	if isDown {
		mtype = messaging.MessageTypeMouseDown
		msg = &messaging.MouseDownMessage{Button: int(button)}
	} else {
		mtype = messaging.MessageTypeMouseUp
		msg = messaging.MouseUpMessage{Button: int(button)}
	}
	bodyb, err := json.Marshal(msg)
	if err != nil {
		t.Fatal(err)
	}
	b, err := json.Marshal(&messaging.Message{Type: mtype, Body: bodyb})
	if err != nil {
		t.Fatal(err)
	}
	if err := conn.SendMessage(context.Background(), b); err != nil {
		t.Fatal(err)
	}
}

func sendKeyMessage(t *testing.T, conn transport.PlayerConn, key int, isDown bool) {
	mtype := messaging.MessageTypeKeyDown
	var msg interface{}
	if isDown {
		mtype = messaging.MessageTypeKeyDown
		msg = &messaging.KeyDownMessage{Key: key}
	} else {
		mtype = messaging.MessageTypeKeyUp
		msg = messaging.KeyUpMessage{Key: key}
	}
	bodyb, err := json.Marshal(msg)
	if err != nil {
		t.Fatal(err)
	}
	b, err := json.Marshal(&messaging.Message{Type: mtype, Body: bodyb})
	if err != nil {
		t.Fatal(err)
	}
	if err := conn.SendMessage(context.Background(), b); err != nil {
		t.Fatal(err)
	}
}

func sendExitGameMessage(t *testing.T, conn transport.PlayerConn) {
	b, err := json.Marshal(&messaging.Message{Type: messaging.MessageTypeExitGame})
	if err != nil {
		t.Fatal(err)
	}
	if err := conn.SendMessage(context.Background(), b); err != nil {
		t.Fatal(err)
	}
}

func TestAgent(t *testing.T) {
	checkAyame(t)

	ctx := context.Background()
	gameServer := &gameserver.GameServer{
		Name: "test-gs",
		Addr: "dummy",
	}
	gameMetadata := &game.Metadata{
		GameID:  "notepad",
		Command: "wine notepad",
	}
	sstore := gamesession.NewInMemoryStore()
	mstore := game.NewMockMetadataStore()
	err := mstore.AddGameMetadata(ctx, gameMetadata.GameID, gameMetadata)
	assert.NoError(t, err)
	bc := newInternalBrokerClient(t, sstore, mstore)
	gwc := newGameWrapperClient(t)
	signaler := gameagent.NewAyameSignaler(ayameURL)
	streamingConfig := &gameagent.StreamingConfig{
		XDisplay:  ":0",
		PulseAddr: "localhost:4713",
	}
	ss, err := sstore.NewSession(ctx, &gamesession.NewSessionRequest{
		GameID:     gameMetadata.GameID,
		GameServer: gameServer,
	})
	assert.NoError(t, err)
	agent := gameagent.NewAgent(bc, gwc, signaler, streamingConfig)
	agentExited := make(chan struct{})
	agent.OnExit(func() {
		close(agentExited)
	})
	go func() {
		if err := agent.Run(ctx, gameServer.Name); err != nil {
			log.Printf("failed to run game agent: %+v", err)
		}
	}()

	conn, err := transport.NewWebRTCPlayerConn(webrtc.Configuration{})
	assert.NoError(t, err)
	connected := make(chan struct{})
	conn.OnConnect(func() {
		close(connected)
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
		sendMoveMessage(t, conn, &messaging.MoveMessage{X: i * 10, Y: i * 10})
		time.Sleep(10 * time.Millisecond)
	}
	sendMouseMessage(t, conn, xorg.XButtonCodeLeft, true)
	sendMouseMessage(t, conn, xorg.XButtonCodeLeft, false)
	for _, c := range "Hello,World." {
		sendKeyMessage(t, conn, int(c), true)
		sendKeyMessage(t, conn, int(c), false)
		time.Sleep(10 * time.Millisecond)
	}
	time.Sleep(1 * time.Second)
	sendExitGameMessage(t, conn)
	<-agentExited
}
