package tests

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/url"
	"testing"
	"time"

	"github.com/castaneai/mashimaro/pkg/ayame"

	"github.com/castaneai/mashimaro/pkg/gameagent"

	"github.com/castaneai/mashimaro/pkg/gameserver"
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

func TestSignaling(t *testing.T) {
	checkAyame(t)

	ctx := context.Background()
	gs := &gameserver.GameServer{Name: "dummy", Addr: "dummy-addr"}
	store := gamesession.NewInMemoryStore()
	bc := newInternalBrokerClient(t, store)
	sigConf := &gameagent.SignalingConfig{AyameURL: ayameURL}
	agent := gameagent.NewAgent(bc, sigConf)
	mediaTracks, err := gameagent.NewMediaTracks()
	assert.NoError(t, err)
	go func() {
		if err := agent.Run(ctx, gs.Name, mediaTracks); err != nil {
			log.Printf("failed to run agent: %+v", err)
		}
	}()

	ss, err := store.NewSession(ctx, &gamesession.NewSessionRequest{
		GameID:     "test-game",
		GameServer: gs,
	})
	assert.NoError(t, err)
	sid := ss.SessionID

	pcOffer := ayame.NewClient(ayame.WithInitPeerConnection(func(pc *webrtc.PeerConnection) error {
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
		ClientID: "gameplayer",
	}); err != nil {
		t.Fatal(err)
	}
	<-connected
}
