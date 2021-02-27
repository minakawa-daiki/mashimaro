package streamerserver

import (
	"context"
	"log"
	"net"
	"sync"

	"github.com/castaneai/mashimaro/pkg/proto"
)

type streamerServer struct {
	gstServers   map[string]*GstServer
	gstServersMu sync.Mutex
}

func NewStreamerServer() proto.StreamerServer {
	return &streamerServer{
		gstServers: map[string]*GstServer{},
	}
}

func (s *streamerServer) StartStreaming(ctx context.Context, req *proto.StartStreamingRequest) (*proto.StartStreamingResponse, error) {
	s.stopStreaming(req.MediaId)
	addr, err := s.startGstStreaming(req.MediaId, req.GstPipeline, int(req.Port))
	if err != nil {
		return nil, err
	}
	return &proto.StartStreamingResponse{ListenPort: uint32(addr.Port)}, nil
}

func (s *streamerServer) startGstStreaming(mediaID, pipelineStr string, port int) (*net.TCPAddr, error) {
	gs, err := StartGstServer(pipelineStr, port)
	if err != nil {
		return nil, err
	}
	s.gstServersMu.Lock()
	defer s.gstServersMu.Unlock()
	s.gstServers[mediaID] = gs
	log.Printf("gst pipeline started (mediaID: %s, %s)", mediaID, pipelineStr)
	return gs.Addr().(*net.TCPAddr), nil
}

func (s *streamerServer) stopStreaming(mediaID string) {
	s.gstServersMu.Lock()
	defer s.gstServersMu.Unlock()
	gs, ok := s.gstServers[mediaID]
	if ok {
		gs.Stop()
		delete(s.gstServers, mediaID)
		log.Printf("gst pipeline stopped (mediaID: %s, gst: %v)", mediaID, gs)
	}
}
