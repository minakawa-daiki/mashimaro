package encoder

import (
	"fmt"
	"io"
	"log"
	"net"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/castaneai/mashimaro/pkg/encoder/encoderproto"

	"github.com/notedit/gst"
	"github.com/pkg/errors"
)

func startGstServer(pipelineStr string, port int) (*GstServer, error) {
	lis, err := listenTCP(port)
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
	lis         net.Listener
	pipelineStr string
	pipeline    *gst.Pipeline
	conn        net.Conn
	mu          sync.Mutex
}

func newGstServer(lis net.Listener) *GstServer {
	return &GstServer{
		lis: lis,
	}
}

func (g *GstServer) String() string {
	g.mu.Lock()
	defer g.mu.Unlock()
	return fmt.Sprintf("%T(pipeline: %s)", g, g.pipelineStr)
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
		return errors.Wrapf(err, "failed to parse pipeline str: %s", pipelineStr)
	}
	src := pipeline.GetByName("out")
	g.mu.Lock()
	g.pipelineStr = pipelineStr
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
		packet := encoderproto.SamplePacket{
			Data:     sample.Data,
			Duration: time.Duration(sample.Duration),
		}
		if err := encoderproto.WriteSamplePacket(w, &packet); err != nil {
			if errors.Is(err, syscall.EPIPE) || errors.Is(err, syscall.ECONNRESET) {
				log.Printf("media data client disconnected")
				return nil
			}
			return errors.Wrap(err, "failed to write sample packet")
		}
	}
}

func (g *GstServer) startPipeline() error {
	g.mu.Lock()
	defer g.mu.Unlock()
	go func() {
		bus := g.pipeline.GetBus()
		for {
			msg := bus.Pull(gst.MessageAny)
			if msg.GetType() != gst.MessageStateChanged {
				s := msg.GetName()
				if st := msg.GetStructure(); st.C != nil {
					s = strings.ReplaceAll(st.ToString(), "\\", "")
				}
				log.Printf("[gst] %s", s)
			}
		}

	}()
	if err := g.setPipelineStateLocked(gst.StatePlaying); err != nil {
		return fmt.Errorf("failed to stop pipeline: %+v (%s)", err, g.pipelineStr)
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
		log.Printf("failed to stop pipeline: %+v (%s)", err, g.pipelineStr)
	}
	g.pipeline = nil
	log.Printf("pipeline stopped: %s", g.pipelineStr)
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
		addr := g.lis.Addr()
		if err := g.lis.Close(); err != nil {
			if !strings.Contains(err.Error(), "use of closed network connection") {
				log.Printf("failed to close listener: %+v", err)
			}
		}
		g.lis = nil
		log.Printf("media data connection listener was closed: %v", addr)
	}
}

func listenTCP(port int) (*net.TCPListener, error) {
	addr, err := net.ResolveTCPAddr("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		return nil, err
	}
	lis, err := net.ListenTCP("tcp", addr)
	if err != nil {
		return nil, err
	}
	return lis, nil
}
