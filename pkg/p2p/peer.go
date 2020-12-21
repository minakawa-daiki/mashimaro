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
	conn       *webrtc.PeerConnection
	videoTrack *webrtc.Track
	audioTrack *webrtc.Track
}

func NewPeer(conn *webrtc.PeerConnection, videoPayloadType, audioPayloadType uint8) (*Peer, error) {
	videoTrack, err := conn.NewTrack(videoPayloadType, rand.Uint32(), "video", "pion")
	if err != nil {
		return nil, err
	}
	if _, err = conn.AddTrack(videoTrack); err != nil {
		return nil, err
	}
	log.Printf("added video track (codec: %v)", videoTrack.Codec())

	audioTrack, err := conn.NewTrack(audioPayloadType, rand.Uint32(), "audio", "pion")
	if err != nil {
		return nil, err
	}
	if _, err := conn.AddTrack(audioTrack); err != nil {
		return nil, err
	}
	log.Printf("added audio track (codec: %v)", audioTrack.Codec())

	conn.OnConnectionStateChange(func(state webrtc.PeerConnectionState) {
		log.Printf("peer state change: %v", state)
	})
	return &Peer{
		conn:       conn,
		videoTrack: videoTrack,
		audioTrack: audioTrack,
	}, nil
}

func (p *Peer) Start(ctx context.Context) {
	go func() {
		if err := p.startServingVideo(ctx); err != nil {
			log.Printf("failed to serve video: %+v", err)
		}
	}()
	if err := p.startServingAudio(ctx); err != nil {
		log.Printf("failed to serve audio: %+v", err)
	}
}

func (p *Peer) Close() {
	if err := p.conn.Close(); err != nil {
		log.Printf("failed to close conn: %+v", err)
	}
}

func (p *Peer) startServingVideo(ctx context.Context) error {
	stream, err := streams.GetX11VideoStream(":0")
	if err != nil {
		return fmt.Errorf("failed to start media stream: %+v", err)
	}
	defer func() { _ = stream.Close() }()
	log.Printf("start video track serving... (sampling rate: %v)", p.videoTrack.Codec().ClockRate)

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			chunk, err := stream.ReadChunk()
			if err != nil {
				return fmt.Errorf("failed to read chunk from stream: %+v", err)
			}
			if err := p.videoTrack.WriteSample(media.Sample{
				Data:    chunk.Data,
				Samples: p.videoTrack.Codec().ClockRate,
			}); err != nil {
				return fmt.Errorf("failed to write video sample: %+v", err)
			}
		}
	}
}

func (p *Peer) startServingAudio(ctx context.Context) error {
	stream, err := streams.GetOpusAudioStream()
	if err != nil {
		return fmt.Errorf("failed to start audio stream: %+v", err)
	}
	defer func() { _ = stream.Close() }()
	log.Printf("start audio track serving... (sampling rate: %v)", p.audioTrack.Codec().ClockRate)

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			chunk, err := stream.ReadChunk()
			if err != nil {
				return fmt.Errorf("failed to read chunk from audio stream: %+v", err)
			}
			if err := p.audioTrack.WriteSample(media.Sample{
				Data:    chunk.Data,
				Samples: p.audioTrack.Codec().ClockRate,
			}); err != nil {
				return fmt.Errorf("failed to write audio sample: %+v", err)
			}
		}
	}
}
