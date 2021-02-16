package gameagent

import (
	"context"
	"fmt"

	"github.com/castaneai/mashimaro/pkg/transport"

	"github.com/pkg/errors"

	"github.com/castaneai/mashimaro/pkg/streamer"
	"golang.org/x/sync/errgroup"
)

func startStreaming(ctx context.Context, conn *transport.WebRTCStreamerConn, videoStream, audioStream streamer.MediaStream) error {
	defer videoStream.Close()
	defer audioStream.Close()
	videoStream.Start()
	audioStream.Start()
	eg := &errgroup.Group{}
	eg.Go(func() error {
		if err := startSendingVideoToConn(ctx, conn, videoStream); err != nil {
			return errors.Wrap(err, "failed to send video to track")
		}
		return nil
	})
	eg.Go(func() error {
		if err := startSendingAudioToConn(ctx, conn, audioStream); err != nil {
			return errors.Wrap(err, "failed to send audio to track")
		}
		return nil
	})
	return eg.Wait()
}

func startSendingVideoToConn(ctx context.Context, conn *transport.WebRTCStreamerConn, stream streamer.MediaStream) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			chunk, err := stream.ReadChunk()
			if err != nil {
				return fmt.Errorf("failed to read chunk from stream: %+v", err)
			}
			if err := conn.SendVideoSample(ctx, transport.MediaSample{
				Data:     chunk.Data,
				Duration: chunk.Duration,
			}); err != nil {
				return fmt.Errorf("failed to write sample to track: %+v", err)
			}
		}
	}
}

func startSendingAudioToConn(ctx context.Context, conn *transport.WebRTCStreamerConn, stream streamer.MediaStream) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			chunk, err := stream.ReadChunk()
			if err != nil {
				return fmt.Errorf("failed to read chunk from stream: %+v", err)
			}
			if err := conn.SendAudioSample(ctx, transport.MediaSample{
				Data:     chunk.Data,
				Duration: chunk.Duration,
			}); err != nil {
				return fmt.Errorf("failed to write sample to track: %+v", err)
			}
		}
	}
}
