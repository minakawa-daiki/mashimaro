package streamerserver

import (
	"io"
	"log"
	"net"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/castaneai/mashimaro/pkg/streamer/streamerproto"

	"github.com/notedit/gst"
	"github.com/pkg/errors"
)

func StartGstServerWithRandomPort(pipelineStr string) (*GstServer, error) {
	lis, err := listenTCPWithRandomPort()
	if err != nil {
		return nil, err
	}
	gs := newGstServer(lis)
	go func() {
		defer gs.Stop()
		if err := gs.Serve(pipelineStr); err != nil {
			log.Printf("failed to serve: %+v", err)
		}
	}()
	return gs, nil
}

type GstServer struct {
	lis      net.Listener
	pipeline *gst.Pipeline
	conn     net.Conn
	mu       sync.Mutex
}

func newGstServer(lis net.Listener) *GstServer {
	return &GstServer{
		lis: lis,
	}
}

func (g *GstServer) Addr() net.Addr {
	return g.lis.Addr()
}

func (g *GstServer) Serve(pipelineStr string) error {
	pipelineStr += " ! appsink name=out"
	pipeline, err := gst.ParseLaunch(pipelineStr)
	if err != nil {
		return errors.Wrap(err, "failed to parse pipeline str")
	}
	src := pipeline.GetByName("out")
	g.mu.Lock()
	g.pipeline = pipeline
	g.mu.Unlock()
	for {
		log.Printf("waiting for connection...")
		conn, err := g.lis.Accept()
		if err != nil {
			if strings.Contains(err.Error(), "use of closed network connection") {
				return nil
			}
			log.Printf("failed to accept conn: %+v", err)
			continue
		}
		g.mu.Lock()
		g.conn = conn
		g.mu.Unlock()
		log.Printf("accepted new conn")
		if err := g.setPipelineState(gst.StatePlaying); err != nil {
			log.Printf("failed to set pipeline state: %+v", err)
			continue
		}
		log.Printf("pipeline started: %s", pipelineStr)
		g.serveSample(conn, src)
	}
}

func (g *GstServer) setPipelineState(state gst.StateOptions) error {
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.pipeline == nil {
		return nil
	}
	ret := g.pipeline.SetState(state)
	switch ret {
	case gst.StateChangeSuccess:
		return nil
	case gst.StateChangeAsync:
		// block until done
		g.pipeline.GetBus().Pull(gst.MessageAsyncDone)
		return nil
	default:
		return errors.New("failed to set state to playing")
	}
}

func (g *GstServer) serveSample(w io.Writer, src *gst.Element) {
	defer g.stopPipeline()
	for {
		sample, err := src.PullSample()
		if err != nil {
			log.Printf("failed to pull sample: %+v", err)
			return
		}
		packet := streamerproto.SamplePacket{
			Data:     sample.Data,
			Duration: time.Duration(sample.Duration),
		}
		if err := streamerproto.WriteSamplePacket(w, &packet); err != nil {
			if errors.Is(err, syscall.EPIPE) {
				return
			}
			log.Printf("failed to write sample packet: %+v", err)
			return
		}
	}
}

func (g *GstServer) stopPipeline() {
	if err := g.setPipelineState(gst.StateNull); err != nil {
		log.Printf("failed to stop pipeline: %+v", err)
	}
	log.Printf("pipeline stopped")
}

func (g *GstServer) Stop() {
	g.stopPipeline()
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.conn != nil {
		_ = g.conn.Close()
	}
	if err := g.lis.Close(); err != nil {
		log.Printf("faild to close listener")
	}
	log.Printf("listener was closed")
}

func listenTCPWithRandomPort() (*net.TCPListener, error) {
	addr, err := net.ResolveTCPAddr("tcp", ":0")
	if err != nil {
		return nil, err
	}
	lis, err := net.ListenTCP("tcp", addr)
	if err != nil {
		return nil, err
	}
	return lis, nil
}
