package ayame

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log"
	"sync"

	"github.com/pion/webrtc/v3"

	"golang.org/x/net/websocket"
)

type Client struct {
	conn *websocket.Conn
	pc   *webrtc.PeerConnection
	rid  string
	cid  string
	opts *opts

	onConnect    func()
	onDisconnect func()
	callbackMu   sync.Mutex
}

type opts struct {
}

func defaultOptions() *opts {
	return &opts{}
}

type ClientOption interface {
	apply(opts *opts)
}

type ClientOptionFunc func(*opts)

func (f ClientOptionFunc) apply(opts *opts) {
	f(opts)
}

func NewClient(pc *webrtc.PeerConnection, options ...ClientOption) *Client {
	opts := defaultOptions()
	for _, opt := range options {
		opt.apply(opts)
	}
	return &Client{
		pc:   pc,
		opts: opts,
	}
}

type ConnectRequest struct {
	RoomID       string
	ClientID     string
	SignalingKey string
}

func (c *Client) Connect(ctx context.Context, url_ string, req *ConnectRequest) error {
	return c.signaling(ctx, url_, req)
}

func (c *Client) signaling(ctx context.Context, url_ string, req *ConnectRequest) error {
	conn, err := websocket.Dial(url_, "", url_)
	if err != nil {
		return err
	}
	c.conn = conn
	go c.recv(ctx)
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
				if errors.Is(err, io.EOF) {
					break
				}
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
		log.Printf("[%s] accepted(room: %s)", c.cid, c.rid)
		if len(acc.IceServers) > 0 {
			log.Printf("[%s] set ICE servers", c.cid)
			conf := c.pc.GetConfiguration()
			conf.ICEServers = nil
			for _, resp := range acc.IceServers {
				conf.ICEServers = append(conf.ICEServers, webrtc.ICEServer{
					URLs:       resp.URLs,
					Username:   resp.Username,
					Credential: resp.Credential,
				})
			}
			if err := c.pc.SetConfiguration(conf); err != nil {
				log.Printf("[%s] failed to set configuration: %+v", c.cid, err)
				return
			}
		}
		c.pc.OnICECandidate(func(candidate *webrtc.ICECandidate) {
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
		if acc.IsExistClient {
			if err := c.sendOffer(); err != nil {
				log.Printf("failed to send offer: %+v", err)
				return
			}
		}
	case "reject":
		var rej rejectMessage
		if err := json.Unmarshal(msg.Payload, &rej); err != nil {
			log.Printf("failed to unmarshal json: %+v", err)
			return
		}
		log.Printf("rejected (reason: %s)", rej.Reason)
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
		}
	case "bye":
		log.Printf("[%s] bye received from ayame", c.cid)
	default:
		log.Printf("unknown type received: %s", msg.Type)
	}
}

func (c *Client) sendRegisterMessage(req *ConnectRequest) error {
	msg := &registerMessage{
		Type:         "register",
		RoomID:       req.RoomID,
		ClientID:     req.ClientID,
		SignalingKey: req.SignalingKey,
	}
	c.cid = req.ClientID
	c.rid = req.RoomID
	return websocket.JSON.Send(c.conn, msg)
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
	log.Printf("[%s] offer sent", c.cid)
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
	log.Printf("[%s] answer sent", c.cid)
	return nil
}
