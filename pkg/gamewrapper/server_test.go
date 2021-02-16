package gamewrapper

import (
	"context"
	"testing"

	"github.com/castaneai/mashimaro/pkg/proto"
	"github.com/castaneai/mashimaro/pkg/testutils"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
)

func newGameWrapperClient(t *testing.T) proto.GameWrapperClient {
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

func TestHealthCheck(t *testing.T) {
	wc := newGameWrapperClient(t)
	ctx := context.Background()
	if _, err := wc.StartGame(ctx, &proto.StartGameRequest{Command: "wine", Args: []string{"notepad"}}); err != nil {
		t.Fatal(err)
	}

	{
		resp, err := wc.HealthCheck(ctx, &proto.HealthCheckRequest{})
		assert.NoError(t, err)
		assert.True(t, resp.Healthy)
	}

	{
		_, err := wc.ExitGame(ctx, &proto.ExitGameRequest{})
		assert.NoError(t, err)
	}

	{
		resp, err := wc.HealthCheck(ctx, &proto.HealthCheckRequest{})
		assert.NoError(t, err)
		assert.False(t, resp.Healthy)
	}
}
