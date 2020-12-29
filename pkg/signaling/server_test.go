package signaling

import (
	"context"
	"io"
	"log"
	"net"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/castaneai/mashimaro/pkg/gameserver"

	"github.com/castaneai/mashimaro/pkg/gamesession"
	"github.com/castaneai/mashimaro/pkg/proto"
	"google.golang.org/grpc"

	"golang.org/x/net/websocket"

	"github.com/stretchr/testify/assert"

	"github.com/pion/webrtc/v3"
)

type mockGameServerService struct {
	iceChannels *gameserver.ICEChannels
}

func newMockGameServerService() *mockGameServerService {
	return &mockGameServerService{
		iceChannels: gameserver.NewICEChannels(),
	}
}

func (s *mockGameServerService) FirstSignaling(ctx context.Context, req *proto.Offer) (*proto.Answer, error) {
	offer, err := decodeSDP(req.Body)
	if err != nil {
		return nil, err
	}
	answer, err := gameserver.StartSignaling(ctx, *offer, s.iceChannels)
	if err != nil {
		return nil, err
	}
	answerBody, err := encodeSDP(answer)
	if err != nil {
		return nil, err
	}
	return &proto.Answer{Body: answerBody}, nil
}

func (s *mockGameServerService) TrickleSignaling(stream proto.GameServer_TrickleSignalingServer) error {
	go func() {
		for {
			select {
			case <-stream.Context().Done():
				return
			case candidate := <-s.iceChannels.AnswerCandidate:
				if err := stream.Send(&proto.ICECandidate{Body: encodeICECandidate(candidate)}); err != nil {
					log.Printf("failed to send ice candidate from pcAnswer to pcOffer: %+v", err)
				}
			}
		}
	}()
	for {
		recv, err := stream.Recv()
		if err != nil {
			return err
		}
		candidate := decodeICECandidate(recv.Body)
		s.iceChannels.OfferCandidate <- candidate
	}
}

func TestSignaling(t *testing.T) {
	gss := newMockGameServerService()
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

	connected := make(chan struct{})
	pcOffer.OnICEConnectionStateChange(func(state webrtc.ICEConnectionState) {
		log.Printf("[pcOffer] ICE state has changed: %s", state)
	})
	pcOffer.OnConnectionStateChange(func(state webrtc.PeerConnectionState) {
		log.Printf("[pcOffer] connection state has changed: %s", state)
		if state == webrtc.PeerConnectionStateConnected {
			close(connected)
		}
	})

	pendingCandidateCh := make(chan webrtc.ICECandidateInit, 10)
	pcOffer.OnICECandidate(func(candidate *webrtc.ICECandidate) {
		log.Printf("[pcOffer] new ICE candidate: %v", candidate)
		if candidate == nil {
			close(pendingCandidateCh)
			return
		}
		if err := websocket.JSON.Send(ws, &message{Operation: OperationICECandidate, SessionID: sid, Body: encodeICECandidate(candidate.ToJSON())}); err != nil {
			log.Printf("failed to send ICE candidate from pcOffer: %+v", err)
		}
	})

	answerCh := make(chan webrtc.SessionDescription)
	go func() {
		for {
			var msg message
			if err := websocket.JSON.Receive(ws, &msg); err != nil {
				if err != io.EOF {
					log.Printf("failed to receive from server: %+v", err)
				}
				return
			}
			switch msg.Operation {
			case OperationAnswer:
				answer, err := decodeSDP(msg.Body)
				if err != nil {
					log.Printf("failed to decode answer SDP: %+v", err)
					return
				}
				answerCh <- *answer
			case OperationICECandidate:
				candidate := decodeICECandidate(msg.Body)
				if pcOffer.RemoteDescription() != nil {
					if err := pcOffer.AddICECandidate(candidate); err != nil {
						log.Printf("[offer] failed to add candidate: %+v", err)
					}
				} else {
					pendingCandidateCh <- candidate
				}
			}
		}
	}()

	offer, err := pcOffer.CreateOffer(nil)
	assert.NoError(t, err)
	assert.NoError(t, pcOffer.SetLocalDescription(offer))
	body, err := encodeSDP(&offer)
	assert.NoError(t, err)
	assert.NoError(t, websocket.JSON.Send(ws, &message{Operation: OperationOffer, SessionID: sid, Body: body}))

	answer := <-answerCh
	assert.NoError(t, pcOffer.SetRemoteDescription(answer))
	log.Printf("[pcOffer] got answer")
	for candidate := range pendingCandidateCh {
		assert.NoError(t, pcOffer.AddICECandidate(candidate))
	}

	<-connected
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
