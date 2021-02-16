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

func signalStreamerPlayer(t *testing.T, wc webrtc.Configuration) (streamer *WebRTCStreamerConn, player *WebRTCPlayerConn) {
	streamerConnected := make(chan struct{})
	streamer, err := NewWebRTCStreamerConn(wc)
	if err != nil {
		t.Fatal(err)
	}
	streamer.OnConnect(func() {
		close(streamerConnected)
	})
	playerConnected := make(chan struct{})
	player, err = NewWebRTCPlayerConn(wc)
	if err != nil {
		t.Fatal(err)
	}
	player.OnConnect(func() {
		close(playerConnected)
	})
	if err := signalPair(player.pc, streamer.pc); err != nil {
		t.Fatal(err)
	}
	<-streamerConnected
	<-playerConnected
	return streamer, player
}

func TestMessaging(t *testing.T) {
	streamer, player := signalStreamerPlayer(t, webrtc.Configuration{})
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
}
