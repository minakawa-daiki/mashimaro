package signaling

import (
	"context"
	"errors"
	"io"
	"log"

	"github.com/castaneai/mashimaro/pkg/gamesession"
	"github.com/castaneai/mashimaro/pkg/proto"
)

type internalServer struct {
	sessionStore gamesession.Store
	channels     *Channels
}

func NewInternalServer(store gamesession.Store, channels *Channels) proto.SignalingServer {
	return &internalServer{
		sessionStore: store,
		channels:     channels,
	}
}

func (s *internalServer) ReceiveSignalingOffer(ctx context.Context, req *proto.ReceiveSignalingOfferRequest) (*proto.ReceiveSignalingOfferResponse, error) {
	sid := gamesession.SessionID(req.SessionId)
	_, err := s.sessionStore.GetSession(ctx, sid)
	if errors.Is(err, gamesession.ErrSessionNotFound) {
		return &proto.ReceiveSignalingOfferResponse{Found: false}, nil
	}
	if err != nil {
		return nil, err
	}
	select {
	case offerBody := <-s.channels.OfferCh(sid):
		return &proto.ReceiveSignalingOfferResponse{
			Found: true,
			Offer: &proto.SignalingOffer{Body: offerBody},
		}, nil
	default:
		return &proto.ReceiveSignalingOfferResponse{Found: false}, nil
	}
}

func (s *internalServer) SendSignalingAnswer(ctx context.Context, req *proto.SendSignalingAnswerRequest) (*proto.SendSignalingAnswerResponse, error) {
	sid := gamesession.SessionID(req.SessionId)
	_, err := s.sessionStore.GetSession(ctx, sid)
	if err != nil {
		return nil, err
	}
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case s.channels.AnswerCh(sid) <- req.SignalingAnswer.Body:
		delete(s.channels.answerChs, sid)
	}
	return &proto.SendSignalingAnswerResponse{}, nil
}

func (s *internalServer) TrickleSignaling(stream proto.Signaling_TrickleSignalingServer) error {
	req, err := stream.Recv()
	if err != nil {
		return err
	}
	sid := gamesession.SessionID(req.SessionId)
	select {
	case <-stream.Context().Done():
		return stream.Context().Err()
	case s.channels.AnswerCandidateCh(sid) <- req.Candidate.Body:
	}

	// transfer ICE candidates from offerPC to answerPC (WebSocket -> gRPC)
	go func() {
		for {
			select {
			case <-stream.Context().Done():
				return
			case body := <-s.channels.OfferCandidateCh(sid):
				if err := stream.Send(&proto.TrickleSignalingResponse{
					SessionId: string(sid),
					Candidate: &proto.ICECandidate{Body: body},
				}); err != nil {
					log.Printf("failed to send offer candidate: %+v", err)
				}
			}
		}
	}()

	// transfer ICE candidates from answerPC to offerPC (gRPC -> WebSocket)
	for {
		req, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		sid := gamesession.SessionID(req.SessionId)
		select {
		case <-stream.Context().Done():
			break
		case s.channels.AnswerCandidateCh(sid) <- req.Candidate.Body:
		}
	}
	return nil
}
