package gamewrapper

import (
	"context"
	"testing"
	"time"

	"github.com/castaneai/mashimaro/pkg/proto"
	"github.com/castaneai/mashimaro/pkg/testutils"
	"google.golang.org/grpc"
)

func NewGameWrapperClient(t *testing.T) proto.GameWrapperClient {
	lis := testutils.ListenTCPWithRandomPort(t)
	s := grpc.NewServer()
	proto.RegisterGameWrapperServer(s, NewGameWrapperServer())
	go s.Serve(lis)
	cc, err := grpc.Dial(lis.Addr().String(), grpc.WithInsecure())
	if err != nil {
		t.Fatalf("failed to dial to game wrapper: %+v", err)
	}
	return proto.NewGameWrapperClient(cc)
}

func TestNotePad(t *testing.T) {
	wc := NewGameWrapperClient(t)
	ctx := context.Background()
	if _, err := wc.StartGame(ctx, &proto.StartGameRequest{Command: "wine", Args: []string{"notepad"}}); err != nil {
		t.Fatal(err)
	}
	time.Sleep(1000 * time.Millisecond)
	if _, err := wc.ExitGame(ctx, &proto.ExitGameRequest{}); err != nil {
		t.Fatal(err)
	}
}
