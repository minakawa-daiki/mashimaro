package streamer

import (
	"fmt"
	"io"
	"log"
	"math"
	"sync"
	"time"

	"github.com/notedit/gst"
)

type MediaChunk struct {
	Data     []byte
	Duration time.Duration
}

type MediaStream interface {
	io.Closer
	Start()
	ReadChunk() (*MediaChunk, error)
}

type gstStream struct {
	pipelineStr string
	gstPipeline *gst.Pipeline
	gstElement  *gst.Element
	mu          sync.Mutex
}

func newGstStream(pipelineStr, sinkName string) (*gstStream, error) {
	pipeline, err := gst.ParseLaunch(pipelineStr)
	if err != nil {
		return nil, err
	}
	element := pipeline.GetByName(sinkName)
	return &gstStream{
		pipelineStr: pipelineStr,
		gstPipeline: pipeline,
		gstElement:  element,
	}, nil
}

func (s *gstStream) Start() {
	log.Printf("Starting GStreamer pipeline: %s", s.pipelineStr)
	s.mu.Lock()
	defer s.mu.Unlock()
	s.gstPipeline.SetState(gst.StatePlaying)
}

func (s *gstStream) Close() error {
	log.Printf("Stopping GStreamer pipeline: %s", s.pipelineStr)
	s.mu.Lock()
	defer s.mu.Unlock()
	s.gstPipeline.SetState(gst.StateNull)
	return nil
}

func (s *gstStream) ReadChunk() (*MediaChunk, error) {
	sample, err := s.gstElement.PullSample()
	if err != nil {
		if s.gstElement.IsEOS() {
			return nil, io.EOF
		}
		return nil, err
	}
	return &MediaChunk{
		Data:     sample.Data,
		Duration: time.Duration(sample.Duration),
	}, nil
}

type CaptureArea struct {
	StartX int
	StartY int
	EndX   int
	EndY   int
}

func (a *CaptureArea) FixDimensionForH264() {
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
	Display     string
	CaptureArea *CaptureArea
}

func NewX11VideoStream(conf *VideoConfig) (MediaStream, error) {
	if err := gst.CheckPlugins([]string{"ximagesrc"}); err != nil {
		return nil, err
	}
	conf.CaptureArea.FixDimensionForH264()
	startX := conf.CaptureArea.StartX
	if startX < 0 {
		startX = 0
	}
	startY := conf.CaptureArea.StartY
	if startY < 0 {
		startY = 0
	}
	endX := conf.CaptureArea.EndX - 1
	if endX < 0 {
		endX = 0
	}
	endY := conf.CaptureArea.EndY - 1
	if endY < 0 {
		endY = 0
	}
	// why use-damage=0?: https://github.com/GoogleCloudPlatform/selkies-vdi/blob/0da21b7c9432bd5c99f1f9f7c541ac9c583f9ef4/images/gst-webrtc-app/gstwebrtc_app.py#L148
	src := fmt.Sprintf("ximagesrc display-name=%s remote=1 use-damage=0 startx=%d starty=%d endx=%d endy=%d",
		conf.CaptureDisplay, startX, startY, endX, endY)
	return NewX264VideoStream(src, conf.X264Param)
}

func NewVideoTestStream() (MediaStream, error) {
	return NewX264VideoStream("videotestsrc", "")
}

func NewAudioTestStream() (MediaStream, error) {
	return NewOpusAudioStream("audiotestsrc")
}

func NewX264VideoStream(src, x264params string) (MediaStream, error) {
	if err := gst.CheckPlugins([]string{"x264"}); err != nil {
		return nil, err
	}
	pipelineStr := fmt.Sprintf("%s ! videoconvert ! video/x-raw,format=I420 ! x264enc %s ! appsink name=video", src, x264params)
	return newGstStream(pipelineStr, "video")
}

func NewPulseAudioStream(conf *AudioConfig) (MediaStream, error) {
	if err := gst.CheckPlugins([]string{"pulseaudio"}); err != nil {
		return nil, err
	}
	return NewOpusAudioStream(fmt.Sprintf("pulsesrc server=%s", conf.PulseServer))
}

func NewOpusAudioStream(src string) (MediaStream, error) {
	if err := gst.CheckPlugins([]string{"opus"}); err != nil {
		return nil, err
	}
	pipelineStr := fmt.Sprintf("%s ! opusenc ! appsink name=audio", src)
	return newGstStream(pipelineStr, "audio")
}
