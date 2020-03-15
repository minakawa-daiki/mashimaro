package streamer

import (
	"fmt"
	"github.com/notedit/gstreamer-go"
)

func StartStreamingJPEGRTP(port int) (<-chan []byte, error) {
	caps := "application/x-rtp,encoding-name=JPEG,payload=26,clock-rate=90000"
	x264params := fmt.Sprintf(`speed-preset=ultrafast tune=zerolatency byte-stream=true key-int-max=1 intra-refresh=true`)
	p := fmt.Sprintf(`
udpsrc port=%d caps="%s" ! rtpjitterbuffer ! rtpjpegdepay ! jpegdec ! 
x264enc %s ! appsink name=out`, port, caps, x264params)
	pipeline, err := gstreamer.New(p)
	if err != nil {
		return nil, err
	}

	pipeline.Start()
	out := pipeline.FindElement("out")
	return out.Poll(), nil
}
