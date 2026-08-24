FROM golang:1.23-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . ./
RUN go build -o /wera-api .

FROM alpine:3.20
WORKDIR /app
COPY --from=builder /wera-api .

# app.env is intentionally NOT copied: it holds local secrets and is gitignored
# (absent in CI). Under docker compose there is no app.env file -- configuration
# comes entirely from environment variables, which config.LoadConfig reads by
# binding each key and treating a missing app.env as non-fatal.
#
# The migrations are copied because the server applies them itself on startup;
# MIGRATION_URL defaults to file://db/migrations, relative to this WORKDIR.
COPY db/migrations ./db/migrations

EXPOSE 8080
CMD ["./wera-api"]
