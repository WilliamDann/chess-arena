package postgres

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/williamdann/chess-arena/internal/model"
)

// maximum outstanding challenges a single user may have at once
const maxChallengesPerUser = 5

var ErrChallengeLimit = errors.New("challenge limit reached")

type ChallengeStore struct {
	db *pgxpool.Pool
}

func NewChallengeStore(db *pgxpool.Pool) *ChallengeStore {
	return &ChallengeStore{db: db}
}

func (s *ChallengeStore) GetById(ctx context.Context, id uuid.UUID) (model.Challenge, error) {
	rows, err := s.db.Query(ctx, "select * from challenges where id = $1", id)
	if err != nil {
		return model.Challenge{}, err
	}

	data, err := pgx.CollectOneRow(rows, pgx.RowToStructByName[model.Challenge])
	if err != nil {
		return model.Challenge{}, err
	}

	return data, nil
}

func (s *ChallengeStore) GetOpen(ctx context.Context) ([]model.Challenge, error) {
	rows, err := s.db.Query(ctx, "select * from challenges where to_player is null")
	if err != nil {
		return []model.Challenge{}, err
	}

	data, err := pgx.CollectRows(rows, pgx.RowToStructByName[model.Challenge])
	if err != nil {
		return []model.Challenge{}, err
	}

	return data, nil
}

func (s *ChallengeStore) GetFromUser(ctx context.Context, userId uuid.UUID) ([]model.Challenge, error) {
	rows, err := s.db.Query(ctx, "select * from challenges where from_player = $1", userId)
	if err != nil {
		return []model.Challenge{}, err
	}

	data, err := pgx.CollectRows(rows, pgx.RowToStructByName[model.Challenge])
	if err != nil {
		return []model.Challenge{}, err
	}

	return data, nil
}

func (s *ChallengeStore) GetToUser(ctx context.Context, userId uuid.UUID) ([]model.Challenge, error) {
	rows, err := s.db.Query(ctx, "select * from challenges where to_player = $1", userId)
	if err != nil {
		return []model.Challenge{}, err
	}

	data, err := pgx.CollectRows(rows, pgx.RowToStructByName[model.Challenge])
	if err != nil {
		return []model.Challenge{}, err
	}

	return data, nil
}

func (s *ChallengeStore) Create(ctx context.Context, request model.CreateChallenge) (model.Challenge, error) {
	rows, err := s.db.Query(
		ctx,
		`insert into challenges (from_player, to_player, clock_init_ms, clock_inc_ms)
		 select $1::uuid, $2::uuid, $3::int, $4::int
		 where (select count(*) from challenges where from_player = $1) < $5
		 returning *`,
		request.FromPlayer,
		request.ToPlayer,
		request.ClockInitial,
		request.ClockIncrement,
		maxChallengesPerUser,
	)

	if err != nil {
		return model.Challenge{}, err
	}

	challenge, err := pgx.CollectOneRow(rows, pgx.RowToStructByName[model.Challenge])
	if errors.Is(err, pgx.ErrNoRows) {
		return model.Challenge{}, ErrChallengeLimit
	}
	if err != nil {
		return model.Challenge{}, err
	}

	return challenge, nil
}

func (s *ChallengeStore) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := s.db.Exec(ctx, "delete from challenges where id = $1", id)
	return err
}

func (s *ChallengeStore) DeleteForUser(ctx context.Context, itemId, userId uuid.UUID) error {
	_, err := s.db.Exec(ctx, "delete from challenges where id = $2 and (from_player = $1 or to_player = $1)", userId, itemId)
	return err
}

func (s *ChallengeStore) DeleteOutgoing(ctx context.Context, userId uuid.UUID) error {
	_, err := s.db.Exec(ctx, "delete from challenges where from_player = $1", userId)
	return err
}

func (s *ChallengeStore) DeleteIncoming(ctx context.Context, userId uuid.UUID) error {
	_, err := s.db.Exec(ctx, "delete from challenges where to_player = $1", userId)
	return err
}
