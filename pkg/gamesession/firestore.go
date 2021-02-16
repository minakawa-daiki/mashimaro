package gamesession

import (
	"context"

	"google.golang.org/api/iterator"

	"cloud.google.com/go/firestore"
	"github.com/google/uuid"
)

const (
	firestoreCollection = "gameSessions"
)

type FirestoreStore struct {
	c          *firestore.Client
	collection string
}

func NewFirestoreStore(c *firestore.Client) *FirestoreStore {
	return &FirestoreStore{
		c:          c,
		collection: firestoreCollection,
	}
}

func (s *FirestoreStore) NewSession(ctx context.Context, req *NewSessionRequest) (*Session, error) {
	sid := SessionID(uuid.Must(uuid.NewRandom()).String())
	ss := &Session{
		SessionID:      sid,
		State:          StateWaitingForSession,
		GameID:         req.GameID,
		GameServerName: req.GameServer.Name,
	}
	if _, _, err := s.c.Collection(s.collection).Add(ctx, ss); err != nil {
		return nil, err
	}
	return ss, nil
}

func (s *FirestoreStore) GetSession(ctx context.Context, sid SessionID) (*Session, error) {
	ds, err := s.c.Collection(s.collection).Where("sessionId", "==", sid).Documents(ctx).Next()
	if err == iterator.Done {
		return nil, ErrSessionNotFound
	}
	if err != nil {
		return nil, err
	}
	if !ds.Exists() {
		return nil, ErrSessionNotFound
	}
	var ss Session
	if err := ds.DataTo(&ss); err != nil {
		return nil, err
	}
	return &ss, nil
}

func (s *FirestoreStore) GetSessionByGameServerName(ctx context.Context, gsName string) (*Session, error) {
	ds, err := s.c.Collection(s.collection).Where("gameServerName", "==", gsName).Documents(ctx).Next()
	if err == iterator.Done {
		return nil, ErrSessionNotFound
	}
	if err != nil {
		return nil, err
	}
	if !ds.Exists() {
		return nil, ErrSessionNotFound
	}
	var ss Session
	if err := ds.DataTo(&ss); err != nil {
		return nil, err
	}
	return &ss, nil
}

func (s *FirestoreStore) UpdateSessionState(ctx context.Context, sid SessionID, newState State) error {
	ds, err := s.c.Collection(s.collection).Where("sessionId", "==", sid).Documents(ctx).Next()
	if err == iterator.Done {
		return ErrSessionNotFound
	}
	if err != nil {
		return err
	}
	if !ds.Exists() {
		return ErrSessionNotFound
	}
	var ss Session
	if err := ds.DataTo(&ss); err != nil {
		return err
	}
	if _, err := ds.Ref.Update(ctx, []firestore.Update{
		{Path: "state", Value: newState},
	}); err != nil {
		return err
	}
	return nil
}
