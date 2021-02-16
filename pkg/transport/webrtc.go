package transport

import (
	"context"
	"log"
	"sync"

	"github.com/pkg/errors"

	"github.com/tevino/abool"

	"github.com/pion/webrtc/v3"
	"github.com/pion/webrtc/v3/pkg/media"
)

type WebRTCConn struct {
	cid        string
	pc         *webrtc.PeerConnection
	dc         *webrtc.DataChannel
	dcMu       sync.Mutex
	onMessage  func([]byte)
	onConnect  func()
	connected  *abool.AtomicBool
	callbackMu sync.Mutex
}

type dataChannelFactory struct {
	onOpen func(dc *webrtc.DataChannel)
	mu     sync.Mutex
}

func (f *dataChannelFactory) OnOpen(onOpen func(dc *webrtc.DataChannel)) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.onOpen = onOpen
}

func (f *dataChannelFactory) Open(dc *webrtc.DataChannel) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.onOpen != nil {
		f.onOpen(dc)
	}
}

func NewWebRTCConn(cid string, pc *webrtc.PeerConnection) (*WebRTCConn, error) {
	conn := &WebRTCConn{
		cid:       cid,
		pc:        pc,
		connected: abool.New(),
	}

	// WebRTCConn will be ready when both ICE and DataChannel are ready
	var wg sync.WaitGroup
	wg.Add(2)
	dcf := &dataChannelFactory{}
	dcf.OnOpen(func(dc *webrtc.DataChannel) {
		wg.Done()
		conn.dcMu.Lock()
		conn.dc = dc
		conn.dcMu.Unlock()
		dc.OnMessage(func(msg webrtc.DataChannelMessage) {
			conn.callbackMu.Lock()
			h := conn.onMessage
			conn.callbackMu.Unlock()
			if h != nil {
				h(msg.Data)
			}
		})
		wg.Wait()
		conn.connected.Set()
		conn.callbackMu.Lock()
		h := conn.onConnect
		conn.callbackMu.Unlock()
		if h != nil {
			h()
		}
	})

	dcOffer, err := pc.CreateDataChannel("data", nil)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create data channel")
	}
	pc.OnICEConnectionStateChange(func(state webrtc.ICEConnectionState) {
		log.Printf("[%s] state has changed: %s", conn.cid, state)
		switch state {
		case webrtc.ICEConnectionStateChecking:
			weOffer := pc.RemoteDescription().Type == webrtc.SDPTypeAnswer
			if weOffer {
				dcf.Open(dcOffer)
			} else {
				pc.OnDataChannel(func(dc *webrtc.DataChannel) {
					dc.OnOpen(func() {
						dcf.Open(dc)
					})
				})
			}
		case webrtc.ICEConnectionStateConnected:
			wg.Done()
		}
	})
	return conn, nil
}

func (c *WebRTCConn) PeerConnection() *webrtc.PeerConnection {
	return c.pc
}

func (c *WebRTCConn) ConnectionID() string {
	return c.cid
}

func (c *WebRTCConn) OnConnect(f func()) {
	c.callbackMu.Lock()
	defer c.callbackMu.Unlock()
	c.onConnect = f
}

func (c *WebRTCConn) SendMessage(ctx context.Context, data []byte) error {
	if c.connected.IsNotSet() {
		return errors.New("not connected")
	}
	c.dcMu.Lock()
	defer c.dcMu.Unlock()
	return c.dc.Send(data)
}

func (c *WebRTCConn) OnMessage(f func(data []byte)) {
	c.callbackMu.Lock()
	defer c.callbackMu.Unlock()
	c.onMessage = f
}

type WebRTCStreamerConn struct {
	*WebRTCConn
	videoTrack *webrtc.TrackLocalStaticSample
	audioTrack *webrtc.TrackLocalStaticSample
}

func NewWebRTCStreamerConn(wc webrtc.Configuration) (*WebRTCStreamerConn, error) {
	pc, err := webrtc.NewPeerConnection(wc)
	if err != nil {
		return nil, err
	}
	videoTrack, err := newVideoTrack()
	if err != nil {
		return nil, err
	}
	audioTrack, err := newAudioTrack()
	if err != nil {
		return nil, err
	}
	if _, err := pc.AddTrack(videoTrack); err != nil {
		return nil, err
	}
	if _, err := pc.AddTrack(audioTrack); err != nil {
		return nil, err
	}
	conn, err := NewWebRTCConn("streamer", pc)
	if err != nil {
		return nil, err
	}
	return &WebRTCStreamerConn{
		WebRTCConn: conn,
		videoTrack: videoTrack,
		audioTrack: audioTrack,
	}, nil
}

func (c *WebRTCStreamerConn) SendVideoSample(ctx context.Context, sample MediaSample) error {
	return c.videoTrack.WriteSample(media.Sample{Data: sample.Data, Duration: sample.Duration})
}

func (c *WebRTCStreamerConn) SendAudioSample(ctx context.Context, sample MediaSample) error {
	return c.audioTrack.WriteSample(media.Sample{Data: sample.Data, Duration: sample.Duration})
}

type WebRTCPlayerConn struct {
	*WebRTCConn
}

func NewWebRTCPlayerConn(wc webrtc.Configuration) (*WebRTCPlayerConn, error) {
	pc, err := webrtc.NewPeerConnection(wc)
	if err != nil {
		return nil, err
	}
	if _, err := pc.AddTransceiverFromKind(webrtc.RTPCodecTypeVideo, webrtc.RTPTransceiverInit{Direction: webrtc.RTPTransceiverDirectionRecvonly}); err != nil {
		return nil, err
	}
	if _, err := pc.AddTransceiverFromKind(webrtc.RTPCodecTypeAudio, webrtc.RTPTransceiverInit{Direction: webrtc.RTPTransceiverDirectionRecvonly}); err != nil {
		return nil, err
	}
	conn, err := NewWebRTCConn("player", pc)
	if err != nil {
		return nil, err
	}
	return &WebRTCPlayerConn{WebRTCConn: conn}, nil
}
