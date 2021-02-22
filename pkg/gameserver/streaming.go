package gameserver

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"net"
	"os"

	"github.com/castaneai/mashimaro/pkg/streamer/streamerproto"

	"github.com/castaneai/mashimaro/pkg/proto"

	"github.com/castaneai/mashimaro/pkg/streamer"
	"github.com/castaneai/mashimaro/pkg/transport"
)

const (
	defaultX264Params = "speed-preset=ultrafast tune=zerolatency byte-stream=true intra-refresh=true"
)

func (s *GameServer) startStreaming(ctx context.Context, conn transport.StreamerConn, captureAreaChanged <-chan *streamer.ScreenCaptureArea) error {
	log.Printf("waiting for capture area has changed")
	area := <-captureAreaChanged

	videoCapturer := streamer.NewX264Encoder(
		streamer.NewX11ScreenCapturer(os.Getenv("DISPLAY"), area),
		defaultX264Params,
	)
	gstPipeline, err := videoCapturer.CompileGstPipeline()
	if err != nil {
		return err
	}
	// TODO: audio
	log.Printf("start streaming")
	resp, err := s.streamer.StartVideoStreaming(ctx, &proto.StartVideoStreamingRequest{GstPipeline: gstPipeline})
	if err != nil {
		return err
	}
	errCh := make(chan error)
	go func() {
		streamConn, err := net.Dial("tcp", fmt.Sprintf("localhost:%d", resp.ListenPort))
		if err != nil {
			errCh <- err
			return
		}
		r := bufio.NewReader(streamConn)
		for {
			var sp streamerproto.SamplePacket
			if err := streamerproto.ReadSamplePacket(r, &sp); err != nil {
				errCh <- err
				return
			}
			if err := conn.SendVideoSample(ctx, transport.MediaSample{
				Data:     sp.Data,
				Duration: sp.Duration,
			}); err != nil {
				errCh <- err
				return
			}
		}
	}()
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case err := <-errCh:
			// TODO: retry streaming
			return err
		case <-captureAreaChanged:
			// TODO: change capture area
		}
	}
}
