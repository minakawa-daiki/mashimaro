package main

import (
	sdk "agones.dev/agones/sdks/go"
	"log"
	"os"
	"os/exec"
	"time"
)

func main() {
	agones, err := sdk.NewSDK()
	if err != nil {
		log.Fatalf("failed to new Agones SDK: %+v", err)
	}
	go doHealth(agones)
	cmd := exec.Command(os.Args[0], os.Args[1:]...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		log.Fatalf("failed to start process: %+v", err)
	}
	cmd.Wait()
}

func doHealth(agones *sdk.SDK) {
	ticker := time.NewTicker(2*time.Second)
	defer ticker.Stop()
	for range ticker.C {
		if err := agones.Health(); err != nil {
			log.Printf("failed to health to Agones SDK: %+v", err)
			break
		}
	}
}
