package services

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/williamdann/chess-arena/internal/model"
	"github.com/williamdann/chess-arena/internal/postgres"
	"golang.org/x/crypto/bcrypt"
)

type ctxKey int

const sessionCtxKey ctxKey = 0

// get the current session for the given request
func SessionFromContext(ctx context.Context) (model.Session, bool) {
	session, ok := ctx.Value(sessionCtxKey).(model.Session)
	return session, ok
}

type SessionService struct {
	users    *postgres.UserStore
	sessions *postgres.SessionStore
}

func NewSessionService(users *postgres.UserStore, sessions *postgres.SessionStore) *SessionService {
	return &SessionService{users, sessions}
}

func (s *SessionService) Register(mux *http.ServeMux) {
	mux.Handle("GET /api/session/my", s.RequireAuth(http.HandlerFunc(s.getMySessions)))

	mux.HandleFunc("POST /api/session", s.createSession)

	mux.Handle("DELETE /api/session/my", s.RequireAuth(http.HandlerFunc(s.signOut)))
	mux.Handle("DELETE /api/session/my/all", s.RequireAuth(http.HandlerFunc(s.signOutAll)))
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

	// set session cookie
	http.SetCookie(w, &http.Cookie{
		Name:     "chess-arena-session",
		Value:    data.Id.String(),
		Path:     "/",
		Expires:  data.ExpiresAt,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   false, // TODO when using https, set to true
	})

	// OK
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
}

func (s *SessionService) getMySessions(w http.ResponseWriter, r *http.Request) {
	session, ok := SessionFromContext(r.Context())
	if !ok {
		http.Error(w, "invalid session", http.StatusUnauthorized)
		return
	}

	sessions, err := s.sessions.GetSessionsForUser(r.Context(), session.UserId, session.Id)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(sessions)
}

func (s *SessionService) signOut(w http.ResponseWriter, r *http.Request) {
	session, ok := SessionFromContext(r.Context())
	if !ok {
		http.Error(w, "invalid session", http.StatusUnauthorized)
		return
	}
	err := s.sessions.DeleteSession(r.Context(), session.Id)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	// unset cookie
	http.SetCookie(w, &http.Cookie{
		Name:     "chess-arena-session",
		Value:    "",
		Path:     "/",
		Expires:  time.Now(),
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   false, // TODO when using https, set to true
	})
	w.WriteHeader(http.StatusOK)
}

func (s *SessionService) signOutAll(w http.ResponseWriter, r *http.Request) {
	session, ok := SessionFromContext(r.Context())
	if !ok {
		http.Error(w, "invalid session", http.StatusUnauthorized)
		return
	}

	err := s.sessions.DeleteAllForUser(r.Context(), session.UserId)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	// unset cookie
	http.SetCookie(w, &http.Cookie{
		Name:     "chess-arena-session",
		Value:    "",
		Path:     "/",
		Expires:  time.Now(),
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   false, // TODO when using https, set to true
	})
	w.WriteHeader(http.StatusOK)
}

// middleware function for checking auth
func (s *SessionService) RequireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// parse cookie
		cookie, err := r.Cookie("chess-arena-session")
		if err != nil {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		id, err := uuid.Parse(cookie.Value)
		if err != nil {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		// check session exists
		session, err := s.sessions.GetSessionById(r.Context(), id)
		if err != nil {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		// if session is expiring soon, refresh
		if time.Until(session.ExpiresAt) < 24*time.Hour {
			refreshed, err := s.sessions.RefreshSession(r.Context(), session.Id)
			if err != nil {
				slog.Error("failed token refresh", "err", err)
			} else {
				http.SetCookie(w, &http.Cookie{
					Name:     "chess-arena-session",
					Value:    refreshed.Id.String(),
					Path:     "/",
					Expires:  refreshed.ExpiresAt,
					HttpOnly: true,
					SameSite: http.SameSiteLaxMode,
					Secure:   false, // TODO when using https, set to true
				})
				session = refreshed
			}
		}

		// provide session as context
		ctx := context.WithValue(r.Context(), sessionCtxKey, session)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
