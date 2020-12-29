package main

import (
	"fmt"
	"log"
	"net"
	"os"

	"github.com/castaneai/mashimaro/pkg/streamer"

	"github.com/castaneai/mashimaro/pkg/gameserver"
	"github.com/castaneai/mashimaro/pkg/proto"
	"google.golang.org/grpc"
)

func main() {
	videoSrc, audioSrc, err := newMediaSources()
	if err != nil {
		log.Fatalf("failed to new media sources: %+v", err)
	}

	s := grpc.NewServer()
	proto.RegisterGameServerServer(s, gameserver.NewGameServerService(videoSrc, audioSrc))

	addr := ":50501"
	if p := os.Getenv("PORT"); p != "" {
		addr = fmt.Sprintf(":%s", p)
	}
	lis, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatalf("failed to listen: %+v", err)
	}

	log.Printf("mashimaro streamer server is listening on %s...", addr)
	log.Fatal(s.Serve(lis))
}

func newMediaSources() (videoSrc, audioSrc streamer.MediaStream, err error) {
	if os.Getenv("USE_TEST_MEDIA_SOURCE") != "" {
		videoSrc, err = streamer.NewVideoTestStream()
		if err != nil {
			return
		}
		audioSrc, err = streamer.NewAudioTestStream()
		return
	}

	videoSrc, err = streamer.NewX11VideoStream(":0")
	if err != nil {
		return
	}
	audioSrc, err = streamer.NewPulseAudioStream("localhost:4713")
	return
}
