package transport

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/pion/webrtc/v3"
)

func signalPair(pcOffer, pcAnswer *webrtc.PeerConnection) error {
	offer, err := pcOffer.CreateOffer(nil)
	if err != nil {
		return err
	}
	offerGatheringComplete := webrtc.GatheringCompletePromise(pcOffer)
	if err = pcOffer.SetLocalDescription(offer); err != nil {
		return err
	}
	<-offerGatheringComplete
	if err = pcAnswer.SetRemoteDescription(*pcOffer.LocalDescription()); err != nil {
		return err
	}

	answer, err := pcAnswer.CreateAnswer(nil)
	if err != nil {
		return err
	}
	answerGatheringComplete := webrtc.GatheringCompletePromise(pcAnswer)
	if err = pcAnswer.SetLocalDescription(answer); err != nil {
		return err
	}
	<-answerGatheringComplete
	return pcOffer.SetRemoteDescription(*pcAnswer.LocalDescription())
}

func newStreamerConn(t *testing.T, wc webrtc.Configuration) *WebRTCStreamerConn {
	streamer, err := NewWebRTCStreamerConn(wc)
	if err != nil {
		t.Fatal(err)
	}
	return streamer
}

func newPlayerConn(t *testing.T, wc webrtc.Configuration) *WebRTCPlayerConn {
	player, err := NewWebRTCPlayerConn(wc)
	if err != nil {
		t.Fatal(err)
	}
	return player
}

func signalStreamerPlayer(t *testing.T, wc webrtc.Configuration, isPlayerOffer bool) (streamer *WebRTCStreamerConn, player *WebRTCPlayerConn) {
	streamer = newStreamerConn(t, wc)
	streamerConnected := make(chan struct{})
	streamer.OnConnect(func() {
		close(streamerConnected)
	})

	player = newPlayerConn(t, wc)
	playerConnected := make(chan struct{})
	player.OnConnect(func() {
		close(playerConnected)
	})
	if isPlayerOffer {
		if err := signalPair(player.PeerConnection(), streamer.PeerConnection()); err != nil {
			t.Fatal(err)
		}
	} else {
		if err := signalPair(streamer.PeerConnection(), player.PeerConnection()); err != nil {
			t.Fatal(err)
		}
	}
	<-streamerConnected
	<-playerConnected
	return streamer, player
}

func TestMessaging(t *testing.T) {
	tcs := []struct {
		name          string
		isPlayerOffer bool
	}{
		{name: "OfferByPlayer", isPlayerOffer: true},
		{name: "OfferByStreamer", isPlayerOffer: false},
	}
	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			streamer, player := signalStreamerPlayer(t, webrtc.Configuration{}, tc.isPlayerOffer)
			assert.NotNil(t, streamer)
			assert.NotNil(t, player)
			streamerReceived := make(chan []byte)
			streamer.OnMessage(func(data []byte) {
				streamerReceived <- data
			})
			playerReceived := make(chan []byte)
			player.OnMessage(func(data []byte) {
				playerReceived <- data
			})
			ctx := context.Background()

			if err := player.SendMessage(ctx, []byte("hello")); err != nil {
				t.Fatal(err)
			}
			assert.Equal(t, []byte("hello"), <-streamerReceived)

			if err := streamer.SendMessage(ctx, []byte("hello2")); err != nil {
				t.Fatal(err)
			}
			assert.Equal(t, []byte("hello2"), <-playerReceived)
		})
	}

}
