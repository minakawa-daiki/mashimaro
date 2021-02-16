package main

import (
	"fmt"
	"log"
	"net"

	"github.com/castaneai/mashimaro/pkg/proto"
	"google.golang.org/grpc"

	"github.com/castaneai/mashimaro/pkg/gamewrapper"
	"github.com/kelseyhightower/envconfig"
)

type config struct {
	Port string `envconfig:"PORT" default:"50501"`
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
	proto.RegisterGameWrapperServer(s, gamewrapper.NewGameWrapperServer())
	log.Fatal(s.Serve(lis))
}
