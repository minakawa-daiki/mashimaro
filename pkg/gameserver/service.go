package gameserver

import (
	"context"
	"log"

	"github.com/castaneai/mashimaro/pkg/streamer"

	"github.com/pion/webrtc/v3"

	"github.com/castaneai/mashimaro/pkg/internal/webrtcutil"

	"github.com/castaneai/mashimaro/pkg/proto"
)

type gameServerService struct {
	handler     *connEventHandler
	videoSource streamer.MediaStream
	audioSource streamer.MediaStream
}

func NewGameServerService(videoSource, audioSource streamer.MediaStream) *gameServerService {
	s := &gameServerService{
		videoSource: videoSource,
		audioSource: audioSource,
	}
	handler := &connEventHandler{
		InitConnection:  s.initConnection,
		OfferCandidate:  make(chan webrtc.ICECandidateInit),
		AnswerCandidate: make(chan webrtc.ICECandidateInit),
	}
	s.handler = handler
	return s
}

func (s *gameServerService) initConnection(pc *webrtc.PeerConnection) error {
	tracks, err := streamer.NewMediaTracks()
	if err != nil {
		return err
	}
	if _, err := pc.AddTrack(tracks.VideoTrack); err != nil {
		return err
	}
	if _, err := pc.AddTrack(tracks.AudioTrack); err != nil {
		return err
	}

	ctx, cancel := context.WithCancel(context.Background())
	pc.OnConnectionStateChange(func(state webrtc.PeerConnectionState) {
		log.Printf("[pcAnswer] connection state has changed: %s", state)
		switch state {
		case webrtc.PeerConnectionStateConnected:
			s.onConnected(ctx, tracks)
		case webrtc.PeerConnectionStateDisconnected:
			cancel()
		}
	})
	return nil
}

func (s *gameServerService) onConnected(ctx context.Context, tracks *streamer.MediaTracks) {
	str := streamer.NewMediaStreamer(s.videoSource, tracks.VideoTrack, s.audioSource, tracks.AudioTrack)
	go func() {
		defer s.videoSource.Close()
		defer s.videoSource.Close()
		s.videoSource.Start()
		s.audioSource.Start()
		str.Start(ctx)
	}()
}

func (s *gameServerService) FirstSignaling(ctx context.Context, req *proto.Offer) (*proto.Answer, error) {
	offer, err := webrtcutil.DecodeSDP(req.Body)
	if err != nil {
		return nil, err
	}
	answer, err := startSignaling(ctx, *offer, s.handler)
	if err != nil {
		return nil, err
	}
	answerBody, err := webrtcutil.EncodeSDP(answer)
	if err != nil {
		return nil, err
	}
	return &proto.Answer{Body: answerBody}, nil
}

func (s *gameServerService) TrickleSignaling(stream proto.GameServer_TrickleSignalingServer) error {
	go func() {
		for {
			select {
			case <-stream.Context().Done():
				return
			case candidate := <-s.handler.AnswerCandidate:
				body, err := webrtcutil.EncodeICECandidate(&candidate)
				if err != nil {
					log.Printf("failed to encode ICE candidate: %+v", err)
					return
				}
				if err := stream.Send(&proto.ICECandidate{Body: body}); err != nil {
					log.Printf("failed to send ice candidate from pcAnswer to pcOffer: %+v", err)
					return
				}
			}
		}
	}()
	for {
		recv, err := stream.Recv()
		if err != nil {
			return err
		}
		candidate, err := webrtcutil.DecodeICECandidate(recv.Body)
		if err != nil {
			log.Printf("failed to decode ICE candidate: %+v", err)
			continue
		}
		s.handler.OfferCandidate <- *candidate
	}
}
