package streamerserver

import (
	"context"
	"net"
	"sync"

	"github.com/castaneai/mashimaro/pkg/proto"
)

type streamerServer struct {
	videoServer *GstServer
	videoMu     sync.Mutex
	audioServer *GstServer
	audioMu     sync.Mutex
}

func NewStreamerServer() proto.StreamerServer {
	return &streamerServer{}
}

func (s *streamerServer) StartVideoStreaming(ctx context.Context, req *proto.StartVideoStreamingRequest) (*proto.StartVideoStreamingResponse, error) {
	s.stopVideoStreaming()
	if err := s.startVideoStreaming(req.GstPipeline); err != nil {
		return nil, err
	}
	s.videoMu.Lock()
	defer s.videoMu.Unlock()
	addr := s.videoServer.Addr().(*net.TCPAddr)
	return &proto.StartVideoStreamingResponse{ListenPort: uint32(addr.Port)}, nil
}

func (s *streamerServer) StartAudioStreaming(ctx context.Context, req *proto.StartAudioStreamingRequest) (*proto.StartAudioStreamingResponse, error) {
	s.stopAudioStreaming()
	if err := s.startAudioStreaming(req.GstPipeline); err != nil {
		return nil, err
	}
	s.audioMu.Lock()
	defer s.audioMu.Unlock()
	addr := s.audioServer.Addr().(*net.TCPAddr)
	return &proto.StartAudioStreamingResponse{ListenPort: uint32(addr.Port)}, nil
}

func (s *streamerServer) startVideoStreaming(pipelineStr string) error {
	gs, err := StartGstServerWithRandomPort(pipelineStr)
	if err != nil {
		return err
	}
	s.videoMu.Lock()
	defer s.videoMu.Unlock()
	s.videoServer = gs
	return nil
}

func (s *streamerServer) stopVideoStreaming() {
	s.videoMu.Lock()
	defer s.videoMu.Unlock()
	if s.videoServer != nil {
		s.videoServer.Stop()
	}
}

func (s *streamerServer) startAudioStreaming(pipelineStr string) error {
	gs, err := StartGstServerWithRandomPort(pipelineStr)
	if err != nil {
		return err
	}
	s.audioMu.Lock()
	defer s.audioMu.Unlock()
	s.audioServer = gs
	return nil
}

func (s *streamerServer) stopAudioStreaming() {
	s.audioMu.Lock()
	defer s.audioMu.Unlock()
	if s.audioServer != nil {
		s.audioServer.Stop()
	}
}
