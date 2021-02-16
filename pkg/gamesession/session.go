package gamesession

type SessionID string

type Session struct {
	SessionID      SessionID `firestore:"sessionId"`
	State          State     `firestore:"state"`
	GameID         string    `firestore:"gameId"`
	GameServerName string    `firestore:"gameServerName"`
}
