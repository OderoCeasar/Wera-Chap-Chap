package main

import (
	"context"
	"log"

	"github.com/jackc/pgx/v5/pgxpool"

	"wera-chap-chap/backend/api"
	"wera-chap-chap/backend/config"
	db "wera-chap-chap/backend/db/sqlc"
	"wera-chap-chap/backend/db_migrator"
	"wera-chap-chap/backend/seed"
	chat "wera-chap-chap/backend/websocket"
)

// main is the composition root: it loads configuration, opens the pool, brings
// the schema up to date, builds the store and the server, and starts listening.
// Nothing else in the program constructs its own dependencies, so the wiring is
// readable in one place and a test can substitute any piece of it.
func main() {
	ctx := context.Background()

	cfg, err := config.LoadConfig(".")
	if err != nil {
		log.Fatalf("cannot load config: %v", err)
	}

	// The JWT secret is symmetric: whoever holds it can mint a token for any
	// user id and role. The compose default and the .env.example placeholder
	// are both public -- they live in this repo -- so booting a real
	// environment with either means anyone who read the source can forge a
	// session. Fail fast rather than serve forgeable tokens.
	if !cfg.IsLocalEnv() && cfg.HasInsecureJWTSecret() {
		log.Fatal("JWT_SECRET is a known public/default value; set a unique secret before starting in this environment")
	}

	connPool, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("cannot connect to database: %v", err)
	}
	defer connPool.Close()

	if err := db_migrator.RunDBMigration(cfg); err != nil {
		log.Fatalf("cannot run db migration: %v", err)
	}

	store := db.NewStore(connPool)
	seed.Run(ctx, store, cfg)

	hub := chat.NewHub()
	go hub.Run()

	server, err := api.NewServer(cfg, store, hub)
	if err != nil {
		log.Fatalf("cannot create server: %v", err)
	}

	log.Printf("Wera Chap Chap backend listening on %s", cfg.Addr())
	if err := server.Start(cfg.Addr()); err != nil {
		log.Fatalf("cannot start server: %v", err)
	}
}
