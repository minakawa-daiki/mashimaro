package p2p

import (
	"context"
	"fmt"
	"log"
	"math/rand"

	"github.com/castaneai/mashimaro/streamer"
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
	port := 9999 // TODO: dynamic port assign
	out, err := streamer.StartStreamingJPEGRTP(port)
	if err != nil {
		return fmt.Errorf("failed to start polling over RTP: %+v", err)
	}
	log.Printf("start video track serving... (udp port: %d)", port)
	go func() {
		for {
			select {
			case <-p.ctx.Done():
				return
			case data := <-out:
				if err := p.videoTrack.WriteSample(media.Sample{
					Data:    data,
					Samples: 90000,
				}); err != nil {
					log.Printf("failed to write video sample: %+v", err)
				}
			}
		}
	}()
	return nil
}
