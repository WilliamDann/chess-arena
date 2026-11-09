# Chess Arena

A multiplayer chess application with a Go backend, a React/TypeScript frontend, and PostgreSQL.

## Features

- Account registration and cookie-based sign-in.
- A live lobby with online players, open challenges, and direct player challenges.
- Real-time games with server-side move validation, configurable clocks and increments, resignation, and timeout claims.
- Game spectating, move history, and player profiles with game history.

## Local setup

### Requirements

- Docker with the Docker Compose plugin.
- Node.js 22.12+ and npm for the frontend.
- Ports `8080`, `8081`, and `5432` available for the backend services.

### 1. Start the backend

From the repository root:

```sh
cp .env.example .env
docker compose up --build
```

Compose starts the HTTP API, WebSocket server, and PostgreSQL. It automatically loads `compose.override.yaml`, which bind-mounts the Go source and runs both Go services with [Air](https://github.com/air-verse/air) for live reload. Go is not required on the host for this setup.

### 2. Start the frontend

In a separate terminal:

```sh
cd web
npm ci
npm run dev
```

Open the URL printed by Vite (normally <http://localhost:5173>) and create an account. There are no seeded accounts. To try a game locally, use a second account in another browser or a private window.

The frontend runs separately from Compose. Vite proxies `/api` to the HTTP service and `/ws` to the WebSocket service, preserving cookie-based authentication. Port `8080` serves the API, not the web interface.

## Configuration

Docker Compose reads `.env` automatically. The example values are for local development only.

| Variable | Default in `.env.example` | Purpose |
| --- | --- | --- |
| `POSTGRES_USER` | `chess` | Database user. |
| `POSTGRES_PASSWORD` | `chess` | Database password; change it outside local development. |
| `POSTGRES_DB` | `chess_arena` | Database name. |
| `ALLOWED_ORIGIN` | `http://localhost:8080` | Exact origin accepted by the WebSocket server. |
| `SECURE_COOKIES` | `false` | Set to `true` when serving over HTTPS. Keep `false` for local HTTP. |

Compose constructs `DATABASE_URL` for both Go services from the PostgreSQL variables. If running the Go services directly, provide `DATABASE_URL` yourself, along with the relevant settings above.

For local development, keep the default `ALLOWED_ORIGIN`: the Vite proxy explicitly sends `Origin: http://localhost:8080` for WebSocket upgrades. If you change it, update the proxy header in `web/vite.config.ts` too.

## Architecture

| Component | Port | Responsibility |
| --- | --- | --- |
| HTTP service | `8080` | JSON API under `/api` for accounts, sessions, profiles, challenges, games, moves, and presence. |
| WebSocket service | `8081` | Authenticated `/lobby` and `/game/{id}` connections for live events and gameplay. |
| PostgreSQL 17 | `5432` | Persistent storage and cross-service events through `LISTEN`/`NOTIFY`. |
| Vite frontend | `5173` by default | React UI using Chessground and chess.js. |

The HTTP and WebSocket services share database-backed sessions. Game moves are validated on the server with `github.com/notnil/chess` and persisted in PostgreSQL.

```text
src/
  cmd/http/         HTTP server entry point
  cmd/ws/           WebSocket server, gameplay, and presence
  internal/
    services/       HTTP handlers and authentication
    postgres/       Database access
    model/          Data models
    events/         Event definitions
    pubsub/         PostgreSQL-backed publish/subscribe
db/
  migrations/       Schema initialization SQL
  scripts/          Seed script (currently empty)
web/                React, TypeScript, and Vite frontend
```

## Development commands

View backend logs or stop the stack from the repository root:

```sh
docker compose logs -f http ws
docker compose down
```

Database data persists in the `db-data` volume when the stack stops. SQL files in `db/migrations/` run only when PostgreSQL initializes an empty data directory; editing them does not update an existing database automatically.

Frontend checks, run from `web/`:

```sh
npm run lint
npm run build
```

The frontend build is written to `web/dist/`.

## Running without live reload

To build and run the backend runtime images without the development override:

```sh
docker compose -f compose.yaml up --build
```

This still starts only the backend and database. A deployment also needs to serve the built frontend and proxy API and WebSocket traffic under the same origin. Use HTTPS, set `SECURE_COOKIES=true`, set `ALLOWED_ORIGIN` to the site's exact origin, and replace the example database credentials before deploying.
