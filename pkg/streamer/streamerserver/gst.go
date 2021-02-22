package streamerserver

import (
	"fmt"
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
	g.mu.Lock()
	defer g.mu.Unlock()
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
	lis := g.lis
	g.mu.Unlock()
	log.Printf("waiting for connection on %v...", lis.Addr())
	conn, err := lis.Accept()
	if err != nil {
		if strings.Contains(err.Error(), "use of closed network connection") {
			return nil
		}
		return errors.Wrap(err, "failed to accept conn")
	}
	g.mu.Lock()
	g.conn = conn
	g.mu.Unlock()
	log.Printf("accepted new conn")
	if err := g.startPipeline(); err != nil {
		return err
	}
	log.Printf("pipeline started: %s", pipelineStr)
	return g.serveSample(conn, src)
}

func (g *GstServer) setPipelineStateLocked(state gst.StateOptions) error {
	ret := g.pipeline.SetState(state)
	switch ret {
	case gst.StateChangeSuccess:
		return nil
	case gst.StateChangeAsync:
		// block until done
		g.pipeline.GetBus().Pull(gst.MessageAsyncDone)
		return nil
	default:
		return fmt.Errorf("failed to set state to playing (return: %v)", ret)
	}
}

func (g *GstServer) serveSample(w io.Writer, src *gst.Element) error {
	defer g.stopPipeline()
	for {
		sample, err := src.PullSample()
		if err != nil {
			if src.IsEOS() {
				return errors.New("received EOS when trying to pull sample")
			}
			return errors.Wrap(err, "failed to pull sample")
		}
		packet := streamerproto.SamplePacket{
			Data:     sample.Data,
			Duration: time.Duration(sample.Duration),
		}
		if err := streamerproto.WriteSamplePacket(w, &packet); err != nil {
			if errors.Is(err, syscall.EPIPE) || errors.Is(err, syscall.ECONNRESET) {
				log.Printf("streaming client disconnected")
				return nil
			}
			return errors.Wrap(err, "failed to write sample packet")
		}
	}
}

func (g *GstServer) startPipeline() error {
	g.mu.Lock()
	defer g.mu.Unlock()
	if err := g.setPipelineStateLocked(gst.StatePlaying); err != nil {
		return fmt.Errorf("failed to stop pipeline: %+v", err)
	}
	return nil
}

func (g *GstServer) stopPipeline() {
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.pipeline == nil {
		return
	}
	if err := g.setPipelineStateLocked(gst.StateNull); err != nil {
		log.Printf("failed to stop pipeline: %+v", err)
	}
	g.pipeline = nil
	log.Printf("pipeline stopped")
}

func (g *GstServer) Stop() {
	g.stopPipeline()
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.conn != nil {
		_ = g.conn.Close()
		g.conn = nil
	}
	if g.lis != nil {
		if err := g.lis.Close(); err != nil {
			if !strings.Contains(err.Error(), "use of closed network connection") {
				log.Printf("failed to close listener: %+v", err)
			}
		}
		g.lis = nil
		log.Printf("listener was closed")
	}
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
