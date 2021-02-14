package broker

import (
	"context"
	"log"

	"github.com/castaneai/mashimaro/pkg/game"
	"github.com/castaneai/mashimaro/pkg/gamesession"
	"github.com/castaneai/mashimaro/pkg/proto"
	"github.com/goccy/go-yaml"
)

type internalServer struct {
	sessionStore gamesession.Store
}

func NewInternalServer(sessionStore gamesession.Store) *internalServer {
	return &internalServer{sessionStore: sessionStore}
}

func (s *internalServer) FindSession(ctx context.Context, req *proto.FindSessionRequest) (*proto.FindSessionResponse, error) {
	log.Printf("Finding session by gsName: %s", req.GameserverName)
	ss, err := s.sessionStore.GetSessionByGameServerName(ctx, req.GameserverName)
	if err == gamesession.ErrSessionNotFound {
		return &proto.FindSessionResponse{Found: false}, nil
	}
	if err != nil {
		return nil, err
	}
	log.Printf("found session: %+v", ss)

	// TODO: metadata store
	log.Printf("Finding metadata by gameID: %s", ss.GameID)
	metadata, err := (&game.MockMetadataStore{}).GetGameMetadata(ctx, ss.GameID)
	if err != nil {
		return nil, err
	}
	metadataBody, err := yaml.Marshal(metadata)
	if err != nil {
		return nil, err
	}
	if _, err := s.sessionStore.UpdateSessionState(ctx, ss.SessionID, gamesession.StateSignaling); err != nil {
		return nil, err
	}
	return &proto.FindSessionResponse{
		Found: true,
		Session: &proto.Session{
			SessionId:    string(ss.SessionID),
			GameMetadata: &proto.GameMetadata{Body: string(metadataBody)},
		},
	}, nil
}
