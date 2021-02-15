package gameagent

import (
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
