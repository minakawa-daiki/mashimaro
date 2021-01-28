package game

import (
	"context"

	"github.com/goccy/go-yaml"
)

type MetadataStore interface {
	GetGameMetadata(ctx context.Context, gameID string) (*Metadata, error)
}

type MockMetadataStore struct {
}

func (s *MockMetadataStore) GetGameMetadata(ctx context.Context, gameID string) (*Metadata, error) {
	body := []byte(`
gameId: microkiri
command: wine /microkiri/microkiri.exe
`)
	var metadata Metadata
	if err := yaml.Unmarshal(body, &metadata); err != nil {
		return nil, err
	}
	return &metadata, nil
}
