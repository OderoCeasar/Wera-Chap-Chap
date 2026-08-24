# Wera Chap Chap

A full-stack on-demand task marketplace for Kenya, inspired by TaskRabbit. Clients post tasks, Taskers apply or get ranked through smart matching, bookings move through a lifecycle, users chat in real time, pay through an M-Pesa-ready payment intent flow, and leave reviews.

## Stack

- Backend: Go, Gin, sqlc over pgx/v5, PostgreSQL, golang-migrate, JWT, Gorilla WebSocket
- Frontend: React, Vite, Tailwind CSS
- Dev environment: Docker Compose with Postgres, backend hot reload via Air, and Vite

## Quick Start

```bash
cp .env.example .env
docker compose up --build
```

- Frontend: `http://localhost:5173`
- Backend health: `http://localhost:8080/api/health`
- Postgres: `localhost:5432`

The backend applies its own migrations on startup, so an empty database is all
it needs. If you are coming from a checkout that predates the sqlc restructure,
reset the volume once — the old schema was created by `AutoMigrate` and by
scripts Postgres ran from `docker-entrypoint-initdb.d`, neither of which
recorded a migration version:

```bash
docker compose down -v && docker compose up --build
```

## Backend layout

Package-per-concern at the root of `backend/`, with `main.go` as the composition
root and nothing else constructing its own dependencies.

| Path | What lives there |
| --- | --- |
| `main.go` | Wiring only: config → pool → migrations → store → seed → hub → server |
| `config/` | One `Config` struct, viper + `mapstructure`, reads `app.env` or the environment |
| `db/migrations/` | Numbered `.up.sql`/`.down.sql` pairs; the source of truth for the schema |
| `db/queries/` | Hand-written SQL, one file per table |
| `db/sqlc/` | Generated code, plus `store.go`, `exec_txn.go`, `errors.go` and the `tx_*.go` transactions |
| `db_migrator/` | Runs golang-migrate at startup |
| `api/` | Gin. `server.go`, `routes.go`, `auth_interceptor.go`, `payloads.go`, one file per feature |
| `token/` | `Maker` interface with a JWT implementation |
| `matching/` | Tasker ranking: `repository.go` loads, `service.go` scores |
| `seed/` | Opt-in demo data (`SEED_DEMO`) |
| `websocket/` | Chat hub |
| `utils/` | Password hashing |

`db/sqlc/store.go` defines the `Store` interface that everything above the data
layer depends on. It embeds the generated `Querier` and adds the multi-statement
transactions, so the set of writes that must be atomic is visible in one place.

### Working on it

```bash
cd backend
make sqlc                          # regenerate db/sqlc after editing db/queries
make new_migration name=add_thing  # scaffold a migration pair
make test
make run
```

Editing a query means editing `db/queries/*.sql` and running `make sqlc`; never
edit `db/sqlc/*.sql.go` by hand. Changing the schema means a new migration pair —
sqlc reads `db/migrations` to know the column types, so the two stay in step.

`backend/requests.http` walks the whole API end to end, capturing tokens and ids
as it goes.

## Implemented Flows

- Auth: register, login, refresh token, logout
- Roles: client-only and tasker-only protected routes
- Tasks: post, browse, edit, cancel, apply, accept/reject applications
- Smart matching: category skill, availability, service radius proxy, rating and price scoring
- Bookings: confirm from accepted application, start, complete, cancel
- Messaging: REST history/send plus WebSocket chat per booking
- Reviews: completed-booking reviews and tasker rating recalculation
- Payments: M-Pesa-style intent placeholder with confirm, tip and callback endpoints
- UI: landing page, auth, tasker directory/profile, client dashboard, tasker dashboard

## Notes

- Response bodies are assembled in `api/payloads.go` rather than serialised
  straight from the generated row types. That is deliberate: sqlc gives
  `db.User` a `password_hash` JSON tag, so returning one directly would put
  every bcrypt hash on the wire.
- Payment routes expose an M-Pesa-ready intent abstraction while preserving the
  `stripe_payment_intent_id` column name.
- File uploads are represented by `avatar_url` and S3-compatible environment
  variables; upload signing can be added behind the existing config.
