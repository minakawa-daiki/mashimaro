package broker

import (
	"context"
	"log"

	"github.com/castaneai/mashimaro/pkg/gameserver"

	"github.com/castaneai/mashimaro/pkg/gamesession"
)

// Broker has two servers:
// 1. HTTP server interacts with client
// 2. gRPC server interacts with game server
type Broker struct {
	sessionStore gamesession.Store
	allocator    gameserver.Allocator
}

func NewBroker(sessionStore gamesession.Store, allocator gameserver.Allocator) *Broker {
	return &Broker{sessionStore: sessionStore, allocator: allocator}
}

func (b *Broker) NewGame(ctx context.Context, gameID string) (*gamesession.Session, error) {
	gameServer, err := b.allocator.Allocate(ctx)
	if err != nil {
		return nil, err
	}
	ss, err := b.sessionStore.NewSession(ctx, &gamesession.NewSessionRequest{
		GameID:     gameID,
		GameServer: gameServer,
	})
	if err != nil {
		return nil, err
	}
	log.Printf("created game session: %s", ss.SessionID)
	return ss, nil
}
