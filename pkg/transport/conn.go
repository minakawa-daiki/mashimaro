package transport

import (
	"context"
	"time"
)

type Conn interface {
	ConnectionID() string
	SendMessage(ctx context.Context, data []byte) error
	OnMessage(f func(data []byte))
	OnConnect(f func())
	OnDisconnect(f func())
}

type StreamerConn interface {
	Conn
	SendVideoSample(ctx context.Context, sample MediaSample) error
	SendAudioSample(ctx context.Context, sample MediaSample) error
}

type MediaSample struct {
	Data     []byte
	Duration time.Duration
}

type PlayerConn interface {
	Conn
}
