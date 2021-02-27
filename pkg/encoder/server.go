package encoder

import (
	"context"
	"log"
	"net"
	"sync"

	"github.com/castaneai/mashimaro/pkg/proto"
)

type encoderServer struct {
	gstServers map[string]*GstServer
	mu         sync.Mutex
}

func NewEncoderServer() proto.EncoderServer {
	return &encoderServer{
		gstServers: map[string]*GstServer{},
	}
}

func (s *encoderServer) StartEncoding(ctx context.Context, req *proto.StartEncodingRequest) (*proto.StartEncodingResponse, error) {
	s.stopGstServer(req.PipelineId)
	addr, err := s.startGstServer(req.PipelineId, req.GstPipeline, int(req.Port))
	if err != nil {
		return nil, err
	}
	return &proto.StartEncodingResponse{ListenPort: uint32(addr.Port)}, nil
}

func (s *encoderServer) startGstServer(pipelineID, pipelineStr string, port int) (*net.TCPAddr, error) {
	gs, err := startGstServer(pipelineStr, port)
	if err != nil {
		return nil, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.gstServers[pipelineID] = gs
	log.Printf("gst pipeline started (pipelineID: %s, %s)", pipelineID, pipelineStr)
	return gs.Addr().(*net.TCPAddr), nil
}

func (s *encoderServer) stopGstServer(pipelineID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	gs, ok := s.gstServers[pipelineID]
	if ok {
		gs.Stop()
		delete(s.gstServers, pipelineID)
		log.Printf("gst pipeline stopped (pipelineID: %s, gst: %v)", pipelineID, gs)
	}
}
