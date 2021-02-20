package transport

import (
	"context"
	"fmt"

	"github.com/pion/webrtc/v3"

	"github.com/castaneai/mashimaro/pkg/ayame"
)

type WebRTCSignaler interface {
	Signaling(ctx context.Context, pc *webrtc.PeerConnection, roomID, clientID string) error
}

type AyameSignaler struct {
	AyameURL string
}

func NewAyameSignaler(ayameURL string) *AyameSignaler {
	return &AyameSignaler{AyameURL: ayameURL}
}

func (s *AyameSignaler) Signaling(ctx context.Context, pc *webrtc.PeerConnection, roomID, clientID string) error {
	ayamec := ayame.NewClient(pc)
	return ayamec.Connect(ctx, s.AyameURL, &ayame.ConnectRequest{RoomID: roomID, ClientID: clientID})
}

type AyameLaboSignaler struct {
	AyameLaboURL  string
	SignalingKey  string
	GitHubAccount string
}

func NewAyameLaboSignaler(ayameLaboURL, signalingKey, githubAccount string) *AyameLaboSignaler {
	return &AyameLaboSignaler{
		AyameLaboURL:  ayameLaboURL,
		SignalingKey:  signalingKey,
		GitHubAccount: githubAccount,
	}
}

func (s *AyameLaboSignaler) Signaling(ctx context.Context, pc *webrtc.PeerConnection, roomID, clientID string) error {
	ayamec := ayame.NewClient(pc)
	return ayamec.Connect(ctx, s.AyameLaboURL, &ayame.ConnectRequest{
		RoomID:       fmt.Sprintf("%s@%s", s.GitHubAccount, roomID),
		ClientID:     clientID,
		SignalingKey: s.SignalingKey,
	})
}
