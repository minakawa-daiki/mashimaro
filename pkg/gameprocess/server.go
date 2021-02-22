package gameprocess

import (
	"context"
	"log"
	"os"
	"os/exec"
	"sync"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/pkg/errors"

	"github.com/castaneai/mashimaro/pkg/proto"
)

type gameProcessServer struct {
	pid   int
	pidMu sync.Mutex
}

func NewGameProcessServer() proto.GameProcessServer {
	return &gameProcessServer{}
}

func (s *gameProcessServer) StartGame(ctx context.Context, req *proto.StartGameRequest) (*proto.StartGameResponse, error) {
	// TODO: provisioning game data and ready to start process

	log.Printf("starting game: %+v", req)
	cmd := exec.Command(req.Command, req.Args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		return nil, errors.Wrap(err, "failed to start process")
	}
	s.pidMu.Lock()
	s.pid = cmd.Process.Pid
	s.pidMu.Unlock()
	return &proto.StartGameResponse{}, nil
}

func (s *gameProcessServer) ExitGame(ctx context.Context, req *proto.ExitGameRequest) (*proto.ExitGameResponse, error) {
	log.Printf("on exit game request")
	s.pidMu.Lock()
	pid := s.pid
	s.pidMu.Unlock()
	if pid == 0 {
		log.Printf("pid is 0; start process before")
		return nil, status.Error(codes.NotFound, "process not found")
	}
	p, err := os.FindProcess(pid)
	if err != nil {
		return nil, status.Error(codes.NotFound, "process not found")
	}
	log.Printf("trying to kill game process")
	_ = p.Kill()
	return &proto.ExitGameResponse{}, nil
}
