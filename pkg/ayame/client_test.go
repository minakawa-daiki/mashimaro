package ayame

import (
	"context"
	"fmt"
	"net"
	"net/url"
	"testing"
	"time"
)

const (
	ayameURL = "ws://localhost:3000/signaling"
)

func TestSignaling(t *testing.T) {
	u, err := url.Parse(ayameURL)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := net.DialTimeout("tcp", u.Host, 100*time.Millisecond); err != nil {
		t.Skip(fmt.Sprintf("A Test was skipped. Make sure that Ayame is running on %s", ayameURL))
	}

	rid := "test-room"
	ctx := context.Background()
	c1 := NewClient()
	c1Connected := make(chan struct{})
	c1.OnConnect(func() {
		close(c1Connected)
	})
	if err := c1.Connect(ctx, ayameURL, &ConnectRequest{
		RoomID:   rid,
		ClientID: "client1",
	}); err != nil {
		t.Fatal(err)
	}

	c2 := NewClient()
	c2Connected := make(chan struct{})
	c2.OnConnect(func() {
		close(c2Connected)
	})
	if err := c2.Connect(ctx, ayameURL, &ConnectRequest{
		RoomID:   rid,
		ClientID: "client2",
	}); err != nil {
		t.Fatal(err)
	}

	<-c1Connected
	<-c2Connected
}
