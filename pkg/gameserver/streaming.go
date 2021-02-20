package gameserver

import (
	"context"
	"log"
	"os"

	"github.com/castaneai/mashimaro/pkg/streamer"
	"github.com/castaneai/mashimaro/pkg/transport"
)

const (
	defaultX264Params = "speed-preset=ultrafast tune=zerolatency byte-stream=true intra-refresh=true"
)

func (s *GameServer) startStreaming(ctx context.Context, conn transport.StreamerConn, captureAreaChanged <-chan *streamer.ScreenCaptureArea) error {
	log.Printf("waiting for capture area has changed")
	area := <-captureAreaChanged
	videoCapturer, err := streamer.NewX11Capturer(&streamer.X11CaptureConfig{
		Display: os.Getenv("DISPLAY"),
		Area:    area,
	}, &streamer.X264EncodeConfig{X264Params: defaultX264Params})
	if err != nil {
		return err
	}
	// TODO: start audio capture
	audioCapturer, err := streamer.NewPulseAudioCapturer(&streamer.PulseAudioCaptureConfig{PulseServer: "localhost:4713"}, &streamer.OpusEncodeConfig{})
	if err != nil {
		return err
	}
	st := streamer.NewStreamer(conn, videoCapturer, audioCapturer)
	log.Printf("start streaming")

	streamErrCh := make(chan error)
	go func() {
		streamErrCh <- st.Start(ctx)
	}()
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case err := <-streamErrCh:
			return err
		case <-captureAreaChanged:
			// TODO: change capture area
		}
	}
}
