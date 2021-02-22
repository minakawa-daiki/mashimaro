package main

import (
	"fmt"
	"log"
	"net"

	"github.com/castaneai/mashimaro/pkg/streamer/streamerserver"

	"github.com/castaneai/mashimaro/pkg/proto"
	"github.com/kelseyhightower/envconfig"
	"google.golang.org/grpc"
)

type config struct {
	Port string `envconfig:"PORT" required:"true"`
}

func main() {
	var conf config
	if err := envconfig.Process("", &conf); err != nil {
		log.Fatalf("failed to process config: %+v", err)
	}
	log.Printf("load config: %+v", conf)

	addr := fmt.Sprintf(":%s", conf.Port)
	lis, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatal(err)
	}
	s := grpc.NewServer()
	proto.RegisterStreamerServer(s, streamerserver.NewStreamerServer())
	log.Fatal(s.Serve(lis))
}
