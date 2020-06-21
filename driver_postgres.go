package migrate

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"
)

// PostgresDriver ...
type PostgresDriver struct {
	conn   *pgxpool.Pool
	tx     pgx.Tx
	schema string
	table  string
}

// NewPostgresDriver returns a new PostgresDriver instance.
// TODO: Config...
func NewPostgresDriver(conn *pgxpool.Pool, schema, table string) *PostgresDriver {
	return &PostgresDriver{
		conn:   conn,
		schema: schema,
		table:  table,
	}
}

// Begin ...
func (d *PostgresDriver) Begin(ctx context.Context) error {
	if d.tx != nil {
		return ErrTransactionAlreadyStarted
	}

	tx, err := d.conn.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to start transaction: %w", err)
	}

	d.tx = tx
	return nil
}

// Commit ...
func (d *PostgresDriver) Commit(ctx context.Context) error {
	if d.tx == nil {
		return ErrTransactionNotStarted
	}

	err := d.tx.Commit(ctx)
	if err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// Rollback ...
func (d *PostgresDriver) Rollback(ctx context.Context) error {
	if d.tx == nil {
		return ErrTransactionNotStarted
	}

	err := d.tx.Rollback(ctx)
	if err != nil {
		return fmt.Errorf("failed to rollback transaction: %w", err)
	}

	return nil
}

// Exec ...
func (d *PostgresDriver) Exec(ctx context.Context, command string) error {
	if d.tx == nil {
		return ErrTransactionNotStarted
	}

	_, err := d.tx.Exec(ctx, command)
	if err != nil {
		return fmt.Errorf("failed to execute command: %w", err)
	}

	return nil
}

// Lock ...
func (d *PostgresDriver) Lock(ctx context.Context) error {
	_, err := d.tx.Exec(ctx, fmt.Sprintf("LOCK TABLE %s.%s IN ACCESS EXCLUSIVE MODE", d.schema, d.table))
	if err != nil {
		return fmt.Errorf("failed to lock versions table: %w", err)
	}

	return nil
}

// CreateVersionsTable ...
func (d *PostgresDriver) CreateVersionsTable(ctx context.Context) error {
	// We use IF NOT EXISTS here because we're not doing this part in a transaction or with any sort
	// of lock. If the table already exists, then we can just skip creating it.
	query := fmt.Sprintf(`
		CREATE SCHEMA IF NOT EXISTS %[1]s;
		CREATE TABLE IF NOT EXISTS %[1]s.%[2]s (
			version int NOT NULL,
			migrated_at timestamp NOT NULL DEFAULT current_timestamp,

			PRIMARY KEY (version)
		);
	`, d.schema, d.table)

	_, err := d.conn.Exec(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to create versions table: %w", err)
	}

	return nil
}

// InsertVersion ...
func (d *PostgresDriver) InsertVersion(ctx context.Context, version int) error {
	query := fmt.Sprintf(`INSERT INTO %s.%s (version) VALUES ($1)`, d.schema, d.table)

	res, err := d.tx.Exec(ctx, query, version)
	if err != nil {
		return fmt.Errorf("failed to insert version: %w", err)
	}

	if res.RowsAffected() == 0 {
		return errors.New("expected new version row to be inserted, but no rows affected")
	}

	return nil
}

// Versions ...
func (d *PostgresDriver) Versions(ctx context.Context) ([]int, error) {
	query := fmt.Sprintf(`SELECT version FROM %s.%s`, d.schema, d.table)

	rows, err := d.tx.Query(ctx, query)
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
func (d *PostgresDriver) VersionTableExists(ctx context.Context) (bool, error) {
	var name sql.NullString

	query := fmt.Sprintf(`SELECT to_regclass('%s.%s')::text`, d.schema, d.table)

	err := d.conn.QueryRow(ctx, query).Scan(&name)
	if err != nil {
		return false, err
	}

	return name.Valid, nil
}
