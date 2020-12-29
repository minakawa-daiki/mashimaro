package signaling

import (
	"context"
	"log"
	"net"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/castaneai/mashimaro/pkg/gamesession"
	"github.com/castaneai/mashimaro/pkg/proto"
	"google.golang.org/grpc"

	"golang.org/x/net/websocket"

	"github.com/stretchr/testify/assert"

	"github.com/pion/webrtc/v3"
)

type mockGameServerService struct {
}

func (s *mockGameServerService) FirstSignaling(ctx context.Context, req *proto.Offer) (*proto.Answer, error) {
	offer, err := decodeSDP(req.Body)
	if err != nil {
		return nil, err
	}
	answer, _, err := startSignaling(ctx, *offer)
	if err != nil {
		return nil, err
	}
	answerBody, err := encodeSDP(answer)
	if err != nil {
		return nil, err
	}
	return &proto.Answer{Body: answerBody}, nil
}

func (s *mockGameServerService) TrickleSignaling(server proto.GameServer_TrickleSignalingServer) error {
	panic("implement me")
}

func TestSignaling(t *testing.T) {
	gss := &mockGameServerService{}
	gs := grpc.NewServer()
	proto.RegisterGameServerServer(gs, gss)
	lis := listenTCPWithRandomPort(t)
	go func() {
		if err := gs.Serve(lis); err != nil {
			log.Printf("failed to serve gRPC server: %+v", err)
		}
	}()

	allocator := &gamesession.MockAllocator{MockedGS: &gamesession.GameServer{Addr: lis.Addr().String()}}
	gsManager := gamesession.NewManager(allocator)
	sv := newTestServer(gsManager)
	ws := sv.DialWebSocket(t)

	assert.NoError(t, websocket.JSON.Send(ws, &message{Operation: OperationNewGame}))
	var newGameRes message
	assert.NoError(t, websocket.JSON.Receive(ws, &newGameRes))
	assert.Equal(t, OperationNewGame, newGameRes.Operation)
	sid := gamesession.SessionID(newGameRes.Body)

	pcOffer, err := webrtc.NewPeerConnection(webrtc.Configuration{
		ICEServers: []webrtc.ICEServer{
			{URLs: []string{"stun:stun.l.google.com:19302"}},
		},
	})
	assert.NoError(t, err)
	_, err = pcOffer.AddTransceiverFromKind(webrtc.RTPCodecTypeVideo, webrtc.RTPTransceiverInit{Direction: webrtc.RTPTransceiverDirectionRecvonly})
	assert.NoError(t, err)
	_, err = pcOffer.AddTransceiverFromKind(webrtc.RTPCodecTypeAudio, webrtc.RTPTransceiverInit{Direction: webrtc.RTPTransceiverDirectionRecvonly})
	assert.NoError(t, err)

	offer, err := pcOffer.CreateOffer(nil)
	assert.NoError(t, err)
	assert.NoError(t, pcOffer.SetLocalDescription(offer))
	body, err := encodeSDP(&offer)
	assert.NoError(t, err)
	assert.NoError(t, websocket.JSON.Send(ws, &message{Operation: OperationOffer, SessionID: sid, Body: body}))

	var offerRes message
	assert.NoError(t, websocket.JSON.Receive(ws, &offerRes))
	assert.Equal(t, OperationAnswer, offerRes.Operation)
	answer, err := decodeSDP(offerRes.Body)
	assert.NoError(t, err)
	assert.NoError(t, pcOffer.SetRemoteDescription(*answer))
}

func listenTCPWithRandomPort(t *testing.T) net.Listener {
	t.Helper()
	taddr, err := net.ResolveTCPAddr("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to resolve TCP addr: %+v", err)
	}
	lis, err := net.Listen("tcp", taddr.String())
	if err != nil {
		t.Fatalf("failed to listen TCP on %v: %+v", taddr.String(), err)
	}
	return lis
}

type testServer struct {
	*Server
	hs *httptest.Server
}

func (ts *testServer) DialWebSocket(t *testing.T) *websocket.Conn {
	wsURL := strings.Replace(ts.hs.URL, "http:", "ws:", 1)
	conn, err := websocket.Dial(wsURL, "", ts.hs.URL)
	if err != nil {
		t.Fatal(err)
	}
	return conn
}

func newTestServer(gsManager *gamesession.Manager) *testServer {
	s := NewServer(gsManager)
	hs := httptest.NewServer(s.webSocketServer())
	return &testServer{
		Server: s,
		hs:     hs,
	}
}
