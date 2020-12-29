package streamer

import (
	"fmt"
	"io"
	"log"
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
	s.gstPipeline.SetState(gst.StatePlaying)
}

func (s *gstStream) Close() error {
	log.Printf("Stopping GStreamer pipeline: %s", s.pipelineStr)
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

func NewX11VideoStream(displayName string) (MediaStream, error) {
	if err := gst.CheckPlugins([]string{"ximagesrc"}); err != nil {
		return nil, err
	}
	// why use-damage=0?: https://github.com/GoogleCloudPlatform/selkies-vdi/blob/0da21b7c9432bd5c99f1f9f7c541ac9c583f9ef4/images/gst-webrtc-app/gstwebrtc_app.py#L148
	return NewH264VideoStream(fmt.Sprintf("ximagesrc display-name=%s remote=1 use-damage=0", displayName))
}

func NewVideoTestStream() (MediaStream, error) {
	return NewH264VideoStream("videotestsrc")
}

func NewAudioTestStream() (MediaStream, error) {
	return NewOpusAudioStream("audiotestsrc")
}

func NewH264VideoStream(src string) (MediaStream, error) {
	if err := gst.CheckPlugins([]string{"x264"}); err != nil {
		return nil, err
	}
	// TODO: x264enc && key-int-max > 1 does not work on Google Chrome on Mac OS
	// https://qiita.com/nakakura/items/87a5de9ba1a85eb39bc6
	x264params := fmt.Sprintf(`speed-preset=ultrafast tune=zerolatency byte-stream=true key-int-max=1 intra-refresh=true`)
	pipelineStr := fmt.Sprintf("%s ! videoconvert ! video/x-raw,format=I420 ! x264enc %s ! appsink name=video", src, x264params)
	return newGstStream(pipelineStr, "video")
}

func NewPulseAudioStream(pulseServer string) (MediaStream, error) {
	if err := gst.CheckPlugins([]string{"pulseaudio"}); err != nil {
		return nil, err
	}
	return NewOpusAudioStream(fmt.Sprintf("pulsesrc server=%s", pulseServer))
}

func NewOpusAudioStream(src string) (MediaStream, error) {
	if err := gst.CheckPlugins([]string{"opus"}); err != nil {
		return nil, err
	}
	pipelineStr := fmt.Sprintf("%s ! opusenc ! appsink name=audio", src)
	return newGstStream(pipelineStr, "audio")
}
