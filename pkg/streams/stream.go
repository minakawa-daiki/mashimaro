package streams

import (
	"fmt"
	"io"
	"log"

	"github.com/notedit/gst"
)

type Stream interface {
	io.Closer
	ReadChunk() ([]byte, error)
}

type gstStream struct {
	gstPipeline *gst.Pipeline
	gstElement  *gst.Element
}

func (s *gstStream) Close() error {
	s.gstPipeline.SetState(gst.StateNull)
	return nil
}

func (s *gstStream) ReadChunk() ([]byte, error) {
	sample, err := s.gstElement.PullSample()
	if err != nil {
		if s.gstElement.IsEOS() {
			return nil, io.EOF
		}
		return nil, err
	}
	return sample.Data, nil
}

func GetX11Stream(displayName string) (Stream, error) {
	if err := gst.CheckPlugins([]string{"ximagesrc"}); err != nil {
		return nil, err
	}
	// why use-damage=0?: https://github.com/GoogleCloudPlatform/selkies-vdi/blob/0da21b7c9432bd5c99f1f9f7c541ac9c583f9ef4/images/gst-webrtc-app/gstwebrtc_app.py#L148
	return GetGstStream(fmt.Sprintf("ximagesrc display-name=%s remote=1 use-damage=0", displayName))
}

func GetVideoTestStream() (Stream, error) {
	return GetGstStream("videotestsrc")
}

func GetGstStream(src string) (Stream, error) {
	if err := gst.CheckPlugins([]string{"x264"}); err != nil {
		return nil, err
	}
	// TODO: x264enc && key-int-max > 1 does not work on Google Chrome on Mac OS
	// https://qiita.com/nakakura/items/87a5de9ba1a85eb39bc6
	x264params := fmt.Sprintf(`speed-preset=ultrafast tune=zerolatency byte-stream=true key-int-max=1 intra-refresh=true`)
	pipelineStr := fmt.Sprintf("%s ! videoconvert ! video/x-raw,format=I420 ! x264enc %s ! appsink name=out", src, x264params)
	log.Printf("starting gstreamer pipeline: %s", pipelineStr)
	pipeline, err := gst.ParseLaunch(pipelineStr)
	if err != nil {
		return nil, err
	}

	element := pipeline.GetByName("out")
	pipeline.SetState(gst.StatePlaying)
	return &gstStream{
		gstPipeline: pipeline,
		gstElement:  element,
	}, nil
}
