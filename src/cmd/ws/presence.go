package main

import (
	"context"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/williamdann/chess-arena/internal/events"
	"github.com/williamdann/chess-arena/internal/model"
	"github.com/williamdann/chess-arena/internal/postgres"
	"github.com/williamdann/chess-arena/internal/pubsub"
)

// Each server heartbeats its own rows; a row that misses three beats blows
// past postgres.StaleAfterSecs and any server's reaper may collect it, so a
// crashed server's connections expire.
const heartbeatInterval = 30 * time.Second

// presence records this server's live sockets in the shared connections
// table, so the active player list can be computed across all servers.
type presence struct {
	server   uuid.UUID
	store    *postgres.PresenceStore
	profiles *postgres.ProfileStore
	ps       *pubsub.PubSub
}

func newPresence(store *postgres.PresenceStore, profiles *postgres.ProfileStore, ps *pubsub.PubSub) *presence {
	return &presence{server: uuid.New(), store: store, profiles: profiles, ps: ps}
}

// run blocks, keeping this server's rows fresh and reaping rows of dead
// servers; cancel ctx to stop.
func (p *presence) run(ctx context.Context) {
	tick := time.NewTicker(heartbeatInterval)
	defer tick.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-tick.C:
			if err := p.store.Heartbeat(ctx, p.server); err != nil {
				slog.Error("presence heartbeat failed", "err", err)
			}
			if err := p.store.DeleteAllStale(ctx, postgres.StaleAfterSecs); err != nil {
				slog.Error("presence reap failed", "err", err)
			}
		}
	}
}

// connected records a socket for user, announcing them if it's their first.
// Tracking is best effort: on error the socket still works, just untracked.
func (p *presence) connected(ctx context.Context, user uuid.UUID) *model.Connection {
	conn, err := p.store.Create(ctx, user, p.server)
	if err != nil {
		slog.Error("presence track failed", "user", user, "err", err)
		return nil
	}
	if conns, err := p.store.GetForUser(ctx, user); err == nil && len(conns) == 1 {
		p.announce(ctx, user, events.JoinPresenceEvent)
	}
	return &conn
}

// disconnected removes a tracked socket, announcing the leave when it was the
// user's last. The request context died with the socket, so it brings its own.
func (p *presence) disconnected(conn *model.Connection) {
	if conn == nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if _, err := p.store.Delete(ctx, conn.Id); err != nil {
		slog.Error("presence untrack failed", "conn", conn.Id, "err", err)
		return
	}
	if conns, err := p.store.GetForUser(ctx, conn.UserId); err == nil && len(conns) == 0 {
		p.announce(ctx, conn.UserId, events.LeavePresenceEvent)
	}
}

func (p *presence) announce(ctx context.Context, user uuid.UUID, event func(model.Profile) *events.PresenceEvent) {
	profile, err := p.profiles.GetProfile(ctx, user)
	if err != nil {
		slog.Error("presence profile lookup failed", "user", user, "err", err)
		return
	}
	if err := p.ps.Pub(ctx, event(profile)); err != nil {
		slog.Error("presence publish failed", "user", user, "err", err)
	}
}
