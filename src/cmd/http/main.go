package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/williamdann/chess-arena/internal/postgres"
	services "github.com/williamdann/chess-arena/internal/services"
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

	mux := http.NewServeMux()

	users := postgres.NewUserStore(pool)

	userService := services.NewUserService(users)
	userService.Register(mux)

	slog.Info("starting http server on 0.0.0.0:8080")
	http.ListenAndServe("0.0.0.0:8080", mux)
}
