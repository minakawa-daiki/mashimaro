package streams

import (
	"fmt"
	"io"
	"log"
	"time"

	"github.com/notedit/gst"
)

type Chunk struct {
	Data     []byte
	Duration time.Duration
}

type Stream interface {
	io.Closer
	ReadChunk() (*Chunk, error)
}

type gstStream struct {
	gstPipeline *gst.Pipeline
	gstElement  *gst.Element
}

func (s *gstStream) Close() error {
	s.gstPipeline.SetState(gst.StateNull)
	return nil
}

func (s *gstStream) ReadChunk() (*Chunk, error) {
	sample, err := s.gstElement.PullSample()
	if err != nil {
		if s.gstElement.IsEOS() {
			return nil, io.EOF
		}
		return nil, err
	}
	s.gstElement.GetClock().GetClockTime()
	return &Chunk{
		Data:     sample.Data,
		Duration: time.Duration(sample.Duration),
	}, nil
}

func GetX11VideoStream(displayName string) (Stream, error) {
	if err := gst.CheckPlugins([]string{"ximagesrc"}); err != nil {
		return nil, err
	}
	// why use-damage=0?: https://github.com/GoogleCloudPlatform/selkies-vdi/blob/0da21b7c9432bd5c99f1f9f7c541ac9c583f9ef4/images/gst-webrtc-app/gstwebrtc_app.py#L148
	return GetGstVideoStream(fmt.Sprintf("ximagesrc display-name=%s remote=1 use-damage=0", displayName))
}

func GetVideoTestStream() (Stream, error) {
	return GetGstVideoStream("videotestsrc")
}

func GetGstVideoStream(src string) (Stream, error) {
	if err := gst.CheckPlugins([]string{"x264"}); err != nil {
		return nil, err
	}
	// TODO: x264enc && key-int-max > 1 does not work on Google Chrome on Mac OS
	// https://qiita.com/nakakura/items/87a5de9ba1a85eb39bc6
	x264params := fmt.Sprintf(`speed-preset=ultrafast tune=zerolatency byte-stream=true key-int-max=1 intra-refresh=true`)
	pipelineStr := fmt.Sprintf("%s ! videoconvert ! video/x-raw,format=I420 ! x264enc %s ! appsink name=video", src, x264params)
	log.Printf("starting gstreamer pipeline: %s", pipelineStr)
	pipeline, err := gst.ParseLaunch(pipelineStr)
	if err != nil {
		return nil, err
	}

	element := pipeline.GetByName("video")
	pipeline.SetState(gst.StatePlaying)
	return &gstStream{
		gstPipeline: pipeline,
		gstElement:  element,
	}, nil
}

func GetOpusAudioStream() (Stream, error) {
	if err := gst.CheckPlugins([]string{"pulseaudio", "opus"}); err != nil {
		return nil, err
	}
	pipelineStr := fmt.Sprintf("pulsesrc server=localhost:4713 ! opusenc ! appsink name=audio")
	log.Printf("starting gstreamer pipeline: %s", pipelineStr)
	pipeline, err := gst.ParseLaunch(pipelineStr)
	if err != nil {
		return nil, err
	}

	element := pipeline.GetByName("audio")
	pipeline.SetState(gst.StatePlaying)
	return &gstStream{
		gstPipeline: pipeline,
		gstElement:  element,
	}, nil
}
