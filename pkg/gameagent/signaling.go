package gameagent

import (
	"context"

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
