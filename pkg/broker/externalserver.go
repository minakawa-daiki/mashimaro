package broker

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/castaneai/mashimaro/pkg/gamesession"

	"github.com/go-chi/chi"
)

type NewGameResponse struct {
	SessionID gamesession.SessionID `json:"sessionId"`
}

func ExternalServer(b *Broker) http.Handler {
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
		ss, err := b.NewGame(req.Context(), gameID)
		if err != nil {
			log.Printf("failed to new game: %+v", err)
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(`{"error": "internal server error"}`))
			return
		}
		enc := json.NewEncoder(w)
		if err := enc.Encode(&NewGameResponse{SessionID: ss.SessionID}); err != nil {
			log.Printf("failed to encode JSON: %+v", err)
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(`{"error": "internal server error"}`))
			return
		}
	})
	return r
}
