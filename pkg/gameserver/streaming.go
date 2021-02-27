package gameserver

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"strings"
	"sync"
	"syscall"

	"github.com/tevino/abool"

	"github.com/pkg/errors"

	"github.com/castaneai/mashimaro/pkg/streamer/streamerproto"

	"github.com/castaneai/mashimaro/pkg/proto"

	"github.com/castaneai/mashimaro/pkg/streamer"
	"github.com/castaneai/mashimaro/pkg/transport"
)

const (
	defaultX264Params = "speed-preset=ultrafast tune=zerolatency byte-stream=true intra-refresh=true"
)

func (s *GameServer) startStreaming(ctx context.Context, conn transport.StreamerConn, captureAreaChanged <-chan streamer.ScreenCaptureArea) error {
	errCh := make(chan error, 2)
	go func() {
		if err := s.startAudioStreaming(ctx, conn); err != nil {
			errCh <- fmt.Errorf("failed to start audio streaming: %+v", err)
		}
	}()

	var videoStreamer *Streamer
	log.Printf("waiting for capture area has changed")
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case err := <-errCh:
			// TODO: retry streaming
			return err
		case area := <-captureAreaChanged:
			log.Printf("capture area detected(%s)", &area)
			if videoStreamer != nil {
				videoStreamer.Stop()
			}
			st, err := s.newVideoStreamer(ctx, conn, &area)
			if err != nil {
				return err
			}
			videoStreamer = st
			go func() {
				if err := st.start(ctx, func(ctx context.Context, packet *streamerproto.SamplePacket) error {
					return conn.SendVideoSample(ctx, transport.MediaSample{
						Data:     packet.Data,
						Duration: packet.Duration,
					})
				}); err != nil {
					errCh <- fmt.Errorf("failed to start video streaming: %+v", err)
				}
			}()
		}
	}
}

func (s *GameServer) newVideoStreamer(ctx context.Context, conn transport.StreamerConn, area *streamer.ScreenCaptureArea) (*Streamer, error) {
	log.Printf("start video streaming")
	videoCapturer := streamer.NewX264Encoder(
		streamer.NewX11ScreenCapturer(os.Getenv("DISPLAY"), area),
		defaultX264Params,
	)
	gstPipeline, err := videoCapturer.CompileGstPipeline()
	if err != nil {
		return nil, err
	}
	st := newStreamer(s.streamer, "video", gstPipeline, conn)
	return st, nil
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
	st := newStreamer(s.streamer, "audio", gstPipeline, conn)
	return st.start(ctx, func(ctx context.Context, packet *streamerproto.SamplePacket) error {
		return conn.SendAudioSample(ctx, transport.MediaSample{
			Data:     packet.Data,
			Duration: packet.Duration,
		})
	})
}

func getStreamerHost() string {
	if h := os.Getenv("STREAMER_HOST"); h != "" {
		return h
	}
	return "localhost"
}

type Streamer struct {
	client        proto.StreamerClient
	mediaID       string
	gstPipeline   string
	streamerConn  transport.StreamerConn
	mediaDataConn net.Conn
	mediaDataMu   sync.Mutex
	stopped       *abool.AtomicBool
}

func newStreamer(client proto.StreamerClient, mediaID, gstPipeline string, streamerConn transport.StreamerConn) *Streamer {
	return &Streamer{
		client:       client,
		mediaID:      mediaID,
		gstPipeline:  gstPipeline,
		streamerConn: streamerConn,
		stopped:      abool.New(),
	}
}

func (s *Streamer) start(ctx context.Context, onPacket func(ctx context.Context, packet *streamerproto.SamplePacket) error) error {
	resp, err := s.client.StartStreaming(ctx, &proto.StartStreamingRequest{
		MediaId:     s.mediaID,
		GstPipeline: s.gstPipeline,
		Port:        0, // random port allocation
	})
	if err != nil {
		return err
	}
	serverAddr := fmt.Sprintf("%s:%d", getStreamerHost(), resp.ListenPort)
	if err := s.startStreaming(serverAddr, func(packet *streamerproto.SamplePacket) error {
		return onPacket(ctx, packet)
	}); err != nil {
		if s.stopped.IsSet() {
			return nil
		}
		return err
	}
	return nil
}

func (s *Streamer) startStreaming(serverAddr string, onPacket func(packet *streamerproto.SamplePacket) error) error {
	conn, err := net.Dial("tcp", serverAddr)
	if err != nil {
		return err
	}
	s.mediaDataMu.Lock()
	s.mediaDataConn = conn
	s.mediaDataMu.Unlock()
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

func isClosedByPeer(err error) bool {
	return errors.Is(err, io.EOF) ||
		errors.Is(err, io.ErrUnexpectedEOF) ||
		errors.Is(err, syscall.ECONNRESET) ||
		(err != nil && strings.Contains(err.Error(), "use of closed network connection"))
}

func (s *Streamer) Stop() {
	s.stopped.Set()
	s.mediaDataMu.Lock()
	defer s.mediaDataMu.Unlock()
	if s.mediaDataConn != nil {
		_ = s.mediaDataConn.Close()
	}
}
