package signaling

import (
	"io"
	"log"
	"net"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/castaneai/mashimaro/pkg/streamer"

	"github.com/castaneai/mashimaro/pkg/gameserver"

	"github.com/castaneai/mashimaro/pkg/internal/webrtcutil"

	"github.com/castaneai/mashimaro/pkg/gamesession"
	"github.com/castaneai/mashimaro/pkg/proto"
	"google.golang.org/grpc"

	"golang.org/x/net/websocket"

	"github.com/stretchr/testify/assert"

	"github.com/pion/webrtc/v3"
)

func TestSignaling(t *testing.T) {
	videoSrc, err := streamer.NewVideoTestStream()
	assert.NoError(t, err)
	audioSrc, err := streamer.NewAudioTestStream()
	assert.NoError(t, err)
	gss := gameserver.NewGameServerService(videoSrc, audioSrc)
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
		candidateInit := candidate.ToJSON()
		body, err := webrtcutil.EncodeICECandidate(&candidateInit)
		if err != nil {
			log.Printf("failed to encode ICE candidate: %+v", err)
			return
		}
		if err := websocket.JSON.Send(ws, &message{Operation: OperationICECandidate, SessionID: sid, Body: body}); err != nil {
			log.Printf("failed to send ICE candidate from pcOffer: %+v", err)
		}
	})

	videoTrackCh := make(chan *webrtc.TrackRemote)
	audioTrackCh := make(chan *webrtc.TrackRemote)
	pcOffer.OnTrack(func(remote *webrtc.TrackRemote, receiver *webrtc.RTPReceiver) {
		switch remote.Kind() {
		case webrtc.RTPCodecTypeAudio:
			audioTrackCh <- remote
		case webrtc.RTPCodecTypeVideo:
			videoTrackCh <- remote
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
				answer, err := webrtcutil.DecodeSDP(msg.Body)
				if err != nil {
					log.Printf("failed to decode answer SDP: %+v", err)
					return
				}
				answerCh <- *answer
			case OperationICECandidate:
				candidate, err := webrtcutil.DecodeICECandidate(msg.Body)
				if err != nil {
					log.Printf("[offer] failed to decode ICE candidate: %+v", err)
					return
				}
				if pcOffer.RemoteDescription() != nil {
					if err := pcOffer.AddICECandidate(*candidate); err != nil {
						log.Printf("[offer] failed to add candidate: %+v", err)
					}
				} else {
					pendingCandidateCh <- *candidate
				}
			}
		}
	}()

	offer, err := pcOffer.CreateOffer(nil)
	assert.NoError(t, err)
	assert.NoError(t, pcOffer.SetLocalDescription(offer))
	body, err := webrtcutil.EncodeSDP(&offer)
	assert.NoError(t, err)
	assert.NoError(t, websocket.JSON.Send(ws, &message{Operation: OperationOffer, SessionID: sid, Body: body}))

	answer := <-answerCh
	assert.NoError(t, pcOffer.SetRemoteDescription(answer))
	log.Printf("[pcOffer] got answer")
	for candidate := range pendingCandidateCh {
		assert.NoError(t, pcOffer.AddICECandidate(candidate))
	}

	<-connected

	videoTrack := <-videoTrackCh
	audioTrack := <-audioTrackCh
	assert.Equal(t, "mashimaro", videoTrack.StreamID())
	assert.Equal(t, "mashimaro", audioTrack.StreamID())
	assert.NoError(t, pcOffer.Close())
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
	hs := httptest.NewServer(s.WebSocketHandler())
	return &testServer{
		Server: s,
		hs:     hs,
	}
}
