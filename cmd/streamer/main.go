package streamer

import (
	"os"

	"github.com/castaneai/mashimaro/pkg/streamer"
)

func main() {
}

func newMediaSources() (videoSrc, audioSrc streamer.MediaStream, err error) {
	if os.Getenv("USE_TEST_MEDIA_SOURCE") != "" {
		videoSrc, err = streamer.NewVideoTestStream()
		if err != nil {
			return
		}
		audioSrc, err = streamer.NewAudioTestStream()
		return
	}

	videoSrc, err = streamer.NewX11VideoStream(":0")
	if err != nil {
		return
	}
	audioSrc, err = streamer.NewPulseAudioStream("localhost:4713")
	return
}
