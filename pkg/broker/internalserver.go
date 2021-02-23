package broker

import (
	"context"
	"log"
	"time"

	"github.com/pkg/errors"

	"google.golang.org/grpc/codes"

	"google.golang.org/grpc/status"

	"github.com/castaneai/mashimaro/pkg/gamemetadata"
	"github.com/castaneai/mashimaro/pkg/gamesession"
	"github.com/castaneai/mashimaro/pkg/proto"
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

func (s *internalServer) WatchSession(req *proto.WatchSessionRequest, stream proto.Broker_WatchSessionServer) error {
	if req.AllocatedServerId == "" {
		return status.Error(codes.FailedPrecondition, "invalid allocated server ID")
	}
	ticker := time.NewTicker(1 * time.Second)
	found := false
	for {
		select {
		case <-stream.Context().Done():
			return nil
		case <-ticker.C:
			ss, err := s.sessionStore.GetSessionByAllocatedServerID(stream.Context(), req.AllocatedServerId)
			if err == gamesession.ErrSessionNotFound {
				if found {
					found = false
					if err := stream.Send(&proto.WatchSessionResponse{
						Found:   false,
						Session: nil,
					}); err != nil {
						return err
					}
				}
				continue
			}
			if err != nil {
				return err
			}
			if !found {
				found = true
				log.Printf("found gamesession for allocated server: %s", req.AllocatedServerId)
				if err := stream.Send(&proto.WatchSessionResponse{
					Found: found,
					Session: &proto.Session{
						SessionId:         string(ss.SessionID),
						AllocatedServerId: ss.AllocatedServerID,
						GameId:            ss.GameID,
					},
				}); err != nil {
					return err
				}
			}
		}
	}
}

func (s *internalServer) GetGameMetadata(ctx context.Context, req *proto.GetGameMetadataRequest) (*proto.GetGameMetadataResponse, error) {
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

func (s *internalServer) DeleteSession(ctx context.Context, req *proto.DeleteSessionRequest) (*proto.DeleteSessionResponse, error) {
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
