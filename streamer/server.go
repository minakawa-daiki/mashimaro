package streamer

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"

	"github.com/pion/webrtc/v2"
)

type service struct{}

func (s *service) ConnectClient(ctx context.Context, req *ConnectClientRequest) (*ConnectClientResponse, error) {
	offer, err := decodeOffer(req.SdpOffer)
	if err != nil {
		return nil, err
	}

	mediaEngine := webrtc.MediaEngine{}
	if err := mediaEngine.PopulateFromSDP(*offer); err != nil {
		return nil, fmt.Errorf("failed to populate from SDP: %+v", err)
	}
	var payloadType uint8
	for _, videoCodec := range mediaEngine.GetCodecsByKind(webrtc.RTPCodecTypeVideo) {
		if videoCodec.Name == "VP8" {
			payloadType = videoCodec.PayloadType
			break
		}
	}
	if payloadType == 0 {
		return nil, fmt.Errorf("Remote peer does not support VP8")
	}
	api := webrtc.NewAPI(webrtc.WithMediaEngine(mediaEngine))
	peerConnection, err := api.NewPeerConnection(webrtc.Configuration{
		ICEServers: []webrtc.ICEServer{
			{
				URLs: []string{"stun:stun.l.google.com:19302"},
			},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to new peer connection: %+v", err)
	}

	videoTrack, err := peerConnection.NewTrack(payloadType, rand.Uint32(), "video", "pion")
	if err != nil {
		return nil, err
	}
	if _, err := peerConnection.AddTrack(videoTrack); err != nil {
		return nil, err
	}
	peerConnection.OnICEConnectionStateChange(func(st webrtc.ICEConnectionState) {
		log.Printf("[peer: %v] connection state has changed: %s", peerConnection, st)
	})
	if err := peerConnection.SetRemoteDescription(*offer); err != nil {
		return nil, err
	}

	answer, err := peerConnection.CreateAnswer(nil)
	if err != nil {
		return nil, err
	}
	if err := peerConnection.SetLocalDescription(answer); err != nil {
		return nil, err
	}
	ans, err := encodeAnswer(&answer)
	if err != nil {
		return nil, fmt.Errorf("failed to encode answer: %+v", err)
	}
	return &ConnectClientResponse{
		SdpAnswer: ans,
	}, nil
}

func (s *service) ConnectServer(context.Context, *ConnectServerRequest) (*ConnectServerResponse, error) {
	panic("implement me")
}

func (s *service) StartStreaming(context.Context, *StartStreamingRequest) (*StartStreamingResponse, error) {
	panic("implement me")
}

func decodeOffer(in string) (*webrtc.SessionDescription, error) {
	b, err := base64.StdEncoding.DecodeString(in)
	if err != nil {
		return nil, err
	}
	var offer webrtc.SessionDescription
	if err := json.Unmarshal(b, &offer); err != nil {
		return nil, err
	}
	return &offer, nil
}

func encodeAnswer(answer *webrtc.SessionDescription) (string, error) {
	b, err := json.Marshal(answer)
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(b), nil
}
