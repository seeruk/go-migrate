package migrate

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"time"
)

var (
	// ErrTransactionAlreadyStarted ...
	ErrTransactionAlreadyStarted = errors.New("migrate: transaction already started")
	// ErrTransactionNotStarted ...
	ErrTransactionNotStarted = errors.New("migrate: transaction not started")
)

// namespacedMigrations contains all registered migrations, by namespace.
var namespacedMigrations = make(NamespacedMigrations)

// Migration ...
type Migration struct {
	Version  int
	Commands []string
}

// NewMigration returns a new Migration value.
func NewMigration(version int, commands ...string) Migration {
	return Migration{
		Version:  version,
		Commands: commands,
	}
}

// Migrations ...
type Migrations map[int]Migration

// NamespacedMigrations ...
type NamespacedMigrations map[string]Migrations

// Register ...
// You can call this manually, or you can take advantage of `init` functions and just import a whole
// package of migrations at once. Sub-packages could easily be the namespace, e.g. migrations/users.
func Register(namespace string, migration Migration) {
	if _, ok := namespacedMigrations[namespace]; !ok {
		namespacedMigrations[namespace] = make(Migrations)
	}

	namespacedMigrations[namespace][migration.Version] = migration
}

// Execute ...
func Execute(driver Driver, events EventHandler, namespace string, timeout time.Duration) (err error) {
	ctx, cfn := context.WithTimeout(context.Background(), timeout)
	defer cfn()

	// Check if we can possibly have any work to do. If we don't, bail.
	migrationsByVersion, ok := namespacedMigrations[namespace]
	if !ok {
		return nil
	}

	defer func() {
		// We always want to roll back the transaction if any error occurred, if we've started doing
		// some work. If we haven't started doing work, then we won't rollback. This just means we
		// don't have to handle rolling back all over the place.
		if err != nil {
			rerr := driver.Rollback(ctx)
			if rerr != nil && rerr != ErrTransactionNotStarted {
				events.OnRollbackError(rerr)
			}

			events.OnExecuteError(err)
		}
	}()

	// Before we can run migrations, lets check that the table exists?
	exists, err := driver.VersionTableExists(ctx)
	if err != nil {
		return fmt.Errorf("failed to check if versions table exists: %w", err)
	}

	if !exists {
		events.OnVersionTableNotExists()

		err := driver.CreateVersionsTable(ctx)
		if err != nil {
			return err
		}

		events.OnVersionTableCreated()
	}

	err = driver.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	// Lock outside migrations. We want to lock before seeing what versions already exist so that we
	// can be certain about the versions we are yet to insert.
	err = driver.Lock(ctx)
	if err != nil {
		return fmt.Errorf("failed to lock versions table: %w", err)
	}

	existingVersions, err := driver.Versions(ctx)
	if err != nil {
		return fmt.Errorf("failed to get current versions: %w", err)
	}

	for _, version := range existingVersions {
		if _, ok := migrationsByVersion[version]; ok {
			delete(migrationsByVersion, version)
		}
	}

	var versions []int
	for version := range migrationsByVersion {
		versions = append(versions, version)
	}

	sort.Ints(versions)

	events.BeforeVersionsMigrate(versions)

	for _, version := range versions {
		migration, ok := migrationsByVersion[version]
		if !ok {
			// This migration probably already existed, and was removed.
			events.OnVersionSkipped(version)
			continue
		}

		events.BeforeVersionMigrate(version)

		for i, command := range migration.Commands {
			err = driver.Exec(ctx, command)
			if err != nil {
				return fmt.Errorf("failed to execute migration (command %d): %w", i, err)
			}
		}

		err = driver.InsertVersion(ctx, version)
		if err != nil {
			return fmt.Errorf("failed to insert version: %w", err)
		}

		events.AfterVersionMigrate(version)
	}

	events.AfterVersionsMigrate(versions)

	err = driver.Commit(ctx)
	if err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}
