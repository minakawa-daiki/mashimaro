package broker

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/castaneai/mashimaro/pkg/game"

	"github.com/castaneai/mashimaro/pkg/testutils"

	"github.com/stretchr/testify/assert"

	"github.com/castaneai/mashimaro/pkg/gameserver"

	"github.com/castaneai/mashimaro/pkg/gamesession"

	"github.com/castaneai/mashimaro/pkg/proto"
	"google.golang.org/grpc"
)

type externalBrokerClient struct {
	hs *httptest.Server
}

func newExternalBrokerClient(b *Broker) *externalBrokerClient {
	return &externalBrokerClient{
		httptest.NewServer(ExternalServer(b)),
	}
}

func (ts *externalBrokerClient) NewGame(gameID string) (gamesession.SessionID, error) {
	url := fmt.Sprintf("%s/newgame/%s", ts.hs.URL, gameID)
	resp, err := http.Post(url, "application/json", nil)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	var newGameResp NewGameResponse
	dec := json.NewDecoder(resp.Body)
	if err := dec.Decode(&newGameResp); err != nil {
		return "", err
	}
	return newGameResp.SessionID, nil
}

func newInternalBrokerClient(t *testing.T, sstore gamesession.Store, mstore game.MetadataStore) proto.BrokerClient {
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

	sstore := gamesession.NewInMemoryStore()
	mstore := game.NewMockMetadataStore()
	if err := mstore.AddGameMetadata(ctx, "test-game", &game.Metadata{
		GameID:  "test-game",
		Command: "wine notepad",
	}); err != nil {
		t.Fatal(err)
	}
	gs := &gameserver.GameServer{Name: "dummy", Addr: "dummy-addr"}
	allocator := gameserver.NewMockAllocator(gs)
	b := NewBroker(sstore, mstore, allocator)

	ic := newInternalBrokerClient(t, sstore, mstore)
	{
		resp, err := ic.FindSession(ctx, &proto.FindSessionRequest{GameserverName: gs.Name})
		assert.NoError(t, err)
		assert.False(t, resp.Found)
	}

	// create game session
	ec := newExternalBrokerClient(b)
	sid, err := ec.NewGame("test-game")
	assert.NoError(t, err)
	assert.NotEmpty(t, sid)

	{
		resp, err := ic.FindSession(ctx, &proto.FindSessionRequest{GameserverName: gs.Name})
		assert.NoError(t, err)
		assert.True(t, resp.Found)
		assert.Equal(t, sid, gamesession.SessionID(resp.Session.SessionId))
	}
}
