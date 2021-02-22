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
		s.videoServer = nil
	}
}
