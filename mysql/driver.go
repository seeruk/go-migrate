package mysql

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/seeruk/go-migrate"
)

// Driver ...
type Driver struct {
	conn     *sql.DB
	tx       *sql.Tx
	database string
	table    string
}

// NewDriver returns a new Driver instance.
func NewDriver(conn *sql.DB, database, table string) *Driver {
	return &Driver{
		conn:     conn,
		database: database,
		table:    table,
	}
}

// Begin ...
func (d *Driver) Begin(ctx context.Context) (*sql.Tx, error) {
	if d.tx != nil {
		return nil, migrate.ErrTransactionAlreadyStarted
	}

	// TODO: Is this the same for every driver?.. Maybe we could move this out of the driver.
	tx, err := d.conn.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to start transaction: %w", err)
	}

	d.tx = tx
	return tx, nil
}

// Commit ...
func (d *Driver) Commit() error {
	if d.tx == nil {
		return migrate.ErrTransactionNotStarted
	}

	defer d.Unlock()

	err := d.tx.Commit()
	if err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// Rollback ...
func (d *Driver) Rollback() error {
	if d.tx == nil {
		return migrate.ErrTransactionNotStarted
	}

	defer d.Unlock()

	err := d.tx.Rollback()
	if err != nil {
		return fmt.Errorf("failed to rollback transaction: %w", err)
	}

	return nil
}

// Lock ...
func (d *Driver) Lock(ctx context.Context) error {
	lock := fmt.Sprintf("migrate_%s_%s", d.database, d.table)

	// TODO: Ideally there would be a timeout, and we'd keep retrying the acquire.
	_, err := d.tx.ExecContext(ctx, fmt.Sprintf(`SELECT GET_LOCK("%s", -1)`, lock))
	if err != nil {
		return fmt.Errorf("failed to acquire named lock: %s: %w", lock, err)
	}

	return nil
}

// Unlock must be explicitly implemented for MySQL.
func (d *Driver) Unlock() {
	ctx, cfn := context.WithTimeout(context.Background(), 30*time.Second)
	defer cfn()

	lock := fmt.Sprintf("migrate_%s_%s", d.database, d.table)

	_, err := d.conn.ExecContext(ctx, fmt.Sprintf(`SELECT RELEASE_LOCK("%s")`, lock))
	if err != nil {
		log.Println("migrate/mysql: failed to explicitly unlock: %v", err)
	}
}

// CreateVersionsTable ...
func (d *Driver) CreateVersionsTable(ctx context.Context) error {
	dbq := fmt.Sprintf(`CREATE DATABASE IF NOT EXISTS %s DEFAULT CHARACTER SET utf8mb4`, d.database)
	tbq := fmt.Sprintf(`
		CREATE TABLE IF NOT EXISTS %s.%s (
			version int NOT NULL,
			migrated_at timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,

			PRIMARY KEY (version)
		) ENGINE=InnoDB DEFAULT CHARACTER SET=utf8mb4
	`, d.database, d.table)

	_, err := d.conn.ExecContext(ctx, dbq)
	if err != nil {
		return fmt.Errorf("failed to create versions database: %w", err)
	}

	_, err = d.conn.ExecContext(ctx, tbq)
	if err != nil {
		return fmt.Errorf("failed to create versions table: %w", err)
	}

	return nil
}

// InsertVersion ...
func (d *Driver) InsertVersion(ctx context.Context, version int) error {
	query := fmt.Sprintf(`INSERT INTO %s.%s (version) VALUES (?)`, d.database, d.table)

	res, err := d.tx.ExecContext(ctx, query, version)
	if err != nil {
		return fmt.Errorf("failed to insert version: %w", err)
	}

	ra, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected by insert version: %w", err)
	}

	if ra == 0 {
		return errors.New("expected new version row to be inserted, but no rows affected")
	}

	return nil
}

// Versions ...
func (d *Driver) Versions(ctx context.Context) ([]int, error) {
	query := fmt.Sprintf(`SELECT version FROM %s.%s`, d.database, d.table)

	rows, err := d.tx.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query current versions: %w", err)
	}

	defer rows.Close()

	var versions []int
	for rows.Next() {
		var version int

		err := rows.Scan(&version)
		if err != nil {
			return nil, fmt.Errorf("failed to scan current version: %w", err)
		}

		versions = append(versions, version)
	}

	return versions, nil
}

// VersionTableExists ...
func (d *Driver) VersionTableExists(ctx context.Context) (bool, error) {
	var count int

	query := `
		SELECT COUNT(1) 
		FROM information_schema.tables 
		WHERE table_schema = ? 
		AND table_name = ?
	`

	err := d.conn.QueryRowContext(ctx, query, d.database, d.table).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("failed to check if version table exists: %w", err)
	}

	return count == 1, nil
}
