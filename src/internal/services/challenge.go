package services

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/google/uuid"
	"github.com/williamdann/chess-arena/internal/model"
	"github.com/williamdann/chess-arena/internal/postgres"
)

type ChallengeService struct {
	challenges *postgres.ChallengeStore
	auth       func(http.Handler) http.Handler
}

func NewChallengeService(store *postgres.ChallengeStore, auth func(http.Handler) http.Handler) *ChallengeService {
	return &ChallengeService{challenges: store, auth: auth}
}

func (s *ChallengeService) Register(mux *http.ServeMux) {
	mux.Handle("GET /api/challenges/open", s.auth(http.HandlerFunc(s.getOpen)))

	mux.Handle("GET /api/challenges/out", s.auth(http.HandlerFunc(s.getOutgoing)))
	mux.Handle("DELETE /api/challenges/out", s.auth(http.HandlerFunc(s.deleteOutgoing)))

	mux.Handle("GET /api/challenges/in", s.auth(http.HandlerFunc(s.getIncoming)))
	mux.Handle("DELETE /api/challenges/in", s.auth(http.HandlerFunc(s.deleteIncoming)))

	mux.Handle("GET /api/challenge/{id}", s.auth(http.HandlerFunc(s.getItem)))
	mux.Handle("POST /api/challenge", s.auth(http.HandlerFunc(s.create)))
	mux.Handle("DELETE /api/challenge/{id}", s.auth(http.HandlerFunc(s.deleteItem)))
}

func (s *ChallengeService) getItem(w http.ResponseWriter, r *http.Request) {
	stringId := r.PathValue("id")
	id, err := uuid.Parse(stringId)
	if err != nil {
		http.Error(w, "invalid item id", http.StatusBadRequest)
		return
	}

	item, err := s.challenges.GetById(r.Context(), id)
	if err != nil {
		http.Error(w, "unable to find challenge", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(item)
}

func (s *ChallengeService) getOpen(w http.ResponseWriter, r *http.Request) {
	data, err := s.challenges.GetOpen(r.Context())
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
}

func (s *ChallengeService) getIncoming(w http.ResponseWriter, r *http.Request) {
	me, ok := SessionFromContext(r.Context())
	if !ok {
		http.Error(w, "invalid session", http.StatusUnauthorized)
		return
	}

	data, err := s.challenges.GetToUser(r.Context(), me.UserId)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
}

func (s *ChallengeService) getOutgoing(w http.ResponseWriter, r *http.Request) {
	me, ok := SessionFromContext(r.Context())
	if !ok {
		http.Error(w, "invalid session", http.StatusUnauthorized)
		return
	}

	data, err := s.challenges.GetFromUser(r.Context(), me.UserId)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
}

func (s *ChallengeService) create(w http.ResponseWriter, r *http.Request) {
	// get current user
	me, ok := SessionFromContext(r.Context())
	if !ok {
		http.Error(w, "invalid session", http.StatusUnauthorized)
		return
	}

	// parse challenge
	var request model.CreateChallenge
	err := json.NewDecoder(r.Body).Decode(&request)
	if err != nil {
		http.Error(w, "error parsing request body", http.StatusBadRequest)
		return
	}

	if request.ClockInitial <= 0 || request.ClockIncrement < 0 {
		http.Error(w, "invalid clock settings", http.StatusBadRequest)
		return
	}
	if request.FromPlayer != uuid.Nil && request.FromPlayer != me.UserId {
		http.Error(w, "you can only create challenges as yourself", http.StatusUnauthorized)
		return
	}
	if request.ToPlayer != nil && *request.ToPlayer == me.UserId {
		http.Error(w, "you cannot play yourself", http.StatusUnauthorized)
		return
	}

	// set FromPlayer to session user
	request.FromPlayer = me.UserId

	// create object
	// TODO emit pubsub
	data, err := s.challenges.Create(r.Context(), request)
	if errors.Is(err, postgres.ErrChallengeLimit) {
		http.Error(w, "too many open challenges", http.StatusConflict)
		return
	}
	if err != nil {
		slog.Error("challenge insert failed", "err", err)
		http.Error(w, "failed to create challenge", http.StatusBadRequest)
		return
	}

	// return result
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
}

func (s *ChallengeService) deleteItem(w http.ResponseWriter, r *http.Request) {
	me, ok := SessionFromContext(r.Context())
	if !ok {
		http.Error(w, "invalid session", http.StatusUnauthorized)
		return
	}

	idString := r.PathValue("id")
	id, err := uuid.Parse(idString)
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}

	err = s.challenges.DeleteForUser(r.Context(), id, me.UserId)
	if err != nil {
		http.Error(w, "failed to delete", http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (s *ChallengeService) deleteIncoming(w http.ResponseWriter, r *http.Request) {
	me, ok := SessionFromContext(r.Context())
	if !ok {
		http.Error(w, "invalid session", http.StatusUnauthorized)
		return
	}

	err := s.challenges.DeleteIncoming(r.Context(), me.UserId)
	if err != nil {
		http.Error(w, "failed to delete", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (s *ChallengeService) deleteOutgoing(w http.ResponseWriter, r *http.Request) {
	me, ok := SessionFromContext(r.Context())
	if !ok {
		http.Error(w, "invalid session", http.StatusUnauthorized)
		return
	}

	err := s.challenges.DeleteOutgoing(r.Context(), me.UserId)
	if err != nil {
		http.Error(w, "failed to delete", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}
