package signaling

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"io"
	"log"
	"net/http"

	"github.com/castaneai/mashimaro/pkg/proto"

	"github.com/castaneai/mashimaro/pkg/gamesession"

	"golang.org/x/net/websocket"

	"github.com/pion/webrtc/v3"
)

const (
	OperationNewGame      = "new_game"
	OperationOffer        = "offer"
	OperationAnswer       = "answer"
	OperationICECandidate = "ice_candidate"
)

type Server struct {
	ctx            context.Context
	cancel         context.CancelFunc
	gsManager      *gamesession.Manager
	trickleManager *trickleManager
}

func NewServer(gsManager *gamesession.Manager) *Server {
	ctx, cancel := context.WithCancel(context.Background())
	return &Server{
		ctx:            ctx,
		cancel:         cancel,
		gsManager:      gsManager,
		trickleManager: newTrickleManager(),
	}
}

func (s *Server) ListenAndServe(addr string) error {
	return http.ListenAndServe(addr, s.webSocketServer())
}

type message struct {
	Operation string                `json:"operation"`
	SessionID gamesession.SessionID `json:"session_id"`
	Body      string                `json:"body"`
}

func decodeSDP(encoded string) (*webrtc.SessionDescription, error) {
	b, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return nil, err
	}
	var offer webrtc.SessionDescription
	if err := json.Unmarshal(b, &offer); err != nil {
		return nil, err
	}
	return &offer, nil
}

func encodeSDP(sdp *webrtc.SessionDescription) (string, error) {
	b, err := json.Marshal(sdp)
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(b), nil
}

func encodeICECandidate(candidate webrtc.ICECandidateInit) string {
	return candidate.Candidate
}

func decodeICECandidate(s string) webrtc.ICECandidateInit {
	return webrtc.ICECandidateInit{Candidate: s}
}

func (s *Server) webSocketServer() websocket.Handler {
	return func(ws *websocket.Conn) {
		for {
			select {
			case <-s.ctx.Done():
				return
			default:
				var msg message
				if err := websocket.JSON.Receive(ws, &msg); err != nil {
					if err != io.EOF {
						log.Printf("failed to receive json: %+v", err)
					}
					return
				}
				s.handleRequest(ws, &msg)
			}
		}
	}
}

func (s *Server) handleRequest(ws *websocket.Conn, msg *message) {
	switch msg.Operation {
	case OperationNewGame:
		ss, err := s.gsManager.NewSession(s.ctx, "test-game") // TODO: gameID request body
		if err != nil {
			log.Printf("failed to new game: %+v", err)
			return
		}
		s.trickleManager.NewSession(s.ctx, ss, func(candidate webrtc.ICECandidateInit) {
			if err := websocket.JSON.Send(ws, &message{Operation: OperationICECandidate, Body: encodeICECandidate(candidate)}); err != nil {
				log.Printf("failed to send ice candidate from pcAnswer to pcOffer: %+v", err)
			}
		})

		log.Printf("created new session: %+v", ss)
		if err := websocket.JSON.Send(ws, &message{Operation: OperationNewGame, Body: string(ss.SessionID)}); err != nil {
			log.Printf("failed to send session via websocket: %+v", err)
			return
		}

	case OperationOffer:
		ss, ok := s.gsManager.GetSession(msg.SessionID)
		if !ok {
			log.Printf("session not found: %s", msg.SessionID)
			return
		}
		answer, err := ss.RPCClient.FirstSignaling(s.ctx, &proto.Offer{Body: msg.Body})
		if err != nil {
			log.Printf("failed to first signaling with game server: %+v", err)
			return
		}
		msg := &message{Operation: OperationAnswer, Body: answer.Body}
		if err := websocket.JSON.Send(ws, msg); err != nil {
			log.Printf("failed to send via websocket: %+v", err)
		}

	case OperationICECandidate:
		ss, ok := s.gsManager.GetSession(msg.SessionID)
		if !ok {
			log.Printf("session not found: %s", msg.SessionID)
			return
		}
		if err := s.trickleManager.AddICECandidate(ss.SessionID, decodeICECandidate(msg.Body)); err != nil {
			log.Printf("failed to add ice candidate: %+v", err)
			return
		}

	default:
		log.Printf("unknown operation received: %s", msg.Operation)
	}
}
