package gameagent

import (
	"errors"
	"os"

	"github.com/castaneai/mashimaro/pkg/streamer"
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

func getMediaStreams() (videoSrc, audioSrc streamer.MediaStream, err error) {
	if os.Getenv("USE_TEST_MEDIA_SOURCE") != "" {
		videoSrc, err = streamer.NewVideoTestStream()
		if err != nil {
			return
		}
		audioSrc, err = streamer.NewAudioTestStream()
		return
	}

	// TODO: :0 -> ${DISPLAY}
	videoSrc, err = streamer.NewX11VideoStream(":0")
	if err != nil {
		return
	}
	paddr := os.Getenv("PULSE_ADDR")
	if paddr == "" {
		err = errors.New("env: PULSE_ADDR not set")
		return
	}
	audioSrc, err = streamer.NewPulseAudioStream(paddr)
	return
}
