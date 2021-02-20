package streamer

import (
	"context"
	"fmt"
	"io"
	"log"
	"math"
	"time"

	"github.com/pkg/errors"

	"github.com/notedit/gst"
)

type MediaChunk struct {
	Data     []byte
	Duration time.Duration
}

type Capturer interface {
	io.Closer
	Start() error
	ReadChunk(ctx context.Context) (*MediaChunk, error)
}

type GstCapturer struct {
	name        string
	pipelineStr string
	gstPipeline *gst.Pipeline
	sinkElement *gst.Element
}

func NewGstCapturer(name, pipelineStr, sinkName string) (*GstCapturer, error) {
	pipeline, err := gst.ParseLaunch(pipelineStr)
	if err != nil {
		return nil, err
	}
	element := pipeline.GetByName(sinkName)
	return &GstCapturer{
		name:        name,
		pipelineStr: pipelineStr,
		gstPipeline: pipeline,
		sinkElement: element,
	}, nil
}

func (c *GstCapturer) Start() error {
	log.Printf("Starting GStreamer pipeline %s: %s", c.name, c.pipelineStr)
	go func() {
		bus := c.gstPipeline.GetBus()
		for {
			msg := bus.Pull(gst.MessageAny)
			st := msg.GetStructure()
			if st.C != nil {
				log.Printf("[gst %s] %s", c.name, st.ToString())
			} else {
				log.Printf("[gst %s] %s", c.name, msg.GetName())
			}
		}
	}()
	c.gstPipeline.SetState(gst.StatePlaying)
	return nil
}

func (c *GstCapturer) Close() error {
	log.Printf("Stopping GStreamer pipeline: %s", c.name)
	c.gstPipeline.SetState(gst.StateNull)
	return nil
}

func (c *GstCapturer) ReadChunk(ctx context.Context) (*MediaChunk, error) {
	sample, err := c.sinkElement.PullSample()
	if err != nil {
		if c.sinkElement.IsEOS() {
			return nil, io.EOF
		}
		return nil, errors.Wrap(err, "failed to pull sample from sinkElement")
	}
	return &MediaChunk{
		Data:     sample.Data,
		Duration: time.Duration(sample.Duration),
	}, nil
}

type ScreenCaptureArea struct {
	StartX int
	StartY int
	EndX   int
	EndY   int
}

func (a *ScreenCaptureArea) Width() int {
	return a.EndX - a.StartX
}

func (a *ScreenCaptureArea) Height() int {
	return a.EndY - a.StartY
}

func (a *ScreenCaptureArea) IsValid() bool {
	return (a.StartX > 0 && a.StartY > 0 && a.EndX > 0 && a.EndY > 0) &&
		a.Width() > 0 && a.Height() > 0
}

func (a *ScreenCaptureArea) FixForH264() {
	// H264 requirement is that video dimensions are divisible by 2.
	// ref: https://github.com/hzbd/kazam/blob/491869ac29860a19254fa8c226f75314a7eee83d/kazam/backend/gstreamer.py#L128
	if int(math.Abs(float64(a.StartX-a.EndX)))%2 != 0 {
		a.EndX -= 1
		if a.EndX < 0 {
			a.EndX = 0
		}
	}
	if int(math.Abs(float64(a.StartY-a.EndY)))%2 != 0 {
		a.EndY -= 1
		if a.EndY < 0 {
			a.EndY = 0
		}
	}
}

type X11CaptureConfig struct {
	Display string
	Area    *ScreenCaptureArea
}

type X264EncodeConfig struct {
	X264Params string
}

func NewX11Capturer(x11conf *X11CaptureConfig, x264conf *X264EncodeConfig) (Capturer, error) {
	if err := gst.CheckPlugins([]string{"ximagesrc"}); err != nil {
		return nil, err
	}
	x11conf.Area.FixForH264()
	startX := x11conf.Area.StartX
	if startX < 0 {
		startX = 0
	}
	startY := x11conf.Area.StartY
	if startY < 0 {
		startY = 0
	}
	endX := x11conf.Area.EndX - 1
	if endX < 0 {
		endX = 0
	}
	endY := x11conf.Area.EndY - 1
	if endY < 0 {
		endY = 0
	}
	// why use-damage=0?: https://github.com/GoogleCloudPlatform/selkies-vdi/blob/0da21b7c9432bd5c99f1f9f7c541ac9c583f9ef4/images/gst-webrtc-app/gstwebrtc_app.py#L148
	src := fmt.Sprintf("ximagesrc display-name=%s remote=1 use-damage=0 startx=%d starty=%d endx=%d endy=%d ! queue ",
		x11conf.Display, startX, startY, endX, endY)
	return newX264EncodeCapturer(src, x264conf)
}

func NewVideoTestCapturer() (Capturer, error) {
	return newX264EncodeCapturer("videotestsrc", &X264EncodeConfig{})
}

func NewAudioTestCapturer() (Capturer, error) {
	return newOpusEncodeCapturer("audiotestsrc", &OpusEncodeConfig{})
}

func newX264EncodeCapturer(gstSrcPipelineStr string, conf *X264EncodeConfig) (Capturer, error) {
	if err := gst.CheckPlugins([]string{"x264"}); err != nil {
		return nil, err
	}
	pipelineStr := fmt.Sprintf("%s ! videoconvert ! video/x-raw,format=I420 ! x264enc %s ! appsink name=video", gstSrcPipelineStr, conf.X264Params)
	return NewGstCapturer("video", pipelineStr, "video")
}

type PulseAudioCaptureConfig struct {
	PulseServer string
}

type OpusEncodeConfig struct {
	// TODO
}

func NewPulseAudioCapturer(pulseConf *PulseAudioCaptureConfig, opusConf *OpusEncodeConfig) (Capturer, error) {
	if err := gst.CheckPlugins([]string{"pulseaudio"}); err != nil {
		return nil, err
	}
	return newOpusEncodeCapturer(fmt.Sprintf("pulsesrc server=%s ! queue ", pulseConf.PulseServer), opusConf)
}

func newOpusEncodeCapturer(gstSrcPipelineStr string, conf *OpusEncodeConfig) (Capturer, error) {
	if err := gst.CheckPlugins([]string{"opus"}); err != nil {
		return nil, err
	}
	pipelineStr := fmt.Sprintf("%s ! opusenc ! appsink name=audio", gstSrcPipelineStr)
	return NewGstCapturer("audio", pipelineStr, "audio")
}
