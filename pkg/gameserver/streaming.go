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

	errCh := make(chan error, 2)
	go func() {
		if err := s.startVideoStreaming(ctx, conn, area); err != nil {
			errCh <- fmt.Errorf("failed to start video streaming: %+v", err)
		}
	}()
	go func() {
		if err := s.startAudioStreaming(ctx, conn); err != nil {
			errCh <- fmt.Errorf("failed to start audio streaming: %+v", err)
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

func (s *GameServer) startVideoStreaming(ctx context.Context, conn transport.StreamerConn, area *streamer.ScreenCaptureArea) error {
	log.Printf("start video streaming")
	videoCapturer := streamer.NewX264Encoder(
		streamer.NewX11ScreenCapturer(os.Getenv("DISPLAY"), area),
		defaultX264Params,
	)
	gstPipeline, err := videoCapturer.CompileGstPipeline()
	if err != nil {
		return err
	}
	resp, err := s.streamer.StartVideoStreaming(ctx, &proto.StartVideoStreamingRequest{GstPipeline: gstPipeline})
	if err != nil {
		return err
	}
	serverAddr := fmt.Sprintf("%s:%d", getStreamerHost(), resp.ListenPort)
	return startStreaming(serverAddr, func(packet *streamerproto.SamplePacket) error {
		return conn.SendVideoSample(ctx, transport.MediaSample{
			Data:     packet.Data,
			Duration: packet.Duration,
		})
	})
}

func (s *GameServer) startAudioStreaming(ctx context.Context, conn transport.StreamerConn) error {
	log.Printf("start audio streaming")
	audioCapturer := streamer.NewOpusEncoder(
		streamer.NewPulseAudioCapturer(os.Getenv("PULSE_ADDR")),
	)
	gstPipeline, err := audioCapturer.CompileGstPipeline()
	if err != nil {
		return err
	}
	resp, err := s.streamer.StartAudioStreaming(ctx, &proto.StartAudioStreamingRequest{GstPipeline: gstPipeline})
	if err != nil {
		return err
	}
	serverAddr := fmt.Sprintf("%s:%d", getStreamerHost(), resp.ListenPort)
	return startStreaming(serverAddr, func(packet *streamerproto.SamplePacket) error {
		return conn.SendAudioSample(ctx, transport.MediaSample{
			Data:     packet.Data,
			Duration: packet.Duration,
		})
	})
}

func startStreaming(serverAddr string, onPacket func(packet *streamerproto.SamplePacket) error) error {
	conn, err := net.Dial("tcp", serverAddr)
	if err != nil {
		return err
	}
	r := bufio.NewReader(conn)
	for {
		var sp streamerproto.SamplePacket
		if err := streamerproto.ReadSamplePacket(r, &sp); err != nil {
			return err
		}
		if err := onPacket(&sp); err != nil {
			return err
		}
	}
}

func getStreamerHost() string {
	if h := os.Getenv("STREAMER_HOST"); h != "" {
		return h
	}
	return "localhost"
}
