package main

import (
	"context"
	"log"
	"os"
	"time"

	grpc_retry "github.com/grpc-ecosystem/go-grpc-middleware/retry"

	"github.com/kelseyhightower/envconfig"

	"github.com/castaneai/mashimaro/pkg/proto"
	"google.golang.org/grpc"

	"github.com/castaneai/mashimaro/pkg/gameagent"

	sdk "agones.dev/agones/sdks/go"
)

type config struct {
	InternalBrokerAddr string `envconfig:"INTERNAL_BROKER_ADDR" default:"broker.mashimaro.svc.cluster.local.:50501"`
	GameWrapperAddr    string `envconfig:"GAME_WRAPPER_ADDR" default:"localhost.50501"`
	AyameAddr          string `envconfig:"AYAME_ADDR" default:"ws://ayame.mashimaro.svc.cluster.local.:3000./signal"`
	UseMockAllocator   bool   `envconfig:"USE_MOCK_ALLOCATOR" default:"false"`
	PulseAddr          string `envconfig:"PULSE_ADDR" default:"localhost:4713"`
	XDisplay           string `envconfig:"DISPLAY" default:":0"`
}

func main() {
	var conf config
	if err := envconfig.Process("", &conf); err != nil {
		log.Fatalf("failed to process config: %+v", err)
	}
	log.Printf("load config: %+v", conf)

	gameServerName := ""
	if conf.UseMockAllocator {
		gameServerName = "dummy"
	}
	if isRunningOnKubernetes() {
		agones := setupAgones()
		gs, err := agones.GameServer()
		if err != nil {
			log.Fatalf("failed to get agones GameServer: %+v", err)
		}
		gameServerName = gs.ObjectMeta.Name
	}
	if gameServerName == "" {
		log.Fatalf("gameServerName not set (Set `USE_MOCK_ALLOCATOR=1` for non-k8s environment)")
	}

	dialOpts := append([]grpc.DialOption{grpc.WithInsecure()}, retryDialOptions()...)
	brokerCC, err := grpc.Dial(conf.InternalBrokerAddr, dialOpts...)
	if err != nil {
		log.Fatalf("failed to dial to broker: %+v", err)
	}
	brokerClient := proto.NewBrokerClient(brokerCC)

	gwCC, err := grpc.Dial(conf.GameWrapperAddr, dialOpts...)
	if err != nil {
		log.Fatalf("failed to dial to gamewrapper: %+v", err)
	}
	gwClient := proto.NewGameWrapperClient(gwCC)

	signalingConfig := &gameagent.SignalingConfig{
		AyameURL: conf.AyameAddr,
	}
	streamingConfig := &gameagent.StreamingConfig{
		XDisplay:  conf.XDisplay,
		PulseAddr: conf.PulseAddr,
	}
	agent := gameagent.NewAgent(brokerClient, gwClient, signalingConfig, streamingConfig)
	ctx := context.Background()
	tracks, err := gameagent.NewMediaTracks()
	if err != nil {
		log.Fatalf("failed to get media tracks: %+v", err)
	}
	if err := agent.Run(ctx, gameServerName, tracks); err != nil {
		log.Fatalf("failed to run agent: %+v", err)
	}
}

func retryDialOptions() []grpc.DialOption {
	return []grpc.DialOption{
		grpc.WithUnaryInterceptor(grpc_retry.UnaryClientInterceptor()),
		grpc.WithStreamInterceptor(grpc_retry.StreamClientInterceptor()),
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
