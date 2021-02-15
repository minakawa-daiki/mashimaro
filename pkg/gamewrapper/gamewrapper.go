package gamewrapper

import (
	"log"
	"net"
	"os"
	"os/exec"

	"github.com/castaneai/mashimaro/pkg/proto"
	"google.golang.org/grpc"
)

type GameWrapper struct{}

func NewGameWrapper() *GameWrapper {
	return &GameWrapper{}
}

func (w *GameWrapper) Run(lis net.Listener) error {
	startGameCh := make(chan *proto.StartGameRequest)
	sv := grpc.NewServer()
	proto.RegisterGameWrapperServer(sv, &gameWrapperServer{startGameCh: startGameCh})
	go func() {
		sv.Serve(lis)
	}()

	log.Printf("waiting for game start...")
	startReq := <-startGameCh
	log.Printf("starting game: %+v", startReq)
	cmd := exec.Command(startReq.Command, startReq.Args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
