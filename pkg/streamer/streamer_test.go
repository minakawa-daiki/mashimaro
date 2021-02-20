package streamer

import (
	"context"

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
