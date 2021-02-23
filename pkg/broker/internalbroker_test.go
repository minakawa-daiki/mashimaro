package broker

import (
	"context"
	"testing"

	"github.com/castaneai/mashimaro/pkg/allocator"
	"github.com/stretchr/testify/assert"

	"github.com/castaneai/mashimaro/pkg/gamemetadata"
	"github.com/castaneai/mashimaro/pkg/proto"
	"github.com/castaneai/mashimaro/pkg/testutils"
	"google.golang.org/grpc"

	"github.com/castaneai/mashimaro/pkg/gamesession"
)

func TestInternalBroker(t *testing.T) {
	ctx := context.Background()
	sstore := gamesession.NewInMemoryStore()
	mstore := gamemetadata.NewInMemoryStore()
	metadata := &gamemetadata.Metadata{
		GameID:  "test-game",
		Command: "test-command",
	}
	assert.NoError(t, mstore.AddGameMetadata(ctx, metadata))
	allocatedServer := &allocator.AllocatedServer{ID: "dummy"}
	client := newInternalBrokerClient(t, sstore, mstore)

	resp, err := client.FindSession(ctx, &proto.FindSessionRequest{AllocatedServerId: allocatedServer.ID})
	assert.NoError(t, err)
	assert.False(t, resp.Found)
	assert.Nil(t, resp.Session)

	ss, err := sstore.NewSession(ctx, &gamesession.NewSessionRequest{
		GameID:            metadata.GameID,
		AllocatedServerID: allocatedServer.ID,
	})
	assert.NoError(t, err)

	resp, err = client.FindSession(ctx, &proto.FindSessionRequest{AllocatedServerId: allocatedServer.ID})
	assert.NoError(t, err)
	assert.True(t, resp.Found)
	assert.Equal(t, string(ss.SessionID), resp.Session.SessionId)

	mdResp, err := client.GetGameMetadata(ctx, &proto.GetGameMetadataRequest{GameId: metadata.GameID})
	assert.NoError(t, err)
	var respMd gamemetadata.Metadata
	assert.NoError(t, gamemetadata.Unmarshal([]byte(mdResp.GameMetadata.Body), &respMd))
	assert.Equal(t, metadata.GameID, respMd.GameID)
	assert.Equal(t, metadata.Command, respMd.Command)

	_, err = client.DeleteSession(ctx, &proto.DeleteSessionRequest{
		SessionId:         resp.Session.SessionId,
		AllocatedServerId: resp.Session.AllocatedServerId,
	})
	assert.NoError(t, err)

	resp, err = client.FindSession(ctx, &proto.FindSessionRequest{AllocatedServerId: allocatedServer.ID})
	assert.NoError(t, err)
	assert.False(t, resp.Found)
	assert.Nil(t, resp.Session)
}

func newInternalBrokerClient(t *testing.T, sstore gamesession.Store, mstore gamemetadata.Store) proto.BrokerClient {
	lis := testutils.ListenTCPWithRandomPort(t)
	s := grpc.NewServer()
	proto.RegisterBrokerServer(s, NewInternalBroker(sstore, mstore))
	go s.Serve(lis)
	cc, err := grpc.Dial(lis.Addr().String(), grpc.WithInsecure())
	if err != nil {
		t.Fatalf("failed to dial to internal broker: %+v", err)
	}
	return proto.NewBrokerClient(cc)
}
