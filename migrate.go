package migrate

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"sort"
)

var (
	// ErrTransactionAlreadyStarted ...
	ErrTransactionAlreadyStarted = errors.New("migrate: transaction already started")
	// ErrTransactionNotStarted ...
	ErrTransactionNotStarted = errors.New("migrate: transaction not started")
)

// namespacedMigrations contains all registered migrations, by namespace.
var namespacedMigrations = make(NamespacedMigrations)

// Migrations ...
type Migrations map[int]Migration

// NamespacedMigrations ...
type NamespacedMigrations map[string]Migrations

// Migration ...
type Migration func(ctx context.Context, tx *sql.Tx) error

// Register ...
// You can call this manually, or you can take advantage of `init` functions and just import a whole
// package of migrations at once. Sub-packages could easily be the namespace, e.g. migrations/users.
func Register(namespace string, version int, migration Migration) {
	if _, ok := namespacedMigrations[namespace]; !ok {
		namespacedMigrations[namespace] = make(Migrations)
	}

	namespacedMigrations[namespace][version] = migration
}

// Execute ...
// TODO: How should any sort of output be handled? Maybe some kind of configurable hooks, callbacks
// for when some events happen so you can use your own logger? Maybe you give a type that implements
// an interface that has all of the events as methods, and you choose how you want to output in
// there?
func Execute(ctx context.Context, driver Driver, eventHandler EventHandler, namespace string) (err error) {
	// Check if we can possibly have any work to do. If we don't, bail.
	migrationsByVersion, ok := namespacedMigrations[namespace]
	if !ok {
		return errors.New("no migrations found")
	}

	defer func() {
		// We always want to roll back the transaction if any error occurred, if we've started doing
		// some work. If we haven't started doing work, then we won't rollback. This just means we
		// don't have to handle rolling back all over the place.
		if err != nil {
			err := driver.Rollback()
			if err != nil && err != ErrTransactionNotStarted {
				eventHandler.OnRollbackError(err)
			}
		}
	}()

	// Before we can run migrations, lets check that the table exists?
	exists, err := driver.VersionTableExists(ctx)
	if err != nil {
		return fmt.Errorf("failed to check if versions table exists: %w", err)
	}

	if !exists {
		eventHandler.OnVersionTableNotExists()

		err := driver.CreateVersionsTable(ctx)
		if err != nil {
			return err
		}

		eventHandler.OnVersionTableCreated()
	}

	// TODO: Configurable transaction.
	// TODO: Configurable transaction options.
	tx, err := driver.Begin(ctx)
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

	eventHandler.BeforeVersionsMigrate(versions)

	for _, version := range versions {
		migration, ok := migrationsByVersion[version]
		if !ok {
			// This migration probably already existed, and was removed.
			eventHandler.OnVersionSkipped(version)
			continue
		}

		eventHandler.BeforeVersionMigrate(version)

		err = migration(ctx, tx)
		if err != nil {
			return fmt.Errorf("failed to execute migration: %w", err)
		}

		err = driver.InsertVersion(ctx, version)
		if err != nil {
			return fmt.Errorf("failed to insert version: %w", err)
		}

		eventHandler.AfterVersionMigrate(version)
	}

	eventHandler.AfterVersionsMigrate(versions)

	err = driver.Commit()
	if err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}
