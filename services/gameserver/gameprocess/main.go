package main

import (
	"fmt"
	"log"
	"net"

	"github.com/castaneai/mashimaro/pkg/proto"
	"google.golang.org/grpc"

	"github.com/castaneai/mashimaro/pkg/gameprocess"
	"github.com/kelseyhightower/envconfig"
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
	proto.RegisterGameProcessServer(s, gameprocess.NewGameProcessServer())
	log.Fatal(s.Serve(lis))
}
