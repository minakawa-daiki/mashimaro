package testutils

import "github.com/pion/webrtc/v3"

type Peers struct {
	Streamer            *webrtc.PeerConnection
	Player              *webrtc.PeerConnection
	StreamerDataChannel *webrtc.DataChannel
	PlayerDataChannel   *webrtc.DataChannel
}
