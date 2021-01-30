package main

import (
	"log"

	"github.com/castaneai/mashimaro/pkg/gamewrapper"
)

func main() {
	w := gamewrapper.GameWrapper{}
	log.Fatal(w.Run())
}
