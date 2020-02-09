package main

import (
	"context"
	"database/sql"
	"log"
	"time"

	"github.com/seeruk/go-migrate"
	"github.com/seeruk/go-migrate/postgres"

	_ "github.com/lib/pq"
)

func main() {
	conn, err := sql.Open("postgres", "user=postgres password=postgres sslmode=disable")
	if err != nil {
		log.Fatalf("failed to open DB connection: %v", err)
	}

	err = conn.Ping()
	if err != nil {
		log.Fatalf("failed to ping DB: %v", err)
	}

	ctx, cfn := context.WithTimeout(context.Background(), time.Hour)
	defer cfn()

	driver := postgres.NewDriver(conn, "migration_versions")

	err = migrate.Execute(ctx, driver, EventHandler{}, "example")
	if err != nil {
		log.Fatalf("failed to execute migrations: %v", err)
	}
}

func init() {
	// Register migrations under the "example" namespace. Migration versions are just numbers,
	// sorted so the lowest number is executed first.
	migrate.Register("example", 1, func(ctx context.Context, tx *sql.Tx) error {
		_, err := tx.ExecContext(ctx, `
			CREATE TABLE example (
			  id SERIAL NOT NULL,
			  
			  PRIMARY KEY (id)
			)
		`)

		return err
	})

	migrate.Register("example", 2, func(ctx context.Context, tx *sql.Tx) error {
		_, err := tx.ExecContext(ctx, `
			ALTER TABLE example ADD COLUMN created_at timestamp NOT NULL DEFAULT current_timestamp
		`)

		return err
	})
}

// EventHandler ...
type EventHandler struct{}

// BeforeVersionsMigrate ...
func (e EventHandler) BeforeVersionsMigrate(versions []int) {
	log.Printf("Found %d versions to migrate", len(versions))
}

// BeforeVersionMigrate ...
func (e EventHandler) BeforeVersionMigrate(version int) {
	log.Printf("Migrating version: %d...", version)
}

// AfterVersionsMigrate ...
func (e EventHandler) AfterVersionsMigrate(versions []int) {
	// No-op.
}

// AfterVersionMigrate ...
func (e EventHandler) AfterVersionMigrate(version int) {
	log.Printf("Migrated version: %d", version)
}

// OnVersionSkipped ...
func (e EventHandler) OnVersionSkipped(version int) {
	log.Printf("Skipping version: %d", version)
}

// OnVersionTableNotExists ...
func (e EventHandler) OnVersionTableNotExists() {
	log.Println("Versions table doesn't exist, creating...")
}

// OnVersionTableCreated ...
func (e EventHandler) OnVersionTableCreated() {
	log.Println("Created versions table")
}

// OnRollbackError ...
func (e EventHandler) OnRollbackError(err error) {
	log.Printf("Failed to rollback migration transaction: %v", err)
}
