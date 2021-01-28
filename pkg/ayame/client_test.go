package ayame

import (
	"context"
	"testing"
)

const (
	ayameURL = "ws://localhost:3000/signaling"
)

func TestRegister(t *testing.T) {
	rid := "test-room"
	ctx := context.Background()
	c1 := NewClient()
	if err := c1.Connect(ctx, ayameURL, &ConnectRequest{
		RoomID:   rid,
		ClientID: "client1",
	}); err != nil {
		t.Fatal(err)
	}

	c2 := NewClient()
	if err := c2.Connect(ctx, ayameURL, &ConnectRequest{
		RoomID:   rid,
		ClientID: "client2",
	}); err != nil {
		t.Fatal(err)
	}

	select {}
}
