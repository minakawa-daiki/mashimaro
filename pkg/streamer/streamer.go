package streamer

import (
	"context"

	"golang.org/x/sync/errgroup"

	"github.com/castaneai/mashimaro/pkg/transport"
)

type Streamer struct {
	videoCapturer Capturer
	audioCapturer Capturer
	conn          transport.StreamerConn
}

func NewStreamer(conn transport.StreamerConn, videoCapturer, audioCapturer Capturer) *Streamer {
	return &Streamer{
		videoCapturer: videoCapturer,
		audioCapturer: audioCapturer,
		conn:          conn,
	}
}

func (s *Streamer) Start(ctx context.Context) error {
	eg, ctx := errgroup.WithContext(ctx)
	eg.Go(func() error { return s.startVideoStreaming(ctx) })
	eg.Go(func() error { return s.startAudioStreaming(ctx) })
	return eg.Wait()
}

func (s *Streamer) startVideoStreaming(ctx context.Context) error {
	if err := s.videoCapturer.Start(); err != nil {
		return err
	}
	defer func() { _ = s.videoCapturer.Close() }()
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			chunk, err := s.videoCapturer.ReadChunk(ctx)
			if err != nil {
				return err
			}
			if err := s.conn.SendVideoSample(ctx, transport.MediaSample{
				Data:     chunk.Data,
				Duration: chunk.Duration,
			}); err != nil {
				return err
			}
		}
	}
}

func (s *Streamer) startAudioStreaming(ctx context.Context) error {
	if err := s.audioCapturer.Start(); err != nil {
		return err
	}
	defer func() { _ = s.audioCapturer.Close() }()
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			chunk, err := s.audioCapturer.ReadChunk(ctx)
			if err != nil {
				return err
			}
			if err := s.conn.SendAudioSample(ctx, transport.MediaSample{
				Data:     chunk.Data,
				Duration: chunk.Duration,
			}); err != nil {
				return err
			}
		}
	}
}
