package gamesession

import "github.com/castaneai/mashimaro/pkg/gameserver"

type SessionID string

type Session struct {
	SessionID  SessionID
	State      State
	GameID     string
	GameServer *gameserver.GameServer
}
