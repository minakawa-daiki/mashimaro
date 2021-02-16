package gameagent

import (
	"context"
	"fmt"

	"github.com/pion/webrtc/v3"

	"github.com/castaneai/mashimaro/pkg/ayame"
	"github.com/castaneai/mashimaro/pkg/transport"
	"github.com/pkg/errors"
)

type Signaler interface {
	Signaling(ctx context.Context, conn transport.Conn, rid, cid string) error
}

type AyameSignaler struct {
	AyameURL string
}

func NewAyameSignaler(ayameURL string) *AyameSignaler {
	return &AyameSignaler{AyameURL: ayameURL}
}

func (s *AyameSignaler) Signaling(ctx context.Context, conn transport.Conn, rid, cid string) error {
	wconn, ok := conn.(interface{ PeerConnection() *webrtc.PeerConnection })
	if !ok {
		return errors.New("failed to cast conn to WebRTCConn")
	}
	ayamec := ayame.NewClient(wconn.PeerConnection())
	return ayamec.Connect(ctx, s.AyameURL, &ayame.ConnectRequest{RoomID: rid, ClientID: cid})
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

func (s *AyameLaboSignaler) Signaling(ctx context.Context, conn transport.Conn, rid, cid string) error {
	wconn, ok := conn.(interface{ PeerConnection() *webrtc.PeerConnection })
	if !ok {
		return errors.New("failed to cast conn to WebRTCConn")
	}
	ayamec := ayame.NewClient(wconn.PeerConnection())
	return ayamec.Connect(ctx, s.AyameLaboURL, &ayame.ConnectRequest{
		RoomID:       fmt.Sprintf("%s@%s", s.GitHubAccount, rid),
		ClientID:     cid,
		SignalingKey: s.SignalingKey,
	})
}
