package tests

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"testing"
	"time"

	"github.com/castaneai/mashimaro/pkg/broker"

	"github.com/castaneai/mashimaro/pkg/ayame"

	"github.com/stretchr/testify/assert"

	"github.com/pion/webrtc/v3"

	"github.com/castaneai/mashimaro/pkg/gamesession"
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

func newGame(gameID string) (gamesession.SessionID, error) {
	addr := os.Getenv("BROKER_EXTERNAL_URL")
	url := fmt.Sprintf("%s/newgame/%s", addr, gameID)
	resp, err := http.Post(url, "application/json", nil)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	var newGameResp broker.NewGameResponse
	dec := json.NewDecoder(resp.Body)
	if err := dec.Decode(&newGameResp); err != nil {
		return "", err
	}
	return newGameResp.SessionID, nil
}

func TestPlaying(t *testing.T) {
	checkAyame(t)

	ctx := context.Background()
	sid, err := newGame("test-game")
	assert.NoError(t, err)

	dcCh := make(chan *webrtc.DataChannel)
	videoTrackCh := make(chan *webrtc.TrackRemote)
	audioTrackCh := make(chan *webrtc.TrackRemote)

	pcOffer := ayame.NewClient(ayame.WithInitPeerConnection(func(pc *webrtc.PeerConnection) error {
		dc, err := pc.CreateDataChannel("data", nil)
		if err != nil {
			return err
		}
		log.Printf("data channel created: %+v", dc)
		dc.OnOpen(func() {
			dcCh <- dc
		})
		dc.OnError(func(err error) {
			log.Printf("data channel error: %+v", err)
		})
		dc.OnClose(func() {
			log.Printf("data channel closed: %+v", dc)
		})
		dc.OnMessage(func(msg webrtc.DataChannelMessage) {
			log.Printf("data channel msg received: %+v", msg)
		})
		pc.OnTrack(func(remote *webrtc.TrackRemote, receiver *webrtc.RTPReceiver) {
			log.Printf("remote track found: %+v", remote)
			switch remote.Kind() {
			case webrtc.RTPCodecTypeVideo:
				videoTrackCh <- remote
			case webrtc.RTPCodecTypeAudio:
				audioTrackCh <- remote
			default:
				log.Printf("unknown track found: %+v", remote)
			}
		})
		if _, err := pc.AddTransceiverFromKind(webrtc.RTPCodecTypeVideo, webrtc.RTPTransceiverInit{Direction: webrtc.RTPTransceiverDirectionRecvonly}); err != nil {
			return err
		}
		if _, err := pc.AddTransceiverFromKind(webrtc.RTPCodecTypeAudio, webrtc.RTPTransceiverInit{Direction: webrtc.RTPTransceiverDirectionRecvonly}); err != nil {
			return err
		}
		return nil
	}))

	connected := make(chan struct{})
	pcOffer.OnConnect(func() {
		close(connected)
	})
	if err := pcOffer.Connect(ctx, ayameURL, &ayame.ConnectRequest{
		RoomID:   string(sid),
		ClientID: "player",
	}); err != nil {
		t.Fatal(err)
	}
	<-connected
	log.Printf("player connected!")

	dc := <-dcCh
	log.Printf("datachannel opened")
	assert.NoError(t, dc.SendText("hello"))

	videoTrack := <-videoTrackCh
	videoRTP, _, err := videoTrack.ReadRTP()
	assert.NotNil(t, videoRTP)
	assert.NoError(t, err)

	audioTrack := <-audioTrackCh
	audioRTP, _, err := audioTrack.ReadRTP()
	assert.NotNil(t, audioRTP)
	assert.NoError(t, err)
}
