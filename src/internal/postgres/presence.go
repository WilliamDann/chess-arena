package postgres

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/williamdann/chess-arena/internal/model"
)

// StaleAfterSecs is how old a connection row may grow before it counts as
// dead everywhere: readers skip such rows, reapers delete them, and ws
// servers must heartbeat their rows well inside this window.
const StaleAfterSecs = 90

type PresenceStore struct {
	db DB
}

func NewPresenceStore(db DB) *PresenceStore {
	return &PresenceStore{db: db}
}

func (s *PresenceStore) Create(ctx context.Context, user, server uuid.UUID) (model.Connection, error) {
	rows, err := s.db.Query(
		ctx,
		"insert into connections (user_id, server_id) values ($1, $2) returning *",
		user,
		server,
	)

	if err != nil {
		return model.Connection{}, err
	}

	data, err := pgx.CollectOneRow(rows, pgx.RowToStructByName[model.Connection])
	if err != nil {
		return model.Connection{}, err
	}

	return data, nil
}

func (s *PresenceStore) GetAll(ctx context.Context) ([]model.Connection, error) {
	rows, err := s.db.Query(ctx, "select * from connections")
	if err != nil {
		return []model.Connection{}, err
	}

	data, err := pgx.CollectRows(rows, pgx.RowToStructByName[model.Connection])
	if err != nil {
		return []model.Connection{}, err
	}

	return data, nil
}

func (s *PresenceStore) GetForUser(ctx context.Context, user uuid.UUID) ([]model.Connection, error) {
	rows, err := s.db.Query(
		ctx,
		"select * from connections where user_id = $1",
		user,
	)

	if err != nil {
		return []model.Connection{}, err
	}

	data, err := pgx.CollectRows(rows, pgx.RowToStructByName[model.Connection])
	if err != nil {
		return []model.Connection{}, err
	}

	return data, nil
}

// GetActiveProfiles lists the profile of every user holding a live
// connection on any server.
func (s *PresenceStore) GetActiveProfiles(ctx context.Context) ([]model.Profile, error) {
	rows, err := s.db.Query(
		ctx,
		`select distinct p.id, p.display_name
		 from connections c
		 join profiles p on p.id = c.user_id
		 where c.last_seen > now() - make_interval(secs => $1)
		 order by p.display_name`,
		StaleAfterSecs,
	)

	if err != nil {
		return []model.Profile{}, err
	}

	data, err := pgx.CollectRows(rows, pgx.RowToStructByName[model.Profile])
	if err != nil {
		return []model.Profile{}, err
	}

	return data, nil
}

func (s *PresenceStore) Heartbeat(ctx context.Context, server uuid.UUID) error {
	_, err := s.db.Exec(ctx, "update connections set last_seen = now() where server_id = $1", server)
	return err
}

func (s *PresenceStore) Delete(ctx context.Context, id uuid.UUID) (model.Connection, error) {
	rows, err := s.db.Query(ctx, "delete from connections where id = $1 returning *", id)
	if err != nil {
		return model.Connection{}, err
	}

	data, err := pgx.CollectOneRow(rows, pgx.RowToStructByName[model.Connection])
	if err != nil {
		return model.Connection{}, err
	}

	return data, nil
}

func (s *PresenceStore) DeleteAllStale(ctx context.Context, maxAgeSecs int) error {
	_, err := s.db.Exec(ctx, "delete from connections where last_seen < now() - make_interval(secs => $1)", maxAgeSecs)
	return err
}
