package streamer

import (
	"log"
	"testing"
	"time"

	"github.com/notedit/gstreamer-go"
)

func TestGst(t *testing.T) {
	// pipeline, err := gstreamer.New("appsrc name=mysource format=time is-live=true do-timestamp=true ! rawvideoparse width=320 height=240 format=rgb ! videoconvert ! appsink name=out")
	pipeline, err := gstreamer.New("appsrc name=mysource format=time is-live=true do-timestamp=true ! videoconvert ! vp8enc ! avmux_ivf ! appsink name=out")
	if err != nil {
		t.Error("pipeline create error", err)
		t.FailNow()
	}

	appsrc := pipeline.FindElement("mysource")
	appsrc.SetCap("video/x-raw,format=RGB,width=320,height=240,bpp=24,depth=24")

	pipeline.Start()
	oute := pipeline.FindElement("out")
	out := oute.Poll()

	go func() {
		for {
			select {
			case b := <-out:
				log.Printf("bytes out! %d", len(b))
			}
		}
	}()

	for {
		time.Sleep(100 * time.Millisecond)
		appsrc.Push(make([]byte, 320*240*3))
	}
}

func TestJPEGOnRTP(t *testing.T) {
	out, err := StartStreamingJPEGRTP(9999)
	if err != nil {
		t.Fatal(err)
	}

	for b := range out {
		log.Printf("bytes! (len: %d)", len(b))
	}
}
