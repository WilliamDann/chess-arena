package postgres

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/williamdann/chess-arena/internal/model"
)

type SessionStore struct {
	db *pgxpool.Pool
}

func NewSessionStore(db *pgxpool.Pool) *SessionStore {
	return &SessionStore{db: db}
}

func (s *SessionStore) CreateSession(ctx context.Context, userID uuid.UUID) (model.Session, error) {
	rows, err := s.db.Query(ctx, "insert into sessions (user_id) values($1) returning *", userID)
	if err != nil {
		return model.Session{}, err
	}

	data, err := pgx.CollectOneRow(rows, pgx.RowToStructByName[model.Session])
	if err != nil {
		return model.Session{}, err
	}

	return data, nil
}

func (s *SessionStore) GetSessionById(ctx context.Context, id uuid.UUID) (model.Session, error) {
	rows, err := s.db.Query(ctx, "select * from sessions where id = $1 and expires_at > now()", id)
	if err != nil {
		return model.Session{}, errors.New("session not found")
	}

	data, err := pgx.CollectOneRow(rows, pgx.RowToStructByName[model.Session])
	if err != nil {
		return model.Session{}, err
	}

	return data, nil
}
