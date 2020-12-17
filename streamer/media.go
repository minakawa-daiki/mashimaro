package streamer

import (
	"fmt"
	"github.com/notedit/gst"
	"io"
)

type MediaStream interface {
	io.Closer
	ReadChunk() ([]byte, error)
}

type gstMediaStream struct {
	gstPipeline *gst.Pipeline
	gstElement *gst.Element
}

func (s *gstMediaStream) Close() error {
	s.gstPipeline.SetState(gst.StateNull)
	return nil
}

func (s *gstMediaStream) ReadChunk() ([]byte, error) {
	sample, err := s.gstElement.PullSample()
	if err != nil {
		if s.gstElement.IsEOS() {
			return nil, io.EOF
		}
		return nil, err
	}
	return sample.Data, nil
}

func GetX11Stream() (MediaStream, error) {
	return GetGstStream("ximagesrc")
}

func GetVideoTestStream() (MediaStream, error) {
	return GetGstStream("videotestsrc")
}

func GetGstStream(pipelineStr string) (MediaStream, error) {
	pipeline, err := gst.ParseLaunch(fmt.Sprintf("%s ! appsink name=out", pipelineStr))
	if err != nil {
		return nil, err
	}
	element := pipeline.GetByName("out")
	pipeline.SetState(gst.StatePlaying)
	return &gstMediaStream{
		gstPipeline: pipeline,
		gstElement:  element,
	}, nil
}

