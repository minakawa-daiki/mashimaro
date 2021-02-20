package broker

import (
	"context"
	"encoding/json"
	"log"
	"net/http"

	"github.com/castaneai/mashimaro/pkg/allocator"

	"github.com/castaneai/mashimaro/pkg/gamemetadata"
	"github.com/castaneai/mashimaro/pkg/gamesession"

	"github.com/go-chi/chi"
)

type ExternalServer struct {
	sessionStore  gamesession.Store
	metadataStore gamemetadata.Store
	allocator     allocator.Allocator
}

func NewExternalServer(sessionStore gamesession.Store, metadataStore gamemetadata.Store, alloc allocator.Allocator) *ExternalServer {
	return &ExternalServer{sessionStore: sessionStore, metadataStore: metadataStore, allocator: alloc}
}

type newGameResponse struct {
	SessionID gamesession.SessionID `json:"sessionId"`
}

func (s *ExternalServer) newGame(ctx context.Context, gameID string) (*gamesession.Session, error) {
	metadata, err := s.metadataStore.GetGameMetadata(ctx, gameID)
	if err != nil {
		return nil, err
	}
	allocatedServer, err := s.allocator.Allocate(ctx)
	if err != nil {
		return nil, err
	}
	ss, err := s.sessionStore.NewSession(ctx, &gamesession.NewSessionRequest{
		GameID:            gameID,
		AllocatedServerID: allocatedServer.ID,
	})
	if err != nil {
		return nil, err
	}
	log.Printf("created game session: %s (gs: %+v, metadata: %+v)", ss.SessionID, allocatedServer, metadata)
	return ss, nil
}

func (s *ExternalServer) Handler() http.Handler {
	r := chi.NewRouter()
	r.Post("/newgame/{gameID}", func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Access-Control-Allow-Headers", "*")
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("content-type", "application/json")

		gameID := chi.URLParam(req, "gameID")
		if gameID == "" {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(`{"error": "gameID is empty"}`))
			return
		}
		ss, err := s.newGame(req.Context(), gameID)
		if err == gamemetadata.ErrMetadataNotFound {
			log.Printf("metadata not found: %+v", err)
			w.WriteHeader(http.StatusNotFound)
			w.Write([]byte(`{"error": "metadata not found"}`))
			return
		}
		if err != nil {
			log.Printf("failed to new game: %+v", err)
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(`{"error": "internal server error"}`))
			return
		}
		enc := json.NewEncoder(w)
		if err := enc.Encode(&newGameResponse{SessionID: ss.SessionID}); err != nil {
			log.Printf("failed to encode JSON: %+v", err)
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(`{"error": "internal server error"}`))
			return
		}
	})
	return r
}
