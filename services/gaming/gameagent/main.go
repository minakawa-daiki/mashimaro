package main

import (
	"context"
	"log"
	"os"
	"time"

	"github.com/castaneai/mashimaro/pkg/streamer"

	"github.com/castaneai/mashimaro/pkg/proto"
	"google.golang.org/grpc"

	"github.com/castaneai/mashimaro/pkg/gameagent"

	sdk "agones.dev/agones/sdks/go"
)

func main() {
	gameServerName := "dummy"
	if isRunningOnKubernetes() {
		agones := setupAgones()
		gs, err := agones.GameServer()
		if err != nil {
			log.Fatalf("failed to get agones GameServer: %+v", err)
		}
		gameServerName = gs.ObjectMeta.Name
	}

	brokerAddr := "broker:50501"
	if a := os.Getenv("BROKER_ADDR"); a != "" {
		brokerAddr = a
	}
	brokerCC, err := grpc.Dial(brokerAddr, grpc.WithInsecure())
	if err != nil {
		log.Fatalf("failed to dial to broker: %+v", err)
	}
	brokerClient := proto.NewBrokerClient(brokerCC)

	signalingAddr := "signaling:50502"
	if a := os.Getenv("SIGNALING_ADDR"); a != "" {
		signalingAddr = a
	}
	signalingCC, err := grpc.Dial(signalingAddr, grpc.WithInsecure())
	if err != nil {
		log.Fatalf("failed to dial to signaling: %+v", err)
	}
	signalingClient := proto.NewSignalingClient(signalingCC)

	agent := gameagent.NewAgent(brokerClient, signalingClient)
	ctx := context.Background()
	tracks, err := gameagent.NewMediaTracks()
	if err != nil {
		log.Fatalf("failed to get media tracks: %+v", err)
	}
	if err := agent.Run(ctx, gameServerName, tracks); err != nil {
		log.Fatalf("failed to run agent: %+v", err)
	}
}

func isRunningOnKubernetes() bool {
	_, err := os.Stat("/var/run/secrets/kubernetes.io")
	return !os.IsNotExist(err)
}

func setupAgones() *sdk.SDK {
	agones, err := sdk.NewSDK()
	if err != nil {
		log.Fatalf("failed to new Agones SDK: %+v", err)
	}
	if err := agones.Ready(); err != nil {
		log.Fatalf("failed to ready to Agones SDK: %+v", err)
	}
	log.Printf("connected to Agones SDK")
	go doHealth(agones)
	return agones
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
