package tests

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/castaneai/mashimaro/pkg/testutils"

	"github.com/stretchr/testify/assert"

	"github.com/castaneai/mashimaro/pkg/gameserver"

	"github.com/castaneai/mashimaro/pkg/broker"

	"github.com/castaneai/mashimaro/pkg/gamesession"

	"github.com/castaneai/mashimaro/pkg/proto"
	"google.golang.org/grpc"
)

type externalBrokerClient struct {
	hs *httptest.Server
}

func newExternalBrokerClient(b *broker.Broker) *externalBrokerClient {
	return &externalBrokerClient{
		httptest.NewServer(broker.ExternalServer(b)),
	}
}

func (ts *externalBrokerClient) NewGame(gameID string) (gamesession.SessionID, error) {
	url := fmt.Sprintf("%s/newgame/%s", ts.hs.URL, gameID)
	resp, err := http.Post(url, "application/json", nil)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	var newGameResp broker.NewGameResponse
	dec := json.NewDecoder(resp.Body)
	if err := dec.Decode(&newGameResp); err != nil {
		return "", err
	}
	return newGameResp.SessionID, nil
}

func newInternalBrokerClient(t *testing.T, store gamesession.Store) proto.BrokerClient {
	lis := testutils.ListenTCPWithRandomPort(t)
	s := grpc.NewServer()
	proto.RegisterBrokerServer(s, broker.NewInternalServer(store))
	go s.Serve(lis)
	cc, err := grpc.Dial(lis.Addr().String(), grpc.WithInsecure())
	if err != nil {
		t.Fatalf("failed to dial to internal broker: %+v", err)
	}
	return proto.NewBrokerClient(cc)
}

func TestBroker(t *testing.T) {
	ctx := context.Background()

	store := gamesession.NewInMemoryStore()
	gs := &gameserver.GameServer{Name: "dummy", Addr: "dummy-addr"}
	allocator := gameserver.NewMockAllocator(gs)
	b := broker.NewBroker(store, allocator)

	ic := newInternalBrokerClient(t, store)
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
