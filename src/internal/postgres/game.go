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
		"insert into games (white_player, black_player, clock_initial_ms, clock_increment_ms, clock_white_ms, clock_black_ms) values ($1, $2, $3, $4, $3, $3) returning *",
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

func (s *GameStore) GetActiveForPlayer(ctx context.Context, player uuid.UUID) (model.Game, error) {
	rows, err := s.db.Query(ctx, "select * from games where (white_player = $1 or black_player = $1) and result = '*' order by created_at desc", player)
	if err != nil {
		return model.Game{}, err
	}

	game, err := pgx.CollectOneRow(rows, pgx.RowToStructByName[model.Game])
	if err != nil {
		return model.Game{}, err
	}

	return game, nil
}
