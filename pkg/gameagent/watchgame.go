package gameagent

import (
	"context"
	"fmt"
	"log"
	"time"

	"golang.org/x/sync/errgroup"

	"github.com/castaneai/mashimaro/pkg/streamer"
	"github.com/pkg/errors"

	"github.com/castaneai/mashimaro/pkg/proto"
)

func (a *Agent) startWatchGame(ctx context.Context, pub *captureAreaPubSub) error {
	var eg errgroup.Group
	eg.Go(func() error {
		return a.startListenCaptureArea(ctx, pub)
	})
	eg.Go(func() error {
		return a.startHealthCheck(ctx, 5*time.Second)
	})
	return eg.Wait()
}

func (a *Agent) startListenCaptureArea(ctx context.Context, pub *captureAreaPubSub) error {
	st, err := a.gameWrapperClient.ListenCaptureArea(ctx, &proto.ListenCaptureAreaRequest{})
	if err != nil {
		return errors.Wrap(err, "failed to listen capture area")
	}
	for {
		resp, err := st.Recv()
		if err != nil {
			return errors.Wrap(err, "failed to recv listen capture area")
		}
		pub.Publish(&streamer.CaptureArea{
			StartX: int(resp.StartX),
			StartY: int(resp.StartY),
			EndX:   int(resp.EndX),
			EndY:   int(resp.EndY),
		})
	}
}

func (a *Agent) startHealthCheck(ctx context.Context, interval time.Duration) error {
	log.Printf("start healthcheck...")
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			resp, err := a.gameWrapperClient.HealthCheck(ctx, &proto.HealthCheckRequest{})
			if err != nil {
				return errors.Wrap(err, "failed to health check")
			}
			if !resp.Healthy {
				return fmt.Errorf("game process is unhealthy(exited)")
			}
		}
	}
}
