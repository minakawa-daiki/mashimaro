package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"

	"cloud.google.com/go/firestore"
	"github.com/castaneai/mashimaro/pkg/broker"
	"github.com/castaneai/mashimaro/pkg/gamemetadata"
	"github.com/castaneai/mashimaro/pkg/gamesession"
	"github.com/castaneai/mashimaro/pkg/proto"
	"github.com/kelseyhightower/envconfig"
	"google.golang.org/grpc"
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

	ctx := context.Background()
	projectID := "mashimaro"
	if p := os.Getenv("GOOGLE_CLOUD_PROJECT"); p != "" {
		projectID = p
	}
	fc, err := firestore.NewClient(ctx, projectID)
	if err != nil {
		log.Fatalf("failed to new firestore client: %+v", err)
	}
	sessionStore := gamesession.NewFirestoreStore(fc)
	metadataStore := gamemetadata.NewFirestoreStore(fc)
	s := grpc.NewServer()
	proto.RegisterBrokerServer(s, broker.NewInternalServer(sessionStore, metadataStore))

	addr := fmt.Sprintf(":%s", conf.Port)
	lis, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatalf("failed to listen %s: %+v", addr, err)
	}
	log.Printf("mashimaro internal broker is listening on %s...", addr)
	log.Fatal(s.Serve(lis))
}
