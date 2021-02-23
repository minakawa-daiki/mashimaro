package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/castaneai/mashimaro/pkg/allocator"

	"github.com/castaneai/mashimaro/pkg/broker"

	"github.com/castaneai/mashimaro/pkg/gamemetadata"
	"github.com/castaneai/mashimaro/pkg/gamesession"
	"github.com/kelseyhightower/envconfig"

	"cloud.google.com/go/firestore"
)

type config struct {
	Port             string `envconfig:"PORT"`
	UseMockAllocator bool   `envconfig:"USE_MOCK_ALLOCATOR" default:"false"`
	AllocatorAddr    string `envconfig:"ALLOCATOR_ADDR" default:"agones-allocator.agones-system.svc.cluster.local.:443"`
	FleetNamespace   string `envconfig:"FLEET_NAMESPACE" default:"mashimaro"`
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
	allocator, err := newAllocator(&conf)
	if err != nil {
		log.Fatalf("failed to new allocator: %+v", err)
	}
	s := broker.NewExternalBroker(sessionStore, metadataStore, allocator)
	http.Handle("/", s.HTTPHandler())
	addr := fmt.Sprintf(":%s", conf.Port)
	log.Printf("mashimaro external broker is listening on %s...", addr)
	log.Fatal(http.ListenAndServe(addr, nil))
}

func newAllocator(conf *config) (allocator.Allocator, error) {
	if conf.UseMockAllocator {
		return &allocator.MockAllocator{MockedGS: &allocator.AllocatedServer{ID: "dummy"}}, nil
	}
	return allocator.NewAgonesAllocator(conf.AllocatorAddr, conf.FleetNamespace), nil
}
