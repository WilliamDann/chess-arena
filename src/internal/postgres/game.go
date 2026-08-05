package postgres

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/williamdann/chess-arena/internal/model"
)

type GameStore struct {
	db DB
}

func NewGameStore(db DB) *GameStore {
	return &GameStore{db: db}
}

// WithTx returns a copy of the store that runs its queries inside tx.
func (s *GameStore) WithTx(tx pgx.Tx) *GameStore {
	return &GameStore{db: tx}
}

func (s *GameStore) Create(ctx context.Context, request model.CreateGame) (model.Game, error) {
	rows, err := s.db.Query(
		ctx,
		"insert into games (white_player, black_player, clock_initial_ms, clock_increment_ms) values ($1, $2, $3, $4) returning *",
		request.WhitePlayer,
		request.BlackPlayer,
		request.ClockInitial,
		request.ClockIncrement,
	)

	if err != nil {
		return model.Game{}, err
	}

	game, err := pgx.CollectOneRow(rows, pgx.RowToStructByName[model.Game])
	if err != nil {
		return model.Game{}, err
	}

	return game, nil
}

func (s *GameStore) GetById(ctx context.Context, item uuid.UUID) (model.Game, error) {
	rows, err := s.db.Query(ctx, "select * from games where id = $1", item)
	if err != nil {
		return model.Game{}, err
	}

	game, err := pgx.CollectOneRow(rows, pgx.RowToStructByName[model.Game])
	if err != nil {
		return model.Game{}, err
	}

	return game, err
}

func (s *GameStore) SetResult(ctx context.Context, id uuid.UUID, result string) (bool, error) {
	tag, err := s.db.Exec(
		ctx,
		"update games set result = $2, result_at = now() where id = $1 and result = '*'",
		id,
		result,
	)
	if err != nil {
		return false, err
	}
	return tag.RowsAffected() > 0, nil
}

// CountForPlayer reports how many games the player has, games in play included.
func (s *GameStore) CountForPlayer(ctx context.Context, player uuid.UUID) (int, error) {
	var total int
	err := s.db.QueryRow(
		ctx,
		"select count(*) from games where (white_player = $1 or black_player = $1)",
		player,
	).Scan(&total)
	return total, err
}

func (s *GameStore) GetForPlayer(ctx context.Context, player uuid.UUID, limit, offset int) (Page[model.Game], error) {
	// clamp rather than reject, so garbage input degrades to defaults
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	if offset < 0 {
		offset = 0
	}

	total, err := s.CountForPlayer(ctx, player)
	if err != nil {
		return Page[model.Game]{}, err
	}

	rows, err := s.db.Query(
		ctx,
		"select * from games where (white_player = $1 or black_player = $1) order by created_at desc, id desc limit $2 offset $3",
		player,
		limit,
		offset,
	)

	if err != nil {
		return Page[model.Game]{}, err
	}

	data, err := pgx.CollectRows(rows, pgx.RowToStructByName[model.Game])
	if err != nil {
		return Page[model.Game]{}, err
	}
	if data == nil {
		data = []model.Game{}
	}

	return Page[model.Game]{
		Items:  data,
		Limit:  limit,
		Offset: offset,
		Total:  total,
	}, nil
}

func (s *GameStore) GetActiveForPlayer(ctx context.Context, player uuid.UUID) ([]model.Game, error) {
	rows, err := s.db.Query(ctx, "select * from games where (white_player = $1 or black_player = $1) and result = '*' order by created_at desc", player)
	if err != nil {
		return []model.Game{}, err
	}

	game, err := pgx.CollectRows(rows, pgx.RowToStructByName[model.Game])
	if err != nil {
		return []model.Game{}, err
	}

	return game, nil
}
