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

func (s *SessionStore) RefreshSession(ctx context.Context, sessionId uuid.UUID) (model.Session, error) {
	rows, err := s.db.Query(ctx, "update sessions set expires_at = now() + interval '7 days' where id =$1 returning *", sessionId)
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

func (s *SessionStore) GetSessionsForUser(ctx context.Context, userId uuid.UUID) ([]model.Session, error) {
	rows, err := s.db.Query(ctx, "select * from sessions where user_id = $1 and expires_at > now()", userId)
	if err != nil {
		return []model.Session{}, errors.New("session not found")
	}

	data, err := pgx.CollectRows(rows, pgx.RowToStructByName[model.Session])
	if err != nil {
		return []model.Session{}, err
	}

	return data, nil
}

func (s *SessionStore) DeleteSession(ctx context.Context, id uuid.UUID) error {
	_, err := s.db.Exec(ctx, "delete from sessions where id = $1", id)
	return err
}

func (s *SessionStore) DeleteAllForUser(ctx context.Context, userId uuid.UUID) error {
	_, err := s.db.Exec(ctx, "delete from sessions where user_id = $1", userId)
	return err
}
