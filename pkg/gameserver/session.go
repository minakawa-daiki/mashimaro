package gameserver

import (
	"context"

	"github.com/castaneai/mashimaro/pkg/gamesession"

	"github.com/castaneai/mashimaro/pkg/proto"
)

func (s *GameServer) startWatchSession(ctx context.Context, created chan<- *gamesession.Session, deleted chan<- struct{}) error {
	stream, err := s.broker.WatchSession(ctx, &proto.WatchSessionRequest{AllocatedServerId: s.allocatedServer.ID})
	if err != nil {
		return err
	}
	sessionFound := false
	for {
		resp, err := stream.Recv()
		if err != nil {
			return err
		}
		if !sessionFound && resp.Found {
			sessionFound = true
			created <- &gamesession.Session{
				SessionID: gamesession.SessionID(resp.Session.SessionId),
				// TODO: State
				GameID:            resp.Session.GameId,
				AllocatedServerID: resp.Session.AllocatedServerId,
			}
		} else if sessionFound && !resp.Found {
			sessionFound = false
			deleted <- struct{}{}
		}
	}
}
