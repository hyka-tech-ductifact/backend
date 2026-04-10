// Package migrations provides versioned SQL migrations embedded in the binary.
//
// Migrations are .sql files following the golang-migrate naming convention:
//
//	{version}_{description}.up.sql   — applies the change
//	{version}_{description}.down.sql — reverts the change
//
// The files are embedded via //go:embed so the binary is self-contained
// — no need to ship SQL files alongside the executable.
//
// Usage:
//
//	import "ductifact/internal/infrastructure/migrations"
//	migrations.Run(db) // applies all pending migrations
package migrations

import (
	"database/sql"
	"embed"
	"errors"
	"fmt"
	"log/slog"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/source/iofs"
)

// Embed all .sql files from this directory into the compiled binary.
// At build time, Go reads every .sql file in internal/infrastructure/migrations/
// and packs them into this variable. At runtime, `fs` acts as a virtual
// read-only filesystem — no disk access needed, no external files to ship.
//
//go:embed *.sql
var fs embed.FS

// Run applies all pending migrations (equivalent to `migrate up`).
// Returns nil if the database is already up to date.
//
// Dirty-state auto-recovery (PostgreSQL-specific):
// If a previous migration failed, golang-migrate marks the DB as "dirty"
// and refuses to run anything until you manually force the version.
// This safeguard exists because some databases (e.g. MySQL) don't support
// transactional DDL — a failed migration can leave the schema half-applied.
//
// PostgreSQL DOES support transactional DDL: if a migration fails, the
// entire transaction is rolled back and the schema stays at the previous
// clean version. So a "dirty" flag on Postgres simply means "it failed
// but nothing actually changed". We can safely reset to version N-1
// and let Up() retry from there.
func Run(db *sql.DB) error {
	src, err := iofs.New(fs, ".")
	if err != nil {
		return fmt.Errorf("migrations: failed to create source: %w", err)
	}

	driver, err := postgres.WithInstance(db, &postgres.Config{})
	if err != nil {
		return fmt.Errorf("migrations: failed to create driver: %w", err)
	}

	m, err := migrate.NewWithInstance("iofs", src, "postgres", driver)
	if err != nil {
		return fmt.Errorf("migrations: failed to create instance: %w", err)
	}

	// Auto-recover from dirty state: reset to the last clean version
	// so Up() can retry the failed migration.
	version, dirty, verr := m.Version()
	if verr == nil && dirty {
		slog.Warn("database is dirty, resetting to last clean version",
			"dirty_version", version,
			"clean_version", version-1,
		)
		if version == 1 {
			// No previous version exists — drop everything and start fresh.
			if err := m.Drop(); err != nil {
				return fmt.Errorf("migrations: failed to drop dirty v1: %w", err)
			}
			// Drop closes the source/driver, so we need a fresh instance.
			return Run(db)
		}
		m.Force(int(version - 1))
	}

	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("migrations: up failed: %w", err)
	}

	return nil
}
