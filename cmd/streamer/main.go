package main

import (
	"fmt"
	"log"
	"net"
	"os"
	"time"

	"github.com/castaneai/mashimaro/pkg/streamer"

	sdk "agones.dev/agones/sdks/go"
	"github.com/castaneai/mashimaro/pkg/gameserver"
	"github.com/castaneai/mashimaro/pkg/proto"
	"google.golang.org/grpc"
)

func main() {
	if isRunningOnKubernetes() {
		setupAgones()
	}

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

func isRunningOnKubernetes() bool {
	_, err := os.Stat("/var/run/secrets/kubernetes.io")
	return !os.IsNotExist(err)
}

func setupAgones() {
	agones, err := sdk.NewSDK()
	if err != nil {
		log.Fatalf("failed to new Agones SDK: %+v", err)
	}
	if err := agones.Ready(); err != nil {
		log.Fatalf("failed to ready to Agones SDK: %+v", err)
	}
	log.Printf("connected to Agones SDK")
	go doHealth(agones)
}

func doHealth(agones *sdk.SDK) {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()
	for range ticker.C {
		if err := agones.Health(); err != nil {
			log.Printf("failed to health to Agones SDK: %+v", err)
			break
		}
	}
}
