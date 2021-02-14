package gamewrapper

import (
	"context"
	"log"

	"github.com/castaneai/mashimaro/pkg/proto"
)

type gameWrapperServer struct {
	startGameCh chan *proto.StartGameRequest
}

func (s *gameWrapperServer) StartGame(ctx context.Context, req *proto.StartGameRequest) (*proto.StartGameResponse, error) {
	log.Printf("received start game request: %+v", req)
	s.startGameCh <- req
	return &proto.StartGameResponse{}, nil
}
