package transport

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/castaneai/mashimaro/pkg/ayame"

	"github.com/pion/webrtc/v3"
	"github.com/stretchr/testify/assert"
)

func TestSignaling(t *testing.T) {
	ayameURL := os.Getenv("AYAME_URL")
	if ayameURL == "" {
		t.Skip("Set AYAME_URL to run this test")
	}

	conn1, err := NewWebRTCStreamerConn(webrtc.Configuration{})
	assert.NoError(t, err)
	conn1Connected := make(chan struct{})
	conn1.OnConnect(func() {
		close(conn1Connected)
	})
	conn2, err := NewWebRTCPlayerConn(webrtc.Configuration{})
	assert.NoError(t, err)
	conn2Connected := make(chan struct{})
	conn2.OnConnect(func() {
		close(conn2Connected)
	})

	rid := "test-room"
	ctx := context.Background()
	c1 := ayame.NewClient(conn1.PeerConnection())
	if err := c1.Connect(ctx, ayameURL, &ayame.ConnectRequest{
		RoomID:   rid,
		ClientID: "client1",
	}); err != nil {
		t.Fatal(err)
	}

	c2 := ayame.NewClient(conn2.PeerConnection())
	if err := c2.Connect(ctx, ayameURL, &ayame.ConnectRequest{
		RoomID:   rid,
		ClientID: "client2",
	}); err != nil {
		t.Fatal(err)
	}

	<-conn1Connected
	<-conn2Connected
}

func TestSignalingViaAyameLabo(t *testing.T) {
	ayameLaboURL := os.Getenv("AYAME_LABO_URL")
	if ayameLaboURL == "" {
		t.Skip("AYAME_LABO_URL not set")
	}
	signalingKey := os.Getenv("AYAME_LABO_SIGNALING_KEY")
	if signalingKey == "" {
		t.Skip("AYAME_LABO_SIGNALING_KEY not set")
	}
	githubAccount := os.Getenv("AYAME_LABO_GITHUB_ACCOUNT")
	if githubAccount == "" {
		t.Skip("AYAME_LABO_GITHUB_ACCOUNT not set")
	}

	conn1, err := NewWebRTCStreamerConn(webrtc.Configuration{})
	assert.NoError(t, err)
	conn1Connected := make(chan struct{})
	conn1.OnConnect(func() {
		close(conn1Connected)
	})
	conn2, err := NewWebRTCPlayerConn(webrtc.Configuration{})
	assert.NoError(t, err)
	conn2Connected := make(chan struct{})
	conn2.OnConnect(func() {
		close(conn2Connected)
	})

	rid := fmt.Sprintf("%s@%s", githubAccount, "test-room")
	ctx := context.Background()
	c1 := ayame.NewClient(conn1.PeerConnection())
	if err := c1.Connect(ctx, ayameLaboURL, &ayame.ConnectRequest{
		RoomID:       rid,
		ClientID:     "client1",
		SignalingKey: signalingKey,
	}); err != nil {
		t.Fatal(err)
	}

	c2 := ayame.NewClient(conn2.PeerConnection())
	if err := c2.Connect(ctx, ayameLaboURL, &ayame.ConnectRequest{
		RoomID:       rid,
		ClientID:     "client2",
		SignalingKey: signalingKey,
	}); err != nil {
		t.Fatal(err)
	}

	<-conn1Connected
	<-conn2Connected
}
