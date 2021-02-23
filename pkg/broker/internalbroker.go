package broker

import (
	"context"

	"github.com/pkg/errors"

	"google.golang.org/grpc/codes"

	"google.golang.org/grpc/status"

	"github.com/castaneai/mashimaro/pkg/gamemetadata"
	"github.com/castaneai/mashimaro/pkg/gamesession"
	"github.com/castaneai/mashimaro/pkg/proto"
)

type internalBroker struct {
	sessionStore  gamesession.Store
	metadataStore gamemetadata.Store
}

func NewInternalBroker(sessionStore gamesession.Store, metadataStore gamemetadata.Store) *internalBroker {
	return &internalBroker{
		sessionStore:  sessionStore,
		metadataStore: metadataStore,
	}
}

func (s *internalBroker) FindSession(ctx context.Context, req *proto.FindSessionRequest) (*proto.FindSessionResponse, error) {
	if req.AllocatedServerId == "" {
		return nil, status.Error(codes.FailedPrecondition, "invalid allocated server ID")
	}
	ss, err := s.sessionStore.GetSessionByAllocatedServerID(ctx, req.AllocatedServerId)
	if err == gamesession.ErrSessionNotFound {
		return &proto.FindSessionResponse{
			Found:   false,
			Session: nil,
		}, nil
	}
	if err != nil {
		return nil, err
	}
	return &proto.FindSessionResponse{
		Found: true,
		Session: &proto.Session{
			SessionId:         string(ss.SessionID),
			AllocatedServerId: ss.AllocatedServerID,
			GameId:            ss.GameID,
		},
	}, nil
}

func (s *internalBroker) GetGameMetadata(ctx context.Context, req *proto.GetGameMetadataRequest) (*proto.GetGameMetadataResponse, error) {
	metadata, err := s.metadataStore.GetGameMetadata(ctx, req.GameId)
	if err == gamemetadata.ErrMetadataNotFound {
		return nil, status.Errorf(codes.FailedPrecondition, "game metadata not found(gameID: %s)", req.GameId)
	}
	if err != nil {
		return nil, err
	}
	metadataBody, err := gamemetadata.Marshal(metadata)
	if err != nil {
		return nil, err
	}
	return &proto.GetGameMetadataResponse{GameMetadata: &proto.GameMetadata{Body: string(metadataBody)}}, nil
}

func (s *internalBroker) DeleteSession(ctx context.Context, req *proto.DeleteSessionRequest) (*proto.DeleteSessionResponse, error) {
	sid := gamesession.SessionID(req.SessionId)
	ss, err := s.sessionStore.GetSession(ctx, sid)
	if errors.Is(err, gamesession.ErrSessionNotFound) {
		return nil, status.Error(codes.NotFound, "game session not found")
	}
	if ss.AllocatedServerID != req.AllocatedServerId {
		return nil, status.Error(codes.FailedPrecondition, "invalid allocated server ID")
	}
	if err != nil {
		return nil, err
	}
	if err := s.sessionStore.DeleteSession(ctx, sid); err != nil {
		return nil, err
	}
	return &proto.DeleteSessionResponse{}, nil
}
