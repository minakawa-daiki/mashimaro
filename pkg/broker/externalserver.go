package broker

import (
	"encoding/json"
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
		gameID := chi.URLParam(req, "gameID")
		if gameID == "" {
			http.Error(w, "gameID is empty", http.StatusUnprocessableEntity)
			return
		}
		ss, err := b.NewGame(req.Context(), gameID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("content-type", "application/json")
		enc := json.NewEncoder(w)
		if err := enc.Encode(&NewGameResponse{SessionID: ss.SessionID}); err != nil {
			http.Error(w, "failed to marshal json", http.StatusInternalServerError)
			return
		}
	})
	return r
}
