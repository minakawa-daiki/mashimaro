package streamer

import (
	"context"
	"log"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/castaneai/mashimaro/pkg/transport"
)

type mockConn struct {
	videoSampleCh chan []byte
	audioSampleCh chan []byte
}

func newMockConn() *mockConn {
	return &mockConn{
		videoSampleCh: make(chan []byte),
		audioSampleCh: make(chan []byte),
	}
}

func (c *mockConn) ConnectionID() string {
	return "mockConn"
}

func (c *mockConn) SendMessage(ctx context.Context, data []byte) error {
	panic("implement me")
}

func (c *mockConn) OnMessage(f func(data []byte)) {
	panic("implement me")
}

func (c *mockConn) SendVideoSample(ctx context.Context, sample transport.MediaSample) error {
	select {
	case <-ctx.Done():
		return nil
	case c.videoSampleCh <- sample.Data:
	}
	return nil
}

func (c *mockConn) SendAudioSample(ctx context.Context, sample transport.MediaSample) error {
	select {
	case <-ctx.Done():
		return nil
	case c.audioSampleCh <- sample.Data:
	}
	return nil
}

func (c *mockConn) OnConnect(f func()) {
	panic("implement me")
}

func (c *mockConn) OnDisconnect(f func()) {
	panic("implement me")
}

func TestStreamer(t *testing.T) {
	conn := newMockConn()
	s := NewStreamer(conn)
	ctx := context.Background()
	vc := &VideoConfig{
		CaptureDisplay: os.Getenv("DISPLAY"),
		CaptureArea:    CaptureArea{StartX: 0, StartY: 0, EndX: 100, EndY: 100},
		X264Param:      defaultX264Params,
	}
	ac := &AudioConfig{PulseServer: "localhost:4713"}
	go func() {
		if err := s.Start(ctx, vc, ac); err != nil {
			log.Printf("failed to start: %+v", err)
		}
	}()
	assert.True(t, len(<-conn.videoSampleCh) > 0)
	assert.True(t, len(<-conn.audioSampleCh) > 0)

	vc2 := &VideoConfig{
		CaptureDisplay: vc.CaptureDisplay,
		CaptureArea:    CaptureArea{StartX: 0, StartY: 0, EndX: 200, EndY: 200},
		X264Param:      defaultX264Params,
	}
	s.RestartVideo(vc2)
	time.Sleep(500 * time.Millisecond)
}
