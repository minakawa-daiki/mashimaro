package gamewrapper

import (
	"context"
	"testing"

	"golang.org/x/sync/errgroup"

	"github.com/castaneai/mashimaro/pkg/proto"
	"github.com/castaneai/mashimaro/pkg/testutils"
	"google.golang.org/grpc"
)

func TestNotePad(t *testing.T) {
	lis := testutils.ListenTCPWithRandomPort(t)
	cc, err := grpc.Dial(lis.Addr().String(), grpc.WithInsecure())
	if err != nil {
		t.Fatal(err)
	}
	wc := proto.NewGameWrapperClient(cc)

	w := NewGameWrapper()
	eg := &errgroup.Group{}
	eg.Go(func() error {
		return w.Run(lis)
	})

	ctx := context.Background()
	if _, err := wc.StartGame(ctx, &proto.StartGameRequest{Command: "wine", Args: []string{"notepad"}}); err != nil {
		t.Fatal(err)
	}
	if err := eg.Wait(); err != nil {
		t.Fatal(err)
	}
}
