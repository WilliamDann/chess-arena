package services

import (
	"encoding/json"
	"net/http"

	"github.com/williamdann/chess-arena/internal/model"
	"github.com/williamdann/chess-arena/internal/postgres"
)

type PresenceService struct {
	presence *postgres.PresenceStore
	auth     func(http.Handler) http.Handler
}

func NewPresenceService(presence *postgres.PresenceStore, auth func(http.Handler) http.Handler) *PresenceService {
	return &PresenceService{presence: presence, auth: auth}
}

func (s *PresenceService) Register(mux *http.ServeMux) {
	mux.Handle("GET /api/players/active", s.auth(http.HandlerFunc(s.getActive)))
}

func (s *PresenceService) getActive(w http.ResponseWriter, r *http.Request) {
	data, err := s.presence.GetActiveProfiles(r.Context())
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	if data == nil {
		data = []model.Profile{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
}
