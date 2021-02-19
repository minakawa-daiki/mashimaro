package streamer

import (
	"context"
	"fmt"
	"log"

	"github.com/castaneai/mashimaro/pkg/transport"
	"github.com/pkg/errors"
)

type VideoConfig struct {
	CaptureDisplay string
	CaptureArea    CaptureArea
	X264Param      string
}

type AudioConfig struct {
	PulseServer string
}

type Streamer struct {
	videoCancel    context.CancelFunc
	audioCancel    context.CancelFunc
	restartVideoCh chan *VideoConfig
	restartAudioCh chan *AudioConfig
	conn           transport.StreamerConn
}

func NewStreamer(conn transport.StreamerConn) *Streamer {
	return &Streamer{
		restartVideoCh: make(chan *VideoConfig),
		restartAudioCh: make(chan *AudioConfig),
		conn:           conn,
	}
}

func (s *Streamer) Start(ctx context.Context, videoConf *VideoConfig, audioConf *AudioConfig) error {
	if err := s.startVideoStreaming(ctx, videoConf); err != nil {
		return err
	}
	if err := s.startAudioStreaming(ctx, audioConf); err != nil {
		return err
	}
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case vc := <-s.restartVideoCh:
			log.Printf("restrating video streaming...")
			s.videoCancel()
			if err := s.startVideoStreaming(ctx, vc); err != nil {
				return err
			}
		case ac := <-s.restartAudioCh:
			log.Printf("restrating audio streaming...")
			s.audioCancel()
			if err := s.startAudioStreaming(ctx, ac); err != nil {
				return err
			}
		}
	}
}

func (s *Streamer) RestartVideo(conf *VideoConfig) {
	s.restartVideoCh <- conf
}

func (s *Streamer) RestartAudio(conf *AudioConfig) {
	s.restartAudioCh <- conf
}

func (s *Streamer) startVideoStreaming(ctx context.Context, conf *VideoConfig) error {
	stream, err := NewX11VideoStream(conf)
	if err != nil {
		return err
	}
	sctx, cancel := context.WithCancel(ctx)
	s.videoCancel = cancel
	go func() {
		stream.Start()
		defer func() {
			_ = stream.Close()
		}()
		if err := startStreaming(sctx, func(ctx context.Context, chunk *MediaChunk) error {
			if err := s.conn.SendVideoSample(ctx, transport.MediaSample{
				Data:     chunk.Data,
				Duration: chunk.Duration,
			}); err != nil {
				return errors.Wrap(err, "failed to write sample to track")
			}
			return nil
		}, stream); err != nil {
			if !errors.Is(err, context.Canceled) {
				log.Printf("failed to stream video: %+v", err)
			}
		}
	}()
	return nil
}

func (s *Streamer) startAudioStreaming(ctx context.Context, conf *AudioConfig) error {
	stream, err := NewPulseAudioStream(conf)
	if err != nil {
		return err
	}
	sctx, cancel := context.WithCancel(ctx)
	s.audioCancel = cancel
	go func() {
		stream.Start()
		defer func() {
			_ = stream.Close()
		}()
		if err := startStreaming(sctx, func(ctx context.Context, chunk *MediaChunk) error {
			if err := s.conn.SendAudioSample(ctx, transport.MediaSample{
				Data:     chunk.Data,
				Duration: chunk.Duration,
			}); err != nil {
				return errors.Wrap(err, "failed to write sample to track")
			}
			return nil
		}, stream); err != nil {
			if !errors.Is(err, context.Canceled) {
				log.Printf("failed to stream audio: %+v", err)
			}
		}
	}()
	return nil
}

func startStreaming(ctx context.Context, onChunk func(ctx context.Context, chunk *MediaChunk) error, stream MediaStream) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			chunk, err := stream.ReadChunk()
			if err != nil {
				return fmt.Errorf("failed to read chunk from stream: %+v", err)
			}
			if err := onChunk(ctx, chunk); err != nil {
				return err
			}
		}
	}
}
