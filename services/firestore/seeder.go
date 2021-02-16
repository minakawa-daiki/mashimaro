package main

import (
	"context"
	"io/ioutil"
	"log"
	"os"

	"github.com/castaneai/mashimaro/pkg/gamemetadata"
	"github.com/goccy/go-yaml"

	"cloud.google.com/go/firestore"
)

type seeds struct {
	Seeds []*gamemetadata.Metadata `yaml:"seeds"`
}

func main() {
	seedFile := os.Args[1]
	ctx := context.Background()
	fs, err := firestore.NewClient(ctx, "mashimaro")
	if err != nil {
		log.Fatalf("failed to new firestore client: %+v", err)
	}
	b, err := ioutil.ReadFile(seedFile)
	if err != nil {
		log.Fatalf("failed to read seed file: %+v", err)
	}
	var seeds seeds
	if err := yaml.Unmarshal(b, &seeds); err != nil {
		log.Fatalf("failed to unmarshal yaml: %+v", err)
	}
	for _, md := range seeds.Seeds {
		if _, _, err := fs.Collection(gamemetadata.FirestoreCollection).Add(ctx, md); err != nil {
			return
		}
		log.Printf("added %s", md.GameID)
	}
}
