package signaling

import (
	"context"
	"errors"
	"log"

	"github.com/castaneai/mashimaro/pkg/gamesession"

	"golang.org/x/net/websocket"
)

const (
	OperationOffer        = "offer"
	OperationAnswer       = "answer"
	OperationICECandidate = "ice_candidate"
)

type ExternalServer struct {
	sessionStore gamesession.Store
	channels     *Channels
}

func NewExternalServer(sessionStore gamesession.Store, channels *Channels) *ExternalServer {
	return &ExternalServer{
		sessionStore: sessionStore,
		channels:     channels,
	}
}

type WSMessage struct {
	Operation string                `json:"operation"`
	SessionID gamesession.SessionID `json:"session_id"`
	Body      string                `json:"body"`
}

func (s *ExternalServer) WebSocketHandler() websocket.Handler {
	return func(ws *websocket.Conn) {
		sessionCtx, cancel := context.WithCancel(context.Background())
		defer cancel()

		for {
			var msg WSMessage
			if err := websocket.JSON.Receive(ws, &msg); err != nil {
				log.Printf("failed to receive json: %+v", err)
				return
			}
			s.handleRequest(sessionCtx, ws, &msg)
		}
	}
}

func (s *ExternalServer) handleRequest(ctx context.Context, ws *websocket.Conn, msg *WSMessage) {
	sid := msg.SessionID
	switch msg.Operation {
	case OperationOffer:
		_, err := s.sessionStore.GetSession(ctx, sid)
		if errors.Is(err, gamesession.ErrSessionNotFound) {
			log.Printf("session not found: %s", msg.SessionID)
			return
		}
		if err != nil {
			log.Printf("failed to get session: %+v", err)
			return
		}
		// TODO: state check
		/*
			if ss.State != gamesession.StateWaitingForSession && ss.State != gamesession.StateSignaling {
				log.Printf("cannot create duplicate offer")
				return
			}
		*/
		select {
		case <-ctx.Done():
			log.Printf("sending offer timed out")
			return
		case s.channels.OfferCh(sid) <- msg.Body:
		}
		select {
		case <-ctx.Done():
			log.Printf("waiting answer timed out")
			return
		case answer := <-s.channels.AnswerCh(sid):
			msg := &WSMessage{Operation: OperationAnswer, Body: answer}
			if err := websocket.JSON.Send(ws, msg); err != nil {
				log.Printf("failed to send via websocket: %+v", err)
				return
			}
		}
		go func(sid gamesession.SessionID) {
			for {
				select {
				case <-ctx.Done():
					return
				case answer := <-s.channels.AnswerCandidateCh(sid):
					if err := websocket.JSON.Send(ws, &WSMessage{
						Operation: OperationICECandidate,
						Body:      answer,
					}); err != nil {
						log.Printf("failed to send via websocket: %+v", err)
					}
					if answer == "" {
						log.Printf("finished gathering answer ICE candidates; stopped goroutine(signaling external answer -> offer)")
						return
					}
				}
			}
		}(sid)
	case OperationICECandidate:
		_, err := s.sessionStore.GetSession(ctx, msg.SessionID)
		if errors.Is(err, gamesession.ErrSessionNotFound) {
			log.Printf("session not found: %s", msg.SessionID)
			return
		}
		if err != nil {
			log.Printf("failed to get session: %+v", err)
			return
		}
		select {
		case <-ctx.Done():
			log.Printf("sending offer candidate canceled")
		case s.channels.OfferCandidateCh(msg.SessionID) <- msg.Body:
		}

	default:
		log.Printf("unknown operation received: %s", msg.Operation)
	}
}
