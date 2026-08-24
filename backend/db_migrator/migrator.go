package db_migrator

import (
	"errors"
	"log"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	_ "github.com/jackc/pgx/v5/stdlib"

	"wera-chap-chap/backend/config"
)

// RunDBMigration applies every pending migration in db/migrations.
//
// This replaces the previous GORM AutoMigrate plus the SQL files that Postgres
// ran from docker-entrypoint-initdb.d. Both were invisible to version control
// in the sense that mattered: initdb scripts run only on an empty volume, and
// AutoMigrate never drops or alters anything it did not add. golang-migrate
// tracks what has been applied in a schema_migrations table, so the schema a
// given commit expects is the schema it gets.
func RunDBMigration(cfg config.Config) error {
	migration, err := migrate.New(cfg.MigrationURL, cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer migration.Close()

	if err := migration.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return err
	}

	log.Println("db migrated successfully")
	return nil
}
