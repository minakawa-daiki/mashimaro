package main

import (
	"fmt"
	"log"
	"net"

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
	w := gamewrapper.GameWrapper{}
	log.Fatal(w.Run(lis))
}
