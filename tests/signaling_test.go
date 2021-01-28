package tests

import (
	"context"
	"fmt"
	"log"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/castaneai/mashimaro/pkg/gameagent"

	"github.com/castaneai/mashimaro/pkg/gameserver"
	"github.com/stretchr/testify/assert"

	"github.com/castaneai/mashimaro/pkg/proto"
	"google.golang.org/grpc"

	"github.com/castaneai/mashimaro/pkg/webrtcutil"
	"github.com/pion/webrtc/v3"

	"github.com/castaneai/mashimaro/pkg/gamesession"
	"github.com/castaneai/mashimaro/pkg/signaling"

	"golang.org/x/net/websocket"
)

type externalSignalingClient struct {
	conn              *websocket.Conn
	hs                *httptest.Server
	answerCh          chan *webrtc.SessionDescription
	answerCandidateCh chan *webrtc.ICECandidateInit
}

func newExternalSignalingClient(t *testing.T, store gamesession.Store, channels *signaling.Channels, answerCandidateCh chan *webrtc.ICECandidateInit) *externalSignalingClient {
	s := signaling.NewExternalServer(store, channels)
	hs := httptest.NewServer(s.WebSocketHandler())
	wsURL := strings.Replace(hs.URL, "http:", "ws:", 1)
	conn, err := websocket.Dial(wsURL, "", hs.URL)
	if err != nil {
		t.Fatal(err)
	}

	answerCh := make(chan *webrtc.SessionDescription)
	go func() {
		for {
			var msg signaling.WSMessage
			if err := websocket.JSON.Receive(conn, &msg); err != nil {
				log.Printf("failed to receive via websocket")
				return
			}
			switch msg.Operation {
			case signaling.OperationAnswer:
				answer, err := webrtcutil.DecodeSDP(msg.Body)
				if err != nil {
					log.Printf("failed to decode SDP: %+v", err)
					return
				}
				answerCh <- answer
			case signaling.OperationICECandidate:
				var cand *webrtc.ICECandidateInit
				if msg.Body != "" {
					c, err := webrtcutil.DecodeICECandidate(msg.Body)
					if err != nil {
						log.Printf("failed to decode ICE candidate: %+v", err)
						return
					}
					cand = c
				}
				answerCandidateCh <- cand
			default:
				log.Printf("unknown operation received: %+v", msg)
			}
		}
	}()
	return &externalSignalingClient{
		conn:              conn,
		hs:                hs,
		answerCh:          answerCh,
		answerCandidateCh: answerCandidateCh,
	}
}

func (c *externalSignalingClient) SendOffer(ctx context.Context, sid gamesession.SessionID, offer *webrtc.SessionDescription) *webrtc.SessionDescription {
	body, err := webrtcutil.EncodeSDP(offer)
	if err != nil {
		panic("failed to encode SDP")
	}
	if err := websocket.JSON.Send(c.conn, &signaling.WSMessage{
		Operation: signaling.OperationOffer,
		SessionID: sid,
		Body:      body,
	}); err != nil {
		panic("failed to send via websocket")
	}
	select {
	case <-ctx.Done():
		panic("waiting for answer timed out")
	case answer := <-c.answerCh:
		return answer
	}
}

func (c *externalSignalingClient) SendICECandidate(ctx context.Context, sid gamesession.SessionID, cand *webrtc.ICECandidate) {
	body := ""
	if cand != nil {
		candidateInit := cand.ToJSON()
		b, err := webrtcutil.EncodeICECandidate(&candidateInit)
		if err != nil {
			panic(fmt.Sprintf("failed to encode ICE candidate: %+v", err))
		}
		body = b
	}
	if err := websocket.JSON.Send(c.conn, &signaling.WSMessage{
		Operation: signaling.OperationICECandidate,
		SessionID: sid,
		Body:      body,
	}); err != nil {
		panic("failed to send ICE candidate from pcOffer")
	}
}

func newInternalSignalingClient(t *testing.T, store gamesession.Store, channels *signaling.Channels) proto.SignalingClient {
	lis := listenTCPWithRandomPort(t)
	s := grpc.NewServer()
	proto.RegisterSignalingServer(s, signaling.NewInternalServer(store, channels))
	go s.Serve(lis)
	cc, err := grpc.Dial(lis.Addr().String(), grpc.WithInsecure())
	if err != nil {
		t.Fatalf("failed to dial to internal broker: %+v", err)
	}
	return proto.NewSignalingClient(cc)
}

func TestSignaling(t *testing.T) {
	ctx := context.Background()
	gs := &gameserver.GameServer{Name: "dummy", Addr: "dummy-addr"}
	store := gamesession.NewInMemoryStore()
	channels := signaling.NewChannels()
	bc := newInternalBrokerClient(t, store)
	ic := newInternalSignalingClient(t, store, channels)
	agent := gameagent.NewAgent(bc, ic)
	go func() {
		if err := agent.Run(ctx, gs.Name); err != nil {
			log.Printf("failed to run agent: %+v", err)
		}
	}()

	ss, err := store.NewSession(ctx, &gamesession.NewSessionRequest{
		GameID:     "test-game",
		GameServer: gs,
	})
	assert.NoError(t, err)
	sid := ss.SessionID

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
	once := &sync.Once{}
	pcOffer.OnICEConnectionStateChange(func(state webrtc.ICEConnectionState) {
		log.Printf("[pcOffer] ICE state has changed: %s", state)
	})
	pcOffer.OnConnectionStateChange(func(state webrtc.PeerConnectionState) {
		log.Printf("[pcOffer] connection state has changed: %s", state)
		if state == webrtc.PeerConnectionStateConnected {
			once.Do(func() {
				close(connected)
			})
		}
	})

	var pendingCandidates []webrtc.ICECandidateInit
	var pendingMu sync.Mutex
	answerCandidateCh := make(chan *webrtc.ICECandidateInit, 10)
	go func() {
		for candidate := range answerCandidateCh {
			if candidate == nil {
				break
			}
			if pcOffer.RemoteDescription() != nil {
				pcOffer.AddICECandidate(*candidate)
			} else {
				pendingMu.Lock()
				pendingCandidates = append(pendingCandidates, *candidate)
				pendingMu.Unlock()
			}
		}
	}()
	ec := newExternalSignalingClient(t, store, channels, answerCandidateCh)

	pcOffer.OnICECandidate(func(candidate *webrtc.ICECandidate) {
		log.Printf("[pcOffer] new ICE candidate: %v", candidate)
		ec.SendICECandidate(ctx, sid, candidate)
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

	offer, err := pcOffer.CreateOffer(nil)
	assert.NoError(t, err)
	assert.NoError(t, pcOffer.SetLocalDescription(offer))

	answerCtx, cancel := context.WithTimeout(ctx, 5*time.Hour)
	defer cancel()
	answer := ec.SendOffer(answerCtx, sid, &offer)
	assert.NoError(t, pcOffer.SetRemoteDescription(*answer))
	log.Printf("[pcOffer] got answer")
	(func() {
		pendingMu.Lock()
		defer pendingMu.Unlock()
		for _, candidate := range pendingCandidates {
			if err := pcOffer.AddICECandidate(candidate); err != nil {
				log.Printf("failed to add ICE candidate: %+v", err)
			}
		}
	})()

	videoTrack := <-videoTrackCh
	audioTrack := <-audioTrackCh
	assert.Equal(t, "mashimaro", videoTrack.StreamID())
	assert.Equal(t, "mashimaro", audioTrack.StreamID())

	<-connected
	assert.NoError(t, pcOffer.Close())
}
