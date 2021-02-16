package gamewrapper

import (
	"context"
	"log"
	"os"
	"os/exec"
	"sync"

	"github.com/tevino/abool"

	"github.com/pkg/errors"

	"github.com/castaneai/mashimaro/pkg/proto"
)

type gameWrapperServer struct {
	pid    int
	mu     sync.Mutex
	exited *abool.AtomicBool
}

func NewGameWrapperServer() proto.GameWrapperServer {
	return &gameWrapperServer{
		exited: abool.New(),
	}
}

func (s *gameWrapperServer) StartGame(ctx context.Context, req *proto.StartGameRequest) (*proto.StartGameResponse, error) {
	log.Printf("starting game: %+v", req)
	cmd := exec.Command(req.Command, req.Args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		s.exited.Set()
		log.Printf("failed to start game process: %+v", err)
		return nil, err
	}
	s.mu.Lock()
	s.pid = cmd.Process.Pid
	s.mu.Unlock()
	go func() {
		_ = cmd.Wait()
		exitCode := cmd.ProcessState.ExitCode()
		log.Printf("game process exited with code: %v", exitCode)
		s.exited.Set()
	}()
	return &proto.StartGameResponse{}, nil
}

func (s *gameWrapperServer) ExitGame(ctx context.Context, req *proto.ExitGameRequest) (*proto.ExitGameResponse, error) {
	s.mu.Lock()
	pid := s.pid
	s.mu.Unlock()
	if pid > 0 {
		if err := killProcess(pid); err != nil {
			log.Printf("failed to kill process: %+v", err)
		}
	} else {
		log.Printf("game process not found, skip")
	}
	return &proto.ExitGameResponse{}, nil
}

func (s *gameWrapperServer) HealthCheck(ctx context.Context, request *proto.HealthCheckRequest) (*proto.HealthCheckResponse, error) {
	return &proto.HealthCheckResponse{Healthy: s.exited.IsNotSet()}, nil
}

func killProcess(pid int) error {
	p, err := os.FindProcess(pid)
	if err != nil {
		return errors.Wrapf(err, "failed to find process(pid: %d)", pid)
	}
	if err := p.Signal(os.Interrupt); err != nil {
		return errors.Wrap(err, "failed to interrupt game process")
	}
	return nil
}
