package main

import (
	"context"
	"log"
	"time"

	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/seeruk/go-migrate"
)

func main() {
	conn, err := pgxpool.Connect(context.TODO(), "user=postgres password=postgres sslmode=disable")
	if err != nil {
		log.Fatalf("failed to open DB connection: %v", err)
	}

	driver := migrate.NewPostgresDriver(conn, "example", "migration_versions")

	err = migrate.Execute(driver, NewEventHandler(), "example", time.Minute)
	if err != nil {
		log.Fatalf("failed to execute migrations: %v", err)
	}
}

func init() {
	// Register migrations under the "example" namespace. Migration versions are just numbers,
	// sorted so the lowest number is executed first. Missing migrations are applied, even if the
	// most recently applied migration has a higher version (e.g. to support working in branches).
	migrate.Register("example", migrate.NewMigration(1, `
		CREATE TABLE example (
		  id SERIAL NOT NULL,
		  
		  PRIMARY KEY (id)
		)
	`))

	// Migrations are code there's no need to package up migrations into your binary using a third-
	// party tool and an extra build step.
	migrate.Register("example", migrate.NewMigration(2, `
		ALTER TABLE example 
		ADD COLUMN created_at timestamp NOT NULL DEFAULT current_timestamp
	`))
}

// EventHandler ...
type EventHandler struct{}

// NewEventHandler ...
func NewEventHandler() EventHandler {
	return EventHandler{}
}

// BeforeVersionsMigrate ...
func (e EventHandler) BeforeVersionsMigrate(versions []int) {
	log.Printf("Found %d new versions to migrate", len(versions))
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

// OnExecuteError ...
func (e EventHandler) OnExecuteError(err error) {
	log.Printf("Failed to migrate: %v", err)
}

// OnRollbackError ...
func (e EventHandler) OnRollbackError(err error) {
	log.Printf("Failed to rollback migration transaction: %v", err)
}
