package main

import (
	"fmt"
	"log"
	"net"
	"net/http"
	"os"

	"golang.org/x/sync/errgroup"

	"google.golang.org/grpc"

	"github.com/castaneai/mashimaro/pkg/broker"
	"github.com/castaneai/mashimaro/pkg/gameserver"
	"github.com/castaneai/mashimaro/pkg/gamesession"
	"github.com/castaneai/mashimaro/pkg/proto"
)

func main() {
	sessionStore := gamesession.NewInMemoryStore()
	allocator, err := newAllocator()
	if err != nil {
		log.Fatalf("failed to new allocator: %+v", err)
	}
	b := broker.NewBroker(sessionStore, allocator)

	eg := &errgroup.Group{}
	eg.Go(func() error {
		return startInternalServer(sessionStore)
	})
	eg.Go(func() error {
		return startExternalServer(b)
	})
	log.Fatal(eg.Wait())
}

func startInternalServer(store gamesession.Store) error {
	s := grpc.NewServer()
	proto.RegisterBrokerServer(s, broker.NewInternalServer(store))

	addr := ":50501"
	if p := os.Getenv("INTERNAL_PORT"); p != "" {
		addr = fmt.Sprintf(":%s", p)
	}
	lis, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}
	log.Printf("mashimaro internal broker server is listening on %s...", addr)
	if err := s.Serve(lis); err != nil {
		return fmt.Errorf("failed to server gRPC: %+v", err)
	}
	return nil
}

func startExternalServer(b *broker.Broker) error {
	http.Handle("/", broker.ExternalServer(b))
	addr := ":8080"
	if p := os.Getenv("PORT"); p != "" {
		addr = fmt.Sprintf(":%s", p)
	}
	log.Printf("mashimaro external broker server is listening on %s...", addr)
	return http.ListenAndServe(addr, nil)
}

func newAllocator() (gameserver.Allocator, error) {
	if sa := os.Getenv("GAMESERVER_ADDR"); sa != "" {
		return &gameserver.MockAllocator{MockedGS: &gameserver.GameServer{Addr: sa}}, nil
	}

	addr := "agones-allocator.agones-system.svc.cluster.local.:443"
	// TODO: current namespace from k8s
	return gameserver.NewAgonesAllocator(addr, "mashimaro"), nil
}
