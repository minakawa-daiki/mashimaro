package gameagent

import (
	"context"
	"fmt"

	"github.com/pion/webrtc/v3/pkg/media"
	"github.com/pkg/errors"

	"github.com/castaneai/mashimaro/pkg/streamer"
	"github.com/pion/webrtc/v3"
	"golang.org/x/sync/errgroup"
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

func startStreaming(ctx context.Context, videoTrack, audioTrack *webrtc.TrackLocalStaticSample, videoStream, audioStream streamer.MediaStream) error {
	defer videoStream.Close()
	defer audioStream.Close()
	videoStream.Start()
	audioStream.Start()
	eg := &errgroup.Group{}
	eg.Go(func() error {
		if err := startSendingMediaToTrack(ctx, videoTrack, videoStream); err != nil {
			return errors.Wrap(err, "failed to send video to track")
		}
		return nil
	})
	eg.Go(func() error {
		if err := startSendingMediaToTrack(ctx, audioTrack, audioStream); err != nil {
			return errors.Wrap(err, "failed to send audio to track")
		}
		return nil
	})
	return eg.Wait()
}

func startSendingMediaToTrack(ctx context.Context, track *webrtc.TrackLocalStaticSample, stream streamer.MediaStream) error {
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
