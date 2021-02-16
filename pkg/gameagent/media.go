package gameagent

import (
	"context"
	"fmt"

	"github.com/castaneai/mashimaro/pkg/transport"

	"github.com/castaneai/mashimaro/pkg/streamer"
)

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
