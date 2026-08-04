package pubsub

import (
	"context"
	"encoding/json"
	"log/slog"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

const pgPubSubChannel = "events"

type Event interface {
	Topic() string
	Payload() ([]byte, error)
}

type Message struct {
	Topic   string          `json:"topic"`
	Payload json.RawMessage `json:"payload"`
}

type PubSub struct {
	db *pgxpool.Pool

	mu   sync.Mutex
	subs map[string]map[chan Message]struct{}
}

func NewPubSub(db *pgxpool.Pool) *PubSub {
	return &PubSub{
		db:   db,
		subs: make(map[string]map[chan Message]struct{}),
	}
}

func (ps *PubSub) Pub(ctx context.Context, e Event) error {
	payload, err := e.Payload()
	if err != nil {
		return err
	}

	msg, err := json.Marshal(Message{Topic: e.Topic(), Payload: payload})
	if err != nil {
		return err
	}

	_, err = ps.db.Exec(ctx, "select pg_notify($1, $2)", pgPubSubChannel, string(msg))
	return err
}

// Sub subscribes to events for topic. The returned func unsubscribes and
// closes the channel; it is safe to call more than once.
func (ps *PubSub) Sub(topic string) (<-chan Message, func()) {
	ch := make(chan Message, 16)

	ps.mu.Lock()
	if ps.subs[topic] == nil {
		ps.subs[topic] = make(map[chan Message]struct{})
	}
	ps.subs[topic][ch] = struct{}{}
	ps.mu.Unlock()

	var once sync.Once
	unsub := func() {
		once.Do(func() {
			ps.mu.Lock()
			defer ps.mu.Unlock()
			delete(ps.subs[topic], ch)
			if len(ps.subs[topic]) == 0 {
				delete(ps.subs, topic)
			}
			close(ch)
		})
	}
	return ch, unsub
}

// Listen blocks, dispatching notifications to subscribers, and reconnects
// on connection loss. Run it in a goroutine; cancel ctx to stop.
func (ps *PubSub) Listen(ctx context.Context) error {
	for {
		err := ps.listen(ctx)
		if ctx.Err() != nil {
			return ctx.Err()
		}
		slog.Error("pubsub: listener lost, reconnecting", "err", err)

		select {
		case <-time.After(time.Second):
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

func (ps *PubSub) listen(ctx context.Context) error {
	pooled, err := ps.db.Acquire(ctx)
	if err != nil {
		return err
	}
	// LISTEN state sticks to the connection, so it must never go back
	// into the pool where another borrower would inherit it.
	conn := pooled.Hijack()
	defer conn.Close(context.Background())

	if _, err := conn.Exec(ctx, "listen "+pgPubSubChannel); err != nil {
		return err
	}

	for {
		n, err := conn.WaitForNotification(ctx)
		if err != nil {
			return err
		}
		var msg Message
		if err := json.Unmarshal([]byte(n.Payload), &msg); err != nil {
			slog.Error("pubsub: dropping undecodable event", "err", err)
			continue
		}
		ps.dispatch(msg)
	}
}

func (ps *PubSub) dispatch(msg Message) {
	ps.mu.Lock()
	defer ps.mu.Unlock()
	for ch := range ps.subs[msg.Topic] {
		select {
		case ch <- msg:
		default: // subscriber is full: drop, consumers resync from the DB
		}
	}
}
