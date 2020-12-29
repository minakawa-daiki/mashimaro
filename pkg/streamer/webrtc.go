package streamer

import (
	"context"
	"fmt"
	"log"

	"github.com/pion/webrtc/v3/pkg/media"

	"github.com/pion/webrtc/v3"
)

type MediaTracks struct {
	VideoTrack *webrtc.TrackLocalStaticSample
	AudioTrack *webrtc.TrackLocalStaticSample
}

func NewMediaTracks() (*MediaTracks, error) {
	videoTrack, err := webrtc.NewTrackLocalStaticSample(webrtc.RTPCodecCapability{
		MimeType:  webrtc.MimeTypeH264,
		ClockRate: 90000, // why?
	}, "video", "mashimaro")
	if err != nil {
		return nil, err
	}

	audioTrack, err := webrtc.NewTrackLocalStaticSample(webrtc.RTPCodecCapability{
		MimeType:  webrtc.MimeTypeOpus,
		ClockRate: 48000, // why?
	}, "audio", "mashimaro")
	if err != nil {
		return nil, err
	}
	return &MediaTracks{
		VideoTrack: videoTrack,
		AudioTrack: audioTrack,
	}, nil

}

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
