package gameserver

import (
	"context"
	"time"

	"github.com/castaneai/mashimaro/pkg/gamesession"

	"github.com/castaneai/mashimaro/pkg/proto"
)

func (s *GameServer) startWatchSession(ctx context.Context, created chan<- *gamesession.Session, deleted chan<- struct{}) error {
	ticker := time.NewTicker(1 * time.Second)
	sessionFound := false
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			resp, err := s.broker.FindSession(ctx, &proto.FindSessionRequest{AllocatedServerId: s.allocatedServer.ID})
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
}
