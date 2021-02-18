package gamewrapper

import (
	"context"
	"log"
	"os/exec"

	"github.com/pkg/errors"

	"github.com/castaneai/mashimaro/pkg/proto"
)

type gameWrapperServer struct {
	processWatcher *processWatcher
}

func NewGameWrapperServer() proto.GameWrapperServer {
	return &gameWrapperServer{}
}

func (s *gameWrapperServer) StartGame(ctx context.Context, req *proto.StartGameRequest) (*proto.StartGameResponse, error) {
	log.Printf("starting game: %+v", req)
	s.processWatcher = newProcessWatcher()
	cmd := exec.Command(req.Command, req.Args...)
	if err := cmd.Start(); err != nil {
		return nil, errors.Wrap(err, "failed to start process")
	}
	go func() {
		if err := s.processWatcher.Start(cmd); err != nil {
			log.Printf("failed to start process watcher: %+v", err)
		}
	}()
	return &proto.StartGameResponse{}, nil
}

func (s *gameWrapperServer) ExitGame(ctx context.Context, req *proto.ExitGameRequest) (*proto.ExitGameResponse, error) {
	if err := s.processWatcher.KillProcess(); err != nil {
		log.Printf("failed to kill game process: %+v", err)
	}
	return &proto.ExitGameResponse{}, nil
}

func (s *gameWrapperServer) HealthCheck(ctx context.Context, request *proto.HealthCheckRequest) (*proto.HealthCheckResponse, error) {
	return &proto.HealthCheckResponse{Healthy: s.processWatcher.IsLiving()}, nil
}
