package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/williamdann/chess-arena/internal/postgres"
	"github.com/williamdann/chess-arena/internal/pubsub"
	"github.com/williamdann/chess-arena/internal/services"
)

// test
func main() {
	ctx := context.Background()

	pool, err := pgxpool.New(ctx, os.Getenv("DATABASE_URL"))
	if err != nil {
		slog.Error("invalid database config", "err", err)
		os.Exit(1)
	}
	defer pool.Close()

	if err := pool.Ping(ctx); err != nil {
		slog.Error("could not reach database", "err", err)
		os.Exit(1)
	}

	pubsub := pubsub.NewPubSub(pool)
	mux := http.NewServeMux()

	users := postgres.NewUserStore(pool)
	profiles := postgres.NewProfileStore(pool)
	sessions := postgres.NewSessionStore(pool)
	challenges := postgres.NewChallengeStore(pool)
	games := postgres.NewGameStore(pool)
	moves := postgres.NewMoveStore(pool)

	userService := services.NewUserService(users, profiles)
	sessionService := services.NewSessionService(users, sessions)
	profileService := services.NewProfileService(profiles, sessionService.RequireAuth)
	challengeService := services.NewChallengeService(challenges, games, pubsub, sessionService.RequireAuth)
	gameService := services.NewGameService(games, sessionService.RequireAuth)
	moveService := services.NewMoveService(moves)

	userService.Register(mux)
	sessionService.Register(mux)
	profileService.Register(mux)
	challengeService.Register(mux)
	gameService.Register(mux)
	moveService.Register(mux)

	slog.Info("starting http server on 0.0.0.0:8080")
	http.ListenAndServe("0.0.0.0:8080", mux)
}
