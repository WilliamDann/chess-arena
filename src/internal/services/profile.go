package services

import (
	"encoding/json"
	"net/http"

	"github.com/google/uuid"
	"github.com/williamdann/chess-arena/internal/postgres"
)

type ProfileService struct {
	profiles *postgres.ProfileStore
	auth     func(http.Handler) http.Handler
}

func NewProfileService(profiles *postgres.ProfileStore, auth func(http.Handler) http.Handler) *ProfileService {
	return &ProfileService{profiles: profiles, auth: auth}
}

func (s *ProfileService) Register(mux *http.ServeMux) {
	mux.Handle("GET /api/profile/my", s.auth(http.HandlerFunc(s.getMyProfile)))
	mux.HandleFunc("GET /api/profile/{id}", s.getProfile)
}

func (s *ProfileService) getProfile(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		http.Error(w, "invalid profile id", http.StatusBadRequest)
		return
	}

	data, err := s.profiles.GetProfile(r.Context(), id)
	if err != nil {
		http.Error(w, "profile not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
}

func (s *ProfileService) getMyProfile(w http.ResponseWriter, r *http.Request) {
	session, ok := SessionFromContext(r.Context())
	if !ok {
		http.Error(w, "invalid session", http.StatusUnauthorized)
		return
	}

	// set id to my id, and call get profile
	r.SetPathValue("id", session.UserId.String())
	s.getProfile(w, r)
}
