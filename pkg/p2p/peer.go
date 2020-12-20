package p2p

import (
	"context"
	"fmt"
	"log"
	"math/rand"

	"github.com/castaneai/mashimaro/pkg/streams"

	"github.com/pion/webrtc/v2"
	"github.com/pion/webrtc/v2/pkg/media"
)

type Peer struct {
	ctx        context.Context
	cancel     context.CancelFunc
	conn       *webrtc.PeerConnection
	videoTrack *webrtc.Track
}

func NewPeer(conn *webrtc.PeerConnection, payloadType uint8) (*Peer, error) {
	videoTrack, err := conn.NewTrack(payloadType, rand.Uint32(), "video", "pion")
	if err != nil {
		return nil, err
	}
	if _, err = conn.AddTrack(videoTrack); err != nil {
		return nil, err
	}
	log.Printf("video track added (codec: %v)", videoTrack.Codec())
	conn.OnConnectionStateChange(func(state webrtc.PeerConnectionState) {
		log.Printf("peer state change: %v", state)
	})
	ctx, cancel := context.WithCancel(context.Background())
	return &Peer{ctx: ctx, cancel: cancel, conn: conn, videoTrack: videoTrack}, nil
}

func (p *Peer) Start() error {
	if err := p.startServingVideo(); err != nil {
		return err
	}
	return nil
}

func (p *Peer) Close() {
	if err := p.conn.Close(); err != nil {
		log.Printf("failed to close conn: %+v", err)
	}
	p.cancel()
}

func (p *Peer) startServingVideo() error {
	stream, err := streams.GetX11Stream(":0")
	if err != nil {
		return fmt.Errorf("failed to start media stream: %+v", err)
	}
	defer func() { _ = stream.Close() }()
	log.Printf("start video track serving...")

	for {
		chunk, err := stream.ReadChunk()
		if err != nil {
			return fmt.Errorf("failed to read chunk from stream: %+v", err)
		}
		if err := p.videoTrack.WriteSample(media.Sample{
			Data:    chunk,
			Samples: 90000, // TODO: why 90000?
		}); err != nil {
			return fmt.Errorf("failed to write video sample: %+v", err)
		}
	}
}
