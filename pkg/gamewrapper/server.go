package gamewrapper

import (
	"context"

	"github.com/castaneai/mashimaro/pkg/proto"
)

type gameWrapperServer struct {
	startGameCh chan *proto.StartGameRequest
}

func (s *gameWrapperServer) StartGame(ctx context.Context, req *proto.StartGameRequest) (*proto.StartGameResponse, error) {
	s.startGameCh <- req
	return &proto.StartGameResponse{}, nil
}
