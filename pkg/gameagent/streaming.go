package gameagent

import (
	"context"
	"log"

	"github.com/castaneai/mashimaro/pkg/streamer"
	"github.com/castaneai/mashimaro/pkg/transport"
)

func (a *Agent) startStreaming(ctx context.Context, conn transport.StreamerConn, videoConf *streamer.VideoConfig, audioConf *streamer.AudioConfig, captureAreaChanged <-chan *streamer.CaptureArea) error {
	s := streamer.NewStreamer(conn)

	streamErrCh := make(chan error)
	go func() {
		streamErrCh <- s.Start(ctx, videoConf, audioConf)
	}()
	log.Printf("start streaming...")
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case err := <-streamErrCh:
			return err
		case area := <-captureAreaChanged:
			s.RestartVideo(&streamer.VideoConfig{
				CaptureDisplay: videoConf.CaptureDisplay,
				CaptureArea:    streamer.CaptureArea{StartX: area.StartX, StartY: area.StartY, EndX: area.EndX, EndY: area.EndY},
				X264Param:      videoConf.X264Param,
			})
		}
	}
}
