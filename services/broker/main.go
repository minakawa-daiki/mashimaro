package main

import (
	"fmt"
	"log"
	"net"
	"net/http"

	"github.com/kelseyhightower/envconfig"

	"golang.org/x/sync/errgroup"

	"google.golang.org/grpc"

	"github.com/castaneai/mashimaro/pkg/broker"
	"github.com/castaneai/mashimaro/pkg/gameserver"
	"github.com/castaneai/mashimaro/pkg/gamesession"
	"github.com/castaneai/mashimaro/pkg/proto"
)

type config struct {
	AllocatorAddr    string `envconfig:"ALLOCATOR_ADDR" default:"agones-allocator.agones-system.svc.cluster.local.:443"`
	FleetNamespace   string `envconfig:"FLEET_NAMESPACE" default:"mashimaro"`
	UseMockAllocator bool   `envconfig:"USE_MOCK_ALLOCATOR" default:"false"`
	InternalPort     string `envconfig:"INTERNAL_PORT" default:"50501"`
	ExternalPort     string `envconfig:"EXTERNAL_PORT" default:"8081"`
}

func main() {
	var conf config
	if err := envconfig.Process("", &conf); err != nil {
		log.Fatalf("failed to process config: %+v", err)
	}
	log.Printf("load config: %+v", conf)

	// TODO: persistent session store
	sessionStore := gamesession.NewInMemoryStore()
	allocator, err := newAllocator(&conf)
	if err != nil {
		log.Fatalf("failed to new allocator: %+v", err)
	}
	b := broker.NewBroker(sessionStore, allocator)

	eg := &errgroup.Group{}
	eg.Go(func() error {
		return startInternalServer(sessionStore, &conf)
	})
	eg.Go(func() error {
		return startExternalServer(b, &conf)
	})
	log.Fatal(eg.Wait())
}

func startInternalServer(store gamesession.Store, conf *config) error {
	s := grpc.NewServer()
	proto.RegisterBrokerServer(s, broker.NewInternalServer(store))

	addr := fmt.Sprintf(":%s", conf.InternalPort)
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

func startExternalServer(b *broker.Broker, conf *config) error {
	http.Handle("/", broker.ExternalServer(b))
	addr := fmt.Sprintf(":%s", conf.ExternalPort)
	log.Printf("mashimaro external broker server is listening on %s...", addr)
	return http.ListenAndServe(addr, nil)
}

func newAllocator(conf *config) (gameserver.Allocator, error) {
	if conf.UseMockAllocator {
		return &gameserver.MockAllocator{MockedGS: &gameserver.GameServer{Name: "dummy", Addr: "dummy"}}, nil
	}
	return gameserver.NewAgonesAllocator(conf.AllocatorAddr, conf.FleetNamespace), nil
}
