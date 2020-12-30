package signaling

import (
	"context"
	"io"
	"log"

	"github.com/castaneai/mashimaro/pkg/internal/webrtcutil"

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

type message struct {
	Operation string                `json:"operation"`
	SessionID gamesession.SessionID `json:"session_id"`
	Body      string                `json:"body"`
}

func (s *Server) WebSocketHandler() websocket.Handler {
	return func(ws *websocket.Conn) {
		sessionCtx, cancel := context.WithCancel(s.ctx)
		defer cancel()
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
				s.handleRequest(sessionCtx, ws, &msg)
			}
		}
	}
}

func (s *Server) handleRequest(ctx context.Context, ws *websocket.Conn, msg *message) {
	switch msg.Operation {
	case OperationNewGame:
		ss, err := s.gsManager.NewSession(ctx, "test-game") // TODO: gameID request body
		if err != nil {
			log.Printf("failed to new game: %+v", err)
			return
		}
		s.trickleManager.NewSession(ctx, ss, func(candidate *webrtc.ICECandidateInit) {
			body, err := webrtcutil.EncodeICECandidate(candidate)
			if err != nil {
				log.Printf("failed to encode ICE candidate: %+v", err)
				return
			}
			if err := websocket.JSON.Send(ws, &message{Operation: OperationICECandidate, Body: body}); err != nil {
				log.Printf("failed to send ice candidate from pcAnswer to pcOffer: %+v", err)
				return
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
		answer, err := ss.RPCClient.FirstSignaling(ctx, &proto.Offer{Body: msg.Body})
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
		candidate, err := webrtcutil.DecodeICECandidate(msg.Body)
		if err != nil {
			log.Printf("failed to decode ICE candidate: %+v", err)
			return
		}
		if err := s.trickleManager.AddICECandidate(ss.SessionID, candidate); err != nil {
			log.Printf("failed to add ice candidate: %+v", err)
			return
		}

	default:
		log.Printf("unknown operation received: %s", msg.Operation)
	}
}
