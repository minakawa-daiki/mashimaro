package ayame

import (
	"context"
	"encoding/json"
	"log"

	"github.com/pion/webrtc/v3"

	"golang.org/x/net/websocket"
)

type Client struct {
	conn *websocket.Conn
	pc   *webrtc.PeerConnection
	rid  string
	cid  string
}

func NewClient() *Client {
	return &Client{}
}

type ConnectRequest struct {
	RoomID   string
	ClientID string
}

func (c *Client) Connect(ctx context.Context, url_ string, req *ConnectRequest) error {
	return c.signaling(ctx, url_, req)
}

func (c *Client) signaling(ctx context.Context, url_ string, req *ConnectRequest) error {
	conn, err := websocket.Dial(url_, "", url_)
	if err != nil {
		return err
	}
	go c.recv(ctx)
	c.conn = conn
	if err := c.sendRegisterMessage(req); err != nil {
		return err
	}
	return nil
}

func (c *Client) recv(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
			var msg receivedMessage
			if err := websocket.JSON.Receive(c.conn, &msg); err != nil {
				log.Printf("failed to receive JSON: %+v", err)
				continue
			}
			c.handleMessage(&msg)
		}
	}
}

func (c *Client) handleMessage(msg *receivedMessage) {
	switch msg.Type {
	case "ping":
		pong := &pingPongMessage{Type: "pong"}
		if err := websocket.JSON.Send(c.conn, pong); err != nil {
			log.Printf("failed to send json: %+v", err)
			return
		}
	case "accept":
		var acc acceptMessage
		if err := json.Unmarshal(msg.Payload, &acc); err != nil {
			log.Printf("failed to unmarshal json: %+v", err)
			return
		}
		log.Printf("accepted! %+v", acc)
		if err := c.createPeerConnection(); err != nil {
			log.Printf("failed to create peer connection: %+v", err)
			return
		}
		if acc.IsExistClient {
			if err := c.sendOffer(); err != nil {
				log.Printf("failed to send offer: %+v", err)
				return
			}
		}
	case "offer":
		log.Printf("[%s] offer received", c.cid)
		var sdp webrtc.SessionDescription
		if err := json.Unmarshal(msg.Payload, &sdp); err != nil {
			log.Printf("failed to unmarshal json: %+v", err)
			return
		}
		if err := c.pc.SetRemoteDescription(sdp); err != nil {
			log.Printf("failed to set remote desc: %+v", err)
			return
		}
		if err := c.sendAnswer(); err != nil {
			log.Printf("failed to send answer: %+v", err)
			return
		}
	case "answer":
		log.Printf("[%s] answer received", c.cid)
		var sdp webrtc.SessionDescription
		if err := json.Unmarshal(msg.Payload, &sdp); err != nil {
			log.Printf("failed to unmarshal json: %+v", err)
			return
		}
		if err := c.pc.SetRemoteDescription(sdp); err != nil {
			log.Printf("failed to set remote desc: %+v", err)
			return
		}
	case "candidate":
		var candMsg candidateMessage
		if err := json.Unmarshal(msg.Payload, &candMsg); err != nil {
			log.Printf("failed to unmarshal json: %+v", err)
			return
		}
		if candMsg.ICECandidate != nil {
			if err := c.pc.AddICECandidate(*candMsg.ICECandidate); err != nil {
				log.Printf("failed to add ice candidate: %+v", err)
				return
			}
		} else {
			log.Printf("[%s] received nil candidate", c.cid)
		}
	default:
		log.Printf("unknown type received: %s", msg.Type)
	}
}

func (c *Client) sendRegisterMessage(req *ConnectRequest) error {
	msg := &registerMessage{
		Type:     "register",
		RoomID:   req.RoomID,
		ClientID: req.ClientID,
	}
	c.cid = req.ClientID
	c.rid = req.RoomID
	return websocket.JSON.Send(c.conn, msg)
}

func (c *Client) createPeerConnection() error {
	// TODO: STUN server
	pc, err := webrtc.NewPeerConnection(webrtc.Configuration{})
	if err != nil {
		return err
	}
	pc.OnICEConnectionStateChange(func(state webrtc.ICEConnectionState) {
		log.Printf("[%s] connection state has changed: %s", c.cid, state)
	})
	pc.OnICECandidate(func(candidate *webrtc.ICECandidate) {
		var ice *webrtc.ICECandidateInit
		if candidate != nil {
			c := candidate.ToJSON()
			ice = &c
		}
		log.Printf("[%s] new ICE candidate: %v", c.cid, candidate)
		if err := websocket.JSON.Send(c.conn, &candidateMessage{Type: "candidate", ICECandidate: ice}); err != nil {
			log.Printf("failed to send JSON: %+v", err)
		}
	})
	if _, err = pc.AddTransceiverFromKind(webrtc.RTPCodecTypeVideo, webrtc.RTPTransceiverInit{Direction: webrtc.RTPTransceiverDirectionRecvonly}); err != nil {
		return err
	}
	if _, err := pc.AddTransceiverFromKind(webrtc.RTPCodecTypeAudio, webrtc.RTPTransceiverInit{Direction: webrtc.RTPTransceiverDirectionRecvonly}); err != nil {
		return err
	}
	c.pc = pc
	return nil
}

func (c *Client) sendOffer() error {
	offer, err := c.pc.CreateOffer(nil)
	if err != nil {
		return err
	}
	if err := c.pc.SetLocalDescription(offer); err != nil {
		return err
	}
	if err := websocket.JSON.Send(c.conn, c.pc.LocalDescription()); err != nil {
		return err
	}
	return nil
}

func (c *Client) sendAnswer() error {
	answer, err := c.pc.CreateAnswer(nil)
	if err != nil {
		return err
	}
	if err := c.pc.SetLocalDescription(answer); err != nil {
		return err
	}
	if err := websocket.JSON.Send(c.conn, c.pc.LocalDescription()); err != nil {
		return err
	}
	return nil
}
