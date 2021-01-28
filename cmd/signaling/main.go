package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"os"

	"golang.org/x/sync/errgroup"

	"github.com/castaneai/mashimaro/pkg/proto"
	"google.golang.org/grpc"

	"github.com/castaneai/mashimaro/pkg/gamesession"

	"github.com/castaneai/mashimaro/pkg/signaling"
)

func main() {
	sessionStore := gamesession.NewInMemoryStore()
	channels := signaling.NewChannels()
	eg := &errgroup.Group{}
	eg.Go(func() error {
		return startInternalServer(sessionStore, channels)
	})
	eg.Go(func() error {
		return startExternalServer(sessionStore, channels)
	})
	log.Fatal(eg.Wait())
}

func startInternalServer(store gamesession.Store, channels signaling.Channels) error {
	sv := grpc.NewServer()
	proto.RegisterSignalingServer(sv, signaling.NewInternalServer(store, channels))

	addr := ":50502"
	if p := os.Getenv("GRPC_PORT"); p != "" {
		addr = fmt.Sprintf(":%s", p)
	}
	log.Printf("mashimaro internal signaling server is listening on %s...", addr)
	lis, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}
	return sv.Serve(lis)
}

func startExternalServer(store gamesession.Store, channels signaling.Channels) error {
	sv := signaling.NewExternalServer(store, channels)
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
	log.Printf("mashimaro external signaling server is listening on %s...", addr)
	return http.ListenAndServe(addr, nil)
}

func respondError(w http.ResponseWriter, err error) {
	log.Printf("error: %+v", err)
	http.Error(w, err.Error(), http.StatusInternalServerError)
}
