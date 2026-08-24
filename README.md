# Wera Chap Chap

A full-stack on-demand task marketplace for Kenya, inspired by TaskRabbit. Clients post tasks, Taskers apply or get ranked through smart matching, bookings move through a lifecycle, users chat in real time, pay through an M-Pesa-ready payment intent flow, and leave reviews.

## Stack

- Backend: Go, Gin, GORM, PostgreSQL, JWT, Gorilla WebSocket
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

## Implemented Flows

- Auth: register, login, refresh token, logout
- Roles: client-only and tasker-only protected routes
- Tasks: post, browse, edit, cancel, apply, accept/reject applications
- Smart matching: category skill, availability, service radius proxy, rating and price scoring
- Bookings: confirm from accepted application, start, complete, cancel
- Messaging: REST history/send plus WebSocket chat per booking
- Reviews: completed-booking reviews and tasker rating recalculation
- Payments: M-Pesa-style intent placeholder with confirm and tip endpoints
- UI: landing page, auth, tasker directory/profile, client dashboard, tasker dashboard

## Notes

- SQL migrations live in `backend/db/migrations` and are mounted into Postgres on first container startup.
- The backend also runs `AutoMigrate` to keep local development forgiving.
- Payment routes expose an M-Pesa-ready intent abstraction while preserving the requested `stripe_payment_intent_id` database column name.
- File uploads are represented by `avatar_url` and S3-compatible environment variables; upload signing can be added behind the existing config.
