package broker

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/castaneai/mashimaro/pkg/allocator"

	"github.com/castaneai/mashimaro/pkg/gamemetadata"

	"github.com/castaneai/mashimaro/pkg/gamesession"
)

type externalBrokerClient struct {
	hs *httptest.Server
}

func newExternalBrokerClient(sstore gamesession.Store, mstore gamemetadata.Store, allocator allocator.Allocator) *externalBrokerClient {
	s := NewExternalBroker(sstore, mstore, allocator)
	return &externalBrokerClient{
		httptest.NewServer(s.HTTPHandler()),
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

func TestExternalBroker(t *testing.T) {
	ctx := context.Background()
	sstore := gamesession.NewInMemoryStore()
	mstore := gamemetadata.NewInMemoryStore()
	metadata := &gamemetadata.Metadata{
		GameID:  "test-game",
		Command: "test-command",
	}
	assert.NoError(t, mstore.AddGameMetadata(ctx, metadata))
	allocatedServer := &allocator.AllocatedServer{ID: "dummy"}
	alloc := allocator.NewMockAllocator(allocatedServer)
	client := newExternalBrokerClient(sstore, mstore, alloc)

	sid, err := client.NewGame(metadata.GameID)
	assert.NoError(t, err)
	assert.NotEmpty(t, sid)
	ss, err := sstore.GetSession(ctx, sid)
	assert.NoError(t, err)
	assert.Equal(t, ss.SessionID, sid)
}
