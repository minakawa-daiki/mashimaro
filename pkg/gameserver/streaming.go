package gameserver

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"sync"

	"github.com/tevino/abool"

	"github.com/castaneai/mashimaro/pkg/encoder/encoderproto"

	"github.com/castaneai/mashimaro/pkg/proto"

	"github.com/castaneai/mashimaro/pkg/transport"
)

const (
	defaultX264Params = "speed-preset=ultrafast tune=zerolatency byte-stream=true intra-refresh=true"
)

func (s *GameServer) startStreaming(ctx context.Context, conn transport.StreamerConn, captureRectChanged <-chan ScreenRect) error {
	errCh := make(chan error, 2)
	go func() {
		if err := s.startAudioStreaming(ctx, conn); err != nil {
			errCh <- fmt.Errorf("failed to start audio streaming: %+v", err)
		}
	}()

	var videoConn *encoderConn
	log.Printf("waiting for capture rect")
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case err := <-errCh:
			// TODO: retry streaming
			return err
		case rect := <-captureRectChanged:
			log.Printf("capture rect detected(%s)", &rect)
			if videoConn != nil {
				videoConn.Stop()
			}
			vc, err := s.newVideoEncoderConn(ctx, conn, &rect)
			if err != nil {
				return err
			}
			videoConn = vc
			go func() {
				if err := vc.start(ctx, func(ctx context.Context, packet *encoderproto.SamplePacket) error {
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

func (s *GameServer) newVideoEncoderConn(ctx context.Context, conn transport.StreamerConn, rect *ScreenRect) (*encoderConn, error) {
	log.Printf("start video streaming")
	video := NewX264Encoder(
		NewX11ScreenCapturer(os.Getenv("DISPLAY"), rect),
		defaultX264Params,
	)
	gstPipeline, err := video.CompileGstPipeline()
	if err != nil {
		return nil, err
	}
	st := newEncoderConn(s.encoder, "video", gstPipeline)
	return st, nil
}

func (s *GameServer) startAudioStreaming(ctx context.Context, conn transport.StreamerConn) error {
	log.Printf("start audio streaming")
	audio := NewOpusEncoder(
		NewPulseAudioCapturer(os.Getenv("PULSE_ADDR")),
	)
	gstPipeline, err := audio.CompileGstPipeline()
	if err != nil {
		return err
	}
	st := newEncoderConn(s.encoder, "audio", gstPipeline)
	return st.start(ctx, func(ctx context.Context, packet *encoderproto.SamplePacket) error {
		return conn.SendAudioSample(ctx, transport.MediaSample{
			Data:     packet.Data,
			Duration: packet.Duration,
		})
	})
}

func getEncoderHost() string {
	if h := os.Getenv("ENCODER_HOST"); h != "" {
		return h
	}
	return "localhost"
}

type encoderConn struct {
	client      proto.EncoderClient
	pipelineID  string
	gstPipeline string
	conn        net.Conn
	connMu      sync.Mutex
	stopped     *abool.AtomicBool
}

func newEncoderConn(client proto.EncoderClient, pipelineID, gstPipeline string) *encoderConn {
	return &encoderConn{
		client:      client,
		pipelineID:  pipelineID,
		gstPipeline: gstPipeline,
		stopped:     abool.New(),
	}
}

func (s *encoderConn) start(ctx context.Context, onPacket func(ctx context.Context, packet *encoderproto.SamplePacket) error) error {
	resp, err := s.client.StartEncoding(ctx, &proto.StartEncodingRequest{
		PipelineId:  s.pipelineID,
		GstPipeline: s.gstPipeline,
		Port:        0, // random port allocation
	})
	if err != nil {
		return err
	}
	serverAddr := fmt.Sprintf("%s:%d", getEncoderHost(), resp.ListenPort)
	if err := s.startReceivingMedia(serverAddr, func(packet *encoderproto.SamplePacket) error {
		return onPacket(ctx, packet)
	}); err != nil {
		if s.stopped.IsSet() {
			return nil
		}
		return err
	}
	return nil
}

func (s *encoderConn) startReceivingMedia(serverAddr string, onPacket func(packet *encoderproto.SamplePacket) error) error {
	conn, err := net.Dial("tcp", serverAddr)
	if err != nil {
		return err
	}
	s.connMu.Lock()
	s.conn = conn
	s.connMu.Unlock()
	r := bufio.NewReader(conn)
	for {
		var sp encoderproto.SamplePacket
		if err := encoderproto.ReadSamplePacket(r, &sp); err != nil {
			return err
		}
		if err := onPacket(&sp); err != nil {
			return err
		}
	}
}

func (s *encoderConn) Stop() {
	s.stopped.Set()
	s.connMu.Lock()
	defer s.connMu.Unlock()
	if s.conn != nil {
		_ = s.conn.Close()
	}
}
