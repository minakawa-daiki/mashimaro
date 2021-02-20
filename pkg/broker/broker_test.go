package broker

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/castaneai/mashimaro/pkg/allocator"

	"github.com/castaneai/mashimaro/pkg/gamemetadata"

	"github.com/castaneai/mashimaro/pkg/testutils"

	"github.com/stretchr/testify/assert"

	"github.com/castaneai/mashimaro/pkg/gamesession"

	"github.com/castaneai/mashimaro/pkg/proto"
	"google.golang.org/grpc"
)

type externalBrokerClient struct {
	hs *httptest.Server
}

func newExternalBrokerClient(sstore gamesession.Store, mstore gamemetadata.Store, allocator allocator.Allocator) *externalBrokerClient {
	s := NewExternalServer(sstore, mstore, allocator)
	return &externalBrokerClient{
		httptest.NewServer(s.Handler()),
	}
}

func (ts *externalBrokerClient) NewGame(gameID string) (gamesession.SessionID, error) {
	url := fmt.Sprintf("%s/newgame/%s", ts.hs.URL, gameID)
	resp, err := http.Post(url, "application/json", nil)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	var newGameResp newGameResponse
	dec := json.NewDecoder(resp.Body)
	if err := dec.Decode(&newGameResp); err != nil {
		return "", err
	}
	return newGameResp.SessionID, nil
}

func newInternalBrokerClient(t *testing.T, sstore gamesession.Store, mstore gamemetadata.Store) proto.BrokerClient {
	lis := testutils.ListenTCPWithRandomPort(t)
	s := grpc.NewServer()
	proto.RegisterBrokerServer(s, NewInternalServer(sstore, mstore))
	go s.Serve(lis)
	cc, err := grpc.Dial(lis.Addr().String(), grpc.WithInsecure())
	if err != nil {
		t.Fatalf("failed to dial to internal broker: %+v", err)
	}
	return proto.NewBrokerClient(cc)
}

func TestBroker(t *testing.T) {
	ctx := context.Background()

	metadata := &gamemetadata.Metadata{
		GameID:  "notepad",
		Command: "wine notepad",
	}
	sstore := gamesession.NewInMemoryStore()
	mstore := gamemetadata.NewInMemoryMetadataStore()
	if err := mstore.AddGameMetadata(ctx, metadata.GameID, metadata); err != nil {
		t.Fatal(err)
	}
	gs := &allocator.AllocatedServer{ID: "dummy"}
	allocator := allocator.NewMockAllocator(gs)

	ic := newInternalBrokerClient(t, sstore, mstore)
	watchStream, err := ic.WatchSession(ctx, &proto.WatchSessionRequest{AllocatedServerId: gs.ID})
	assert.NoError(t, err)

	// create game session
	ec := newExternalBrokerClient(sstore, mstore, allocator)
	sid, err := ec.NewGame(metadata.GameID)
	assert.NoError(t, err)
	assert.NotEmpty(t, sid)

	watchResp, err := watchStream.Recv()
	assert.NoError(t, err)
	assert.True(t, watchResp.Found)
	assert.Equal(t, sid, gamesession.SessionID(watchResp.Session.SessionId))
	assert.Equal(t, gs.ID, watchResp.Session.AllocatedServerId)

	assert.NoError(t, sstore.DeleteSession(ctx, sid))

	watchResp, err = watchStream.Recv()
	assert.NoError(t, err)
	assert.False(t, watchResp.Found)
	assert.Nil(t, watchResp.Session)
}
