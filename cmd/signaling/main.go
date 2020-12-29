package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"

	"github.com/castaneai/mashimaro/pkg/gamesession"
	"github.com/castaneai/mashimaro/pkg/signaling"
)

func main() {
	streamerAddr := "streamer:50501" // TODO: Agones
	if sa := os.Getenv("STREAMER_ADDR"); sa != "" {
		streamerAddr = sa
	}
	allocator := &gamesession.MockAllocator{MockedGS: &gamesession.GameServer{Addr: streamerAddr}}
	gsManager := gamesession.NewManager(allocator)
	sv := signaling.NewServer(gsManager)
	http.Handle("/signal", sv.WebSocketHandler())

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		f, err := os.Open("static/index.html")
		if err != nil {
			respondError(w, err)
			return
		}
		defer f.Close()
		b, err := ioutil.ReadAll(f)
		if err != nil {
			respondError(w, err)
			return
		}
		w.Header().Set("content-type", "text/html")
		if _, err := w.Write(b); err != nil {
			respondError(w, err)
			return
		}
	})

	addr := ":8080"
	if p := os.Getenv("PORT"); p != "" {
		addr = fmt.Sprintf(":%s", p)
	}
	log.Printf("mashimaro signaling server is listening on %s...", addr)
	log.Fatal(http.ListenAndServe(addr, nil))
}

func respondError(w http.ResponseWriter, err error) {
	log.Printf("error: %+v", err)
	http.Error(w, err.Error(), http.StatusInternalServerError)
}
