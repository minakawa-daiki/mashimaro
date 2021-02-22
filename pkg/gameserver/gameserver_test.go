package gameserver

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/url"
	"sync"
	"testing"
	"time"

	"github.com/castaneai/mashimaro/pkg/streamer/streamerserver"

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

func newInternalBrokerClient(t *testing.T, sstore gamesession.Store, mstore gamemetadata.Store) proto.BrokerClient {
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

func newGameProcessClient(t *testing.T) proto.GameProcessClient {
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

func newStreamerClient(t *testing.T) proto.StreamerClient {
	lis := testutils.ListenTCPWithRandomPort(t)
	s := grpc.NewServer()
	proto.RegisterStreamerServer(s, streamerserver.NewStreamerServer())
	go s.Serve(lis)
	cc, err := grpc.Dial(lis.Addr().String(), grpc.WithInsecure())
	if err != nil {
		t.Fatalf("failed to dial to game process: %+v", err)
	}
	return proto.NewStreamerClient(cc)
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
	checkAyame(t)

	ctx := context.Background()
	allocatedServer := &allocator.AllocatedServer{
		ID: "test-gs",
	}
	gameMetadata := &gamemetadata.Metadata{
		GameID:  "notepad",
		Command: "wine notepad",
	}
	sstore := gamesession.NewInMemoryStore()
	mstore := gamemetadata.NewInMemoryMetadataStore()
	err := mstore.AddGameMetadata(ctx, gameMetadata.GameID, gameMetadata)
	assert.NoError(t, err)
	brokerClient := newInternalBrokerClient(t, sstore, mstore)
	gameProcessClient := newGameProcessClient(t)
	streamerClient := newStreamerClient(t)
	signaler := transport.NewAyameSignaler(ayameURL)
	ss, err := sstore.NewSession(ctx, &gamesession.NewSessionRequest{
		GameID:            gameMetadata.GameID,
		AllocatedServerID: allocatedServer.ID,
	})
	assert.NoError(t, err)
	gameServer := NewGameServer(allocatedServer, brokerClient, gameProcessClient, streamerClient, signaler)
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

/*
func TestVideoOnBrowser(t *testing.T) {
	t.Skip("comment out this line if you want test video quality on browser(chromedriver required in your PATH)")
	checkAyame(t)

	ctx := context.Background()
	gameServer := &AllocatedServer{
		ID:   "test-gs",
	}
	gameMetadata := &gamemetadata.Metadata{
		GameID:  "notepad",
		Command: "wine notepad",
	}
	sstore := gamesession.NewInMemoryStore()
	mstore := gamemetadata.NewInMemoryMetadataStore()
	err := mstore.AddGameMetadata(ctx, gameMetadata.GameID, gameMetadata)
	assert.NoError(t, err)
	bc := newInternalBrokerClient(t, sstore, mstore)
	gwc := newGameProcessClient(t)
	signaler := NewAyameSignaler(ayameURL)

	display := os.Getenv("DISPLAY")
	ss, err := sstore.NewSession(ctx, &gamesession.NewSessionRequest{
		GameID:     gameMetadata.GameID,
		GameServer: gameServer,
	})
	assert.NoError(t, err)
	agent := NewGameServer(bc, gwc, signaler)
	agentExited := make(chan struct{})
	agent.OnShutdown(func() {
		close(agentExited)
	})
	log.Printf("%s", ss.SessionID)

	driver := agouti.ChromeDriver(agouti.ChromeOptions("args", []string{"--no-sandbox"}))
	assert.NoError(t, driver.Start())
	t.Cleanup(func() {
		_ = driver.Stop()
	})

	page, err := driver.NewPage()
	assert.NoError(t, err)
	wd, err := os.Getwd()
	assert.NoError(t, err)
	assert.NoError(t, page.Navigate(fmt.Sprintf("file://%s/test.html", wd)))
	var result string
	assert.NoError(t, page.RunScript(fmt.Sprintf("startConn('%s')", ss.SessionID), map[string]interface{}{}, &result))
	assert.NoError(t, page.Click(agouti.SingleClick, agouti.LeftButton)) // to avoid "play() failed because the user didn't interact with the document first"

	videoConfig := &streamer.VideoConfig{
		CaptureDisplay: display,
		CaptureArea:    streamer.ScreenCaptureArea{},
		X264Param:      "",
	}
	audioConfig := &streamer.AudioConfig{PulseServer: "localhost:4713"}
	go func() {
		if err := agent.Serve(ctx, gameServer.ID, videoConfig, audioConfig); err != nil {
			log.Printf("failed to run game agent: %+v", err)
		}
	}()
	select {}
}
*/
