package broker

import (
	"context"

	"google.golang.org/grpc/codes"

	"google.golang.org/grpc/status"

	"github.com/castaneai/mashimaro/pkg/gamemetadata"
	"github.com/castaneai/mashimaro/pkg/gamesession"
	"github.com/castaneai/mashimaro/pkg/proto"
	"github.com/goccy/go-yaml"
)

type internalServer struct {
	sessionStore  gamesession.Store
	metadataStore gamemetadata.Store
}

func NewInternalServer(sessionStore gamesession.Store, metadataStore gamemetadata.Store) *internalServer {
	return &internalServer{
		sessionStore:  sessionStore,
		metadataStore: metadataStore,
	}
}

func (s *internalServer) FindSession(ctx context.Context, req *proto.FindSessionRequest) (*proto.FindSessionResponse, error) {
	ss, err := s.sessionStore.GetSessionByGameServerName(ctx, req.GameserverName)
	if err == gamesession.ErrSessionNotFound {
		return &proto.FindSessionResponse{Found: false}, nil
	}
	if err != nil {
		return nil, err
	}
	metadata, err := s.metadataStore.GetGameMetadata(ctx, ss.GameID)
	if err == gamemetadata.ErrMetadataNotFound {
		return nil, status.Newf(codes.FailedPrecondition, "game metadata not found(gameID: %s)", ss.GameID).Err()
	}
	if err != nil {
		return nil, err
	}
	metadataBody, err := yaml.Marshal(metadata)
	if err != nil {
		return nil, err
	}
	if err := s.sessionStore.UpdateSessionState(ctx, ss.SessionID, gamesession.StateSignaling); err != nil {
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
