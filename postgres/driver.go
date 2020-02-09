package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/seeruk/go-migrate"
)

// Driver ...
type Driver struct {
	conn  *sql.DB
	tx    *sql.Tx
	table string
}

// NewDriver returns a new Driver instance.
// TODO: Config...
func NewDriver(conn *sql.DB, table string) *Driver {
	return &Driver{
		conn:  conn,
		table: table,
	}
}

// Begin ...
func (d *Driver) Begin(ctx context.Context) (*sql.Tx, error) {
	if d.tx != nil {
		return nil, migrate.ErrTransactionAlreadyStarted
	}

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

	err := d.tx.Commit()
	if err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// Rollback ...
func (d *Driver) Rollback() error {
	if d.tx != nil {
		return migrate.ErrTransactionNotStarted
	}

	err := d.tx.Rollback()
	if err != nil {
		return fmt.Errorf("failed to rollback transaction: %w", err)
	}

	return nil
}

// Lock ...
func (d *Driver) Lock(ctx context.Context) error {
	_, err := d.tx.ExecContext(ctx, fmt.Sprintf("LOCK TABLE %s IN ACCESS EXCLUSIVE MODE", d.table))
	if err != nil {
		return fmt.Errorf("failed to lock versions table: %w", err)
	}

	return nil
}

// CreateVersionsTable ...
func (d *Driver) CreateVersionsTable(ctx context.Context) error {
	// We use IF NOT EXISTS here because we're not doing this part in a transaction or with any sort
	// of lock. If the table already exists, then we can just skip creating it.
	query := fmt.Sprintf(`
		CREATE TABLE IF NOT EXISTS %s (
			version int NOT NULL,
			migrated_at timestamp NOT NULL DEFAULT current_timestamp,

			PRIMARY KEY (version)
		);
	`, d.table)

	_, err := d.conn.ExecContext(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to create versions table: %w", err)
	}

	return nil
}

// InsertVersion ...
func (d *Driver) InsertVersion(ctx context.Context, version int) error {
	query := fmt.Sprintf(`INSERT INTO %s (version) VALUES ($1)`, d.table)

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
	query := fmt.Sprintf(`SELECT version FROM %s`, d.table)

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
	var name sql.NullString

	query := fmt.Sprintf(`SELECT to_regclass('%s')`, d.table)

	err := d.conn.QueryRowContext(ctx, query).Scan(&name)
	if err != nil {
		return false, err
	}

	return name.Valid, nil
}
