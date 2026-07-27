# Builds either the http or ws server, selected via the SERVICE build arg.
# The last stage is the production image (default target); the `dev` stage is
# used only by compose.override.yaml for live-reload local development.
FROM golang:1.26-alpine AS base
WORKDIR /app
ENV CGO_ENABLED=0

COPY go.* ./
RUN go mod download

FROM base AS build
COPY . .
ARG SERVICE=http
RUN go build -o /bin/server ./cmd/${SERVICE}

# Dev: air watches the bind-mounted source and rebuilds on change.
# Build output and air's scratch files go to /tmp so they never touch the host mount.
FROM base AS dev
RUN go install github.com/air-verse/air@latest
ARG SERVICE=http
ENV SERVICE=${SERVICE}
CMD air -tmp_dir /tmp/air -build.cmd "go build -o /tmp/server ./cmd/${SERVICE}" -build.bin /tmp/server

FROM alpine:3.22
COPY --from=build /bin/server /usr/local/bin/server
ENTRYPOINT ["server"]
