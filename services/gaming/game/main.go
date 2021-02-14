package main

import (
	"log"
	"net"
	"os"

	"github.com/castaneai/mashimaro/pkg/gamewrapper"
)

func main() {
	addr := ":50051"
	if p := os.Getenv("PORT"); p != "" {
		addr = ":" + p
	}
	lis, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatal(err)
	}
	w := gamewrapper.GameWrapper{}
	log.Fatal(w.Run(lis))
}
