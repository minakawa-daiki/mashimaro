package main

import (
	"context"
	"log"
	"os"
	"time"

	"github.com/castaneai/mashimaro/pkg/allocator"

	"github.com/castaneai/mashimaro/pkg/transport"

	"github.com/castaneai/mashimaro/pkg/gameserver"

	grpc_retry "github.com/grpc-ecosystem/go-grpc-middleware/retry"

	"github.com/kelseyhightower/envconfig"

	"github.com/castaneai/mashimaro/pkg/proto"
	"google.golang.org/grpc"

	sdk "agones.dev/agones/sdks/go"
)

type config struct {
	AyameLaboURL           string `envconfig:"AYAME_LABO_URL" required:"true"`
	AyameLaboSignalingKey  string `envconfig:"AYAME_LABO_SIGNALING_KEY" required:"true"`
	AyameLaboGitHubAccount string `envconfig:"AYAME_LABO_GITHUB_ACCOUNT" required:"true"`
	InternalBrokerAddr     string `envconfig:"INTERNAL_BROKER_ADDR" required:"true"`
	GameProcessAddr        string `envconfig:"GAME_PROCESS_ADDR" required:"true"`
	UseMockAllocator       bool   `envconfig:"USE_MOCK_ALLOCATOR" default:"false"`
}

func main() {
	var conf config
	if err := envconfig.Process("", &conf); err != nil {
		log.Fatalf("failed to process config: %+v", err)
	}
	log.Printf("load config: %+v", conf)

	allocatedServerID := ""
	if conf.UseMockAllocator {
		allocatedServerID = "dummy"
	}
	var agones *sdk.SDK
	if isRunningOnKubernetes() {
		agones = setupAgones()
		gs, err := agones.GameServer()
		if err != nil {
			log.Fatalf("failed to get agones AllocatedServer: %+v", err)
		}
		allocatedServerID = gs.ObjectMeta.Name
	}
	if allocatedServerID == "" {
		log.Fatalf("allocatedServerID not set (Set `USE_MOCK_ALLOCATOR=1` for non-k8s environment)")
	}
	allocatedServer := &allocator.AllocatedServer{ID: allocatedServerID}

	dialOpts := append([]grpc.DialOption{grpc.WithInsecure()}, retryDialOptions()...)
	brokerCC, err := grpc.Dial(conf.InternalBrokerAddr, dialOpts...)
	if err != nil {
		log.Fatalf("failed to dial to broker: %+v", err)
	}
	brokerClient := proto.NewBrokerClient(brokerCC)
	gameProcessCC, err := grpc.Dial(conf.GameProcessAddr, dialOpts...)
	if err != nil {
		log.Fatalf("failed to dial to game process: %+v", err)
	}
	gameProcessClient := proto.NewGameProcessClient(gameProcessCC)
	signaler := transport.NewAyameLaboSignaler(conf.AyameLaboURL, conf.AyameLaboSignalingKey, conf.AyameLaboGitHubAccount)
	gameServer := gameserver.NewGameServer(allocatedServer, brokerClient, gameProcessClient, signaler)
	if agones != nil {
		gameServer.OnShutdown(func() {
			if err := agones.Shutdown(); err != nil {
				log.Printf("failed to send shutdown to Agones SDK: %+v", err)
			}
		})
	}
	ctx := context.Background()
	if err := gameServer.Serve(ctx); err != nil {
		log.Fatalf("failed to serve game server: %+v", err)
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
	go agonesHealth(agones)
	return agones
}

func agonesHealth(agones *sdk.SDK) {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()
	for range ticker.C {
		if err := agones.Health(); err != nil {
			log.Printf("failed to health to Agones SDK: %+v", err)
			break
		}
	}
}
