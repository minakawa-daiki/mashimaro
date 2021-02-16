package gamemetadata

import (
	"context"

	"google.golang.org/api/iterator"

	"cloud.google.com/go/firestore"
)

const (
	FirestoreCollection = "gameMetadata"
)

type FirestoreStore struct {
	c          *firestore.Client
	collection string
}

func NewFirestoreStore(c *firestore.Client) *FirestoreStore {
	return &FirestoreStore{
		c:          c,
		collection: FirestoreCollection,
	}
}

func (s *FirestoreStore) GetGameMetadata(ctx context.Context, gameID string) (*Metadata, error) {
	ds, err := s.c.Collection(s.collection).Where("gameId", "==", gameID).Documents(ctx).Next()
	if err == iterator.Done {
		return nil, ErrMetadataNotFound
	}
	if err != nil {
		return nil, err
	}
	if !ds.Exists() {
		return nil, ErrMetadataNotFound
	}
	var metadata Metadata
	if err := ds.DataTo(&metadata); err != nil {
		return nil, err
	}
	return &metadata, nil
}
