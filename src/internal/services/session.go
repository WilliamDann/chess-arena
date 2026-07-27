package services

import (
	"encoding/json"
	"net/http"

	"github.com/williamdann/chess-arena/internal/model"
	"github.com/williamdann/chess-arena/internal/postgres"
	"golang.org/x/crypto/bcrypt"
)

type SessionService struct {
	users    *postgres.UserStore
	sessions *postgres.SessionStore
}

func NewSessionService(users *postgres.UserStore, sessions *postgres.SessionStore) *SessionService {
	return &SessionService{users, sessions}
}

func (s *SessionService) Register(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/session", s.createSession)
}

func (s *SessionService) createSession(w http.ResponseWriter, r *http.Request) {
	// parse request data
	var requestData model.CreateSession
	err := json.NewDecoder(r.Body).Decode(&requestData)
	if err != nil {
		http.Error(w, "unable to parse json request body", http.StatusBadRequest)
		return
	}

	// validate request data
	err = requestData.Validate()
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// get associated user
	userData, err := s.users.GetUserByEmail(r.Context(), requestData.Email)
	if err != nil {
		http.Error(w, "invalid credentials", http.StatusUnauthorized)
		return
	}

	// check credentials
	err = bcrypt.CompareHashAndPassword([]byte(userData.PasswordHash), []byte(requestData.Password))
	if err != nil {
		http.Error(w, "invalid credentials", http.StatusUnauthorized)
		return
	}

	// create session for user
	data, err := s.sessions.CreateSession(r.Context(), userData.Id)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	// OK
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
}
