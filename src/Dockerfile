# Builds either the http or ws server, selected via the SERVICE build arg.
FROM golang:1.26-alpine AS build
WORKDIR /app

COPY go.* ./
RUN go mod download

COPY . .
ARG SERVICE=http
RUN CGO_ENABLED=0 go build -o /bin/server ./cmd/${SERVICE}

FROM alpine:3.22
COPY --from=build /bin/server /usr/local/bin/server
ENTRYPOINT ["server"]
