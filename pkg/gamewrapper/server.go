package gamewrapper

import (
	"context"
	"log"
	"os"
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
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
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
	log.Printf("trying to kill game process")
	if err := s.processWatcher.KillProcess(); err != nil {
		log.Printf("failed to kill game process: %+v", err)
	}
	return &proto.ExitGameResponse{}, nil
}

func (s *gameWrapperServer) HealthCheck(ctx context.Context, request *proto.HealthCheckRequest) (*proto.HealthCheckResponse, error) {
	healthy := s.processWatcher.IsLiving()
	if !healthy {
		log.Printf("game process is unhealthy!")
	}
	return &proto.HealthCheckResponse{Healthy: healthy}, nil
}

func (s *gameWrapperServer) ListenCaptureArea(req *proto.ListenCaptureAreaRequest, stream proto.GameWrapper_ListenCaptureAreaServer) error {
	for {
		select {
		case <-stream.Context().Done():
			return stream.Context().Err()
		case area := <-s.processWatcher.AreaChanged():
			if err := stream.Send(&proto.ListenCaptureAreaResponse{
				StartX: uint32(area.startX),
				StartY: uint32(area.startY),
				EndX:   uint32(area.endX),
				EndY:   uint32(area.endY),
			}); err != nil {
				return err
			}
		}
	}
}
