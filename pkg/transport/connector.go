package transport

import (
	"context"

	"github.com/pkg/errors"

	"github.com/pion/webrtc/v3"
)

type Connector interface {
	Connect(ctx context.Context, conn StreamerConn) error
}

type WebRTCConnector struct {
	signaler WebRTCSignaler
	roomID   string
	clientID string
}

func NewWebRTCConnector(signaler WebRTCSignaler, roomID, clientID string) *WebRTCConnector {
	return &WebRTCConnector{
		signaler: signaler,
		roomID:   roomID,
		clientID: clientID,
	}
}

func (l *WebRTCConnector) Connect(ctx context.Context, conn StreamerConn) error {
	wconn, ok := conn.(interface{ PeerConnection() *webrtc.PeerConnection })
	if !ok {
		return errors.New("failed to cast to WebRTCConn")
	}
	return l.signaler.Signaling(ctx, wconn.PeerConnection(), l.roomID, l.clientID)
}
