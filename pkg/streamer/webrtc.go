package streamer

import (
	"context"
	"fmt"
	"log"

	"github.com/pion/webrtc/v3"
	"github.com/pion/webrtc/v3/pkg/media"
)

type MediaStreamer struct {
	videoSource MediaStream
	videoSink   *webrtc.TrackLocalStaticSample
	audioSource MediaStream
	audioSink   *webrtc.TrackLocalStaticSample
}

func NewMediaStreamer(videoSource MediaStream, videoSink *webrtc.TrackLocalStaticSample, audioSource MediaStream, audioSink *webrtc.TrackLocalStaticSample) *MediaStreamer {
	return &MediaStreamer{videoSource: videoSource, videoSink: videoSink, audioSource: audioSource, audioSink: audioSink}
}

func (s *MediaStreamer) Start(ctx context.Context) {
	go func() {
		if err := startStreamingMedia(ctx, s.videoSink, s.videoSource); err != nil {
			log.Printf("failed to streaming video: %+v", err)
		}
	}()
	if err := startStreamingMedia(ctx, s.audioSink, s.audioSource); err != nil {
		log.Printf("failed to streaming audio: %+v", err)
	}
}

func startStreamingMedia(ctx context.Context, track *webrtc.TrackLocalStaticSample, stream MediaStream) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			chunk, err := stream.ReadChunk()
			if err != nil {
				return fmt.Errorf("failed to read chunk from stream: %+v", err)
			}
			if err := track.WriteSample(media.Sample{
				Data:     chunk.Data,
				Duration: chunk.Duration,
			}); err != nil {
				return fmt.Errorf("failed to write sample to track: %+v", err)
			}
		}
	}
}
