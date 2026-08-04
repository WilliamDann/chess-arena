package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"

	"github.com/gorilla/websocket"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/williamdann/chess-arena/internal/pubsub"
	"github.com/williamdann/chess-arena/internal/services"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return r.Header.Get("Origin") == "http://localhost:8080"
	},
}

func handleConnect(w http.ResponseWriter, r *http.Request) {
	_, ok := services.SessionFromContext(r.Context())
	if !ok {
		http.Error(w, "invalid session", http.StatusUnauthorized)
		return
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		slog.Error("failed ws upgrade", "err", err)
		return
	}

	defer conn.Close()

	for {
		msgType, data, err := conn.ReadMessage()
		if err != nil {
			return
		}

		// echo for test
		conn.WriteMessage(msgType, data)
	}
}

func main() {
	// connect to database
	ctx := context.Background()
	pool, err := pgxpool.New(context.Background(), os.Getenv("DATABASE_URL"))
	if err != nil {
		slog.Error("invalid database config", "err", err)
		os.Exit(1)
	}
	defer pool.Close()

	if err := pool.Ping(ctx); err != nil {
		slog.Error("could not reach database", "err", err)
		os.Exit(1)
	}

	// start events listner
	pubsub := pubsub.NewPubSub(pool)
	go pubsub.Listen(ctx)

	// start websocket server
	mux := http.NewServeMux()
	mux.HandleFunc("GET /connect", handleConnect)

	slog.Info("started websocket server on 0.0.0.0:8081")
	http.ListenAndServe("0.0.0.0:8081", mux)
}
