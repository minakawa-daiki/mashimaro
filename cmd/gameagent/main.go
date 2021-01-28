package main

import (
	"context"
	"log"
	"os"
	"time"

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

	brokerCC, err := grpc.Dial("broker:50501", grpc.WithInsecure())
	if err != nil {
		log.Fatalf("failed to dial to broker: %+v", err)
	}
	brokerClient := proto.NewBrokerClient(brokerCC)

	signalingCC, err := grpc.Dial("signaling:50502", grpc.WithInsecure())
	if err != nil {
		log.Fatalf("failed to dial to signaling: %+v", err)
	}
	signalingClient := proto.NewSignalingClient(signalingCC)

	agent := gameagent.NewAgent(brokerClient, signalingClient)
	ctx := context.Background()
	if err := agent.Run(ctx, gameServerName); err != nil {
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
